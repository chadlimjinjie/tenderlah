package main

import (
	"encoding/json"
	"os"
	"strings"
	"unicode"

	unorm "golang.org/x/text/unicode/norm"
)

// Tokenizer reimplements HuggingFace BertNormalizer + BertPreTokenizer +
// WordPiece (with add_special_tokens=false) so token ids match model2vec's
// own tokenization exactly. We do NOT depend on the HF tokenizers library;
// for a static model this is ~150 lines and keeps a pure static binary.
type Tokenizer struct {
	vocab    map[string]int
	unkID    int
	maxChars int // max_input_chars_per_word; longer words become [UNK]
}

func loadTokenizer(path string) (*Tokenizer, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tj struct {
		Model struct {
			Vocab                map[string]int `json:"vocab"`
			UnkToken             string         `json:"unk_token"`
			MaxInputCharsPerWord int            `json:"max_input_chars_per_word"`
		} `json:"model"`
	}
	if err := json.Unmarshal(b, &tj); err != nil {
		return nil, err
	}
	unk, ok := tj.Model.Vocab[tj.Model.UnkToken]
	if !ok {
		unk = 1 // [UNK] convention
	}
	maxChars := tj.Model.MaxInputCharsPerWord
	if maxChars == 0 {
		maxChars = 100
	}
	return &Tokenizer{vocab: tj.Model.Vocab, unkID: unk, maxChars: maxChars}, nil
}

// Encode returns the model2vec token ids for text (no special tokens).
func (t *Tokenizer) Encode(text string) []int {
	words := preTokenize(t.normalize(text))
	ids := make([]int, 0, len(words))
	for _, w := range words {
		ids = t.wordpiece(w, ids)
	}
	return ids
}

// normalize applies BertNormalizer: clean_text, handle_chinese_chars,
// strip_accents (NFD + drop nonspacing marks), then lowercase.
func (t *Tokenizer) normalize(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == 0 || r == 0xFFFD || isControl(r):
			// dropped
		case isWhitespace(r):
			b.WriteByte(' ')
		case isChineseChar(r):
			b.WriteByte(' ')
			b.WriteRune(r)
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	decomposed := unorm.NFD.String(b.String())
	var b2 strings.Builder
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) { // nonspacing combining mark
			continue
		}
		b2.WriteRune(r)
	}
	return strings.ToLower(b2.String())
}

// preTokenize applies BertPreTokenizer: split on whitespace, isolate each
// punctuation char as its own token.
func preTokenize(s string) []string {
	var words []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		switch {
		case isWhitespace(r):
			flush()
		case isPunctuation(r):
			flush()
			words = append(words, string(r))
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return words
}

// wordpiece greedily splits a word into the longest matching vocab pieces
// (subword pieces after the first carry the "##" prefix). A word with any
// unmatchable piece maps wholly to [UNK], per the reference algorithm.
func (t *Tokenizer) wordpiece(word string, ids []int) []int {
	runes := []rune(word)
	if len(runes) > t.maxChars {
		return append(ids, t.unkID)
	}
	var sub []int
	for start := 0; start < len(runes); {
		end := len(runes)
		curID := -1
		for end > start {
			piece := string(runes[start:end])
			if start > 0 {
				piece = "##" + piece
			}
			if id, ok := t.vocab[piece]; ok {
				curID = id
				break
			}
			end--
		}
		if curID == -1 {
			return append(ids, t.unkID) // whole word unknown
		}
		sub = append(sub, curID)
		start = end
	}
	return append(ids, sub...)
}

func isWhitespace(r rune) bool {
	if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
		return true
	}
	return unicode.Is(unicode.Zs, r)
}

func isControl(r rune) bool {
	if r == '\t' || r == '\n' || r == '\r' {
		return false
	}
	return unicode.In(r, unicode.Cc, unicode.Cf, unicode.Co, unicode.Cs)
}

func isPunctuation(r rune) bool {
	// BERT treats all ASCII non-alphanumeric punctuation ranges as punctuation
	// (this includes symbols like '+' '<' '=' that Unicode classes as symbols).
	if (r >= 33 && r <= 47) || (r >= 58 && r <= 64) || (r >= 91 && r <= 96) || (r >= 123 && r <= 126) {
		return true
	}
	return unicode.IsPunct(r)
}

func isChineseChar(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || (r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0x20000 && r <= 0x2A6DF) || (r >= 0x2A700 && r <= 0x2B73F) ||
		(r >= 0x2B740 && r <= 0x2B81F) || (r >= 0x2B820 && r <= 0x2CEAF) ||
		(r >= 0xF900 && r <= 0xFAFF) || (r >= 0x2F800 && r <= 0x2FA1F)
}
