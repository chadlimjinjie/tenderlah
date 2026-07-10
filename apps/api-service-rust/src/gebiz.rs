//! Scrapes a GeBIZ opportunity detail page (the tender's `link`) into
//! structured contact + line-item data. The page is server-rendered JSF, so a
//! plain HTTP GET is enough — values sit in stable CSS classes / icon images.
//! Port of internal/gebizdetail/parse.go.
//!
//! Class-name scrape, brittle to GeBIZ redesigns — swap to their API if one appears.

use ego_tree::NodeRef;
use regex::Regex;
use scraper::{Html, Node};
use serde::{Serialize, Serializer};
use std::sync::OnceLock;

/// Serializes an empty Vec as null, matching Go's nil-slice JSON (the stored
/// details jsonb must stay byte-compatible with what the Go service wrote).
fn nil_when_empty<T: Serialize, S: Serializer>(v: &Vec<T>, s: S) -> Result<S::Ok, S::Error> {
    if v.is_empty() {
        s.serialize_none()
    } else {
        v.serialize(s)
    }
}

#[derive(Debug, Default, Serialize)]
pub struct Attr {
    pub label: String,
    pub value: String,
}

#[derive(Debug, Default, Serialize)]
pub struct Item {
    pub no: String,
    pub description: String,
    #[serde(serialize_with = "nil_when_empty")]
    pub attributes: Vec<Attr>,
}

#[derive(Debug, Default, Serialize)]
pub struct Contact {
    pub name: String,
    pub email: String,
    pub phone: String,
    pub address: String,
}

#[derive(Debug, Default, Serialize)]
pub struct Details {
    pub contact: Contact,
    #[serde(serialize_with = "nil_when_empty")]
    pub items: Vec<Item>,
}

fn item_no_re() -> &'static Regex {
    static RE: OnceLock<Regex> = OnceLock::new();
    RE.get_or_init(|| Regex::new(r"Item No\.\s*(\d+)").unwrap())
}

fn email_re() -> &'static Regex {
    static RE: OnceLock<Regex> = OnceLock::new();
    RE.get_or_init(|| Regex::new(r"[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}").unwrap())
}

/// Downloads the opportunity page (3MB read cap, custom UA) and parses it.
pub async fn fetch(url: &str) -> Result<Details, String> {
    let resp = reqwest::Client::new()
        .get(url)
        .header(reqwest::header::USER_AGENT, "Mozilla/5.0 (compatible; MidtoneTenderBot/1.0)")
        .send()
        .await
        .map_err(|e| format!("gebizdetail: fetch {url}: {e}"))?;

    let status = resp.status();
    if status.as_u16() >= 400 {
        return Err(format!("gebizdetail: fetch {url}: status {}", status.as_u16()));
    }

    // Read at most 3MB, like Go's io.LimitReader.
    let mut body: Vec<u8> = Vec::new();
    let mut resp = resp;
    while let Some(chunk) = resp.chunk().await.map_err(|e| e.to_string())? {
        let remaining = (3 << 20) - body.len();
        if remaining == 0 {
            break;
        }
        body.extend_from_slice(&chunk[..chunk.len().min(remaining)]);
    }
    Ok(parse(&body))
}

#[derive(Clone, Copy, PartialEq)]
enum TokKind {
    Section,
    Orange,
    Title,
    Label,
    Value,
    Icon,
}

struct Tok {
    kind: TokKind,
    text: String,
}

/// Turns the page HTML into Details.
pub fn parse(body: &[u8]) -> Details {
    let html = String::from_utf8_lossy(body);
    let doc = Html::parse_document(&html);

    let mut toks: Vec<Tok> = Vec::new();
    walk(doc.tree.root(), &mut toks);

    let mut d = Details {
        contact: parse_contact(&toks),
        items: parse_items(&toks),
    };
    if d.contact.email.is_empty() {
        if let Some(m) = email_re().find(&html) {
            d.contact.email = m.as_str().to_string();
        }
    }
    d
}

/// Depth-first walk emitting tokens for the recognised class/style markers.
/// A matched element is not descended into (its text is captured whole), same
/// as the Go walk's early returns.
fn walk(node: NodeRef<'_, Node>, toks: &mut Vec<Tok>) {
    if let Node::Element(el) = node.value() {
        let class = el.attr("class").unwrap_or("");
        let style = el.attr("style").unwrap_or("");
        if el.name() == "label" {
            toks.push(Tok { kind: TokKind::Label, text: text_of(node) });
            return;
        } else if class.contains("formSectionHeader") {
            toks.push(Tok { kind: TokKind::Section, text: text_of(node) });
            return;
        } else if class.contains("outputText_LABEL-ORANGE") {
            toks.push(Tok { kind: TokKind::Orange, text: text_of(node) });
            return;
        } else if class.contains("outputText_TITLE-BLACK") {
            toks.push(Tok { kind: TokKind::Title, text: text_of(node) });
            return;
        } else if class.contains("formOutputText_VALUE-DIV") {
            toks.push(Tok { kind: TokKind::Value, text: text_of(node) });
            return;
        } else if style.contains("imageicons") {
            toks.push(Tok { kind: TokKind::Icon, text: nearest_text(node) });
            return;
        }
    }
    for child in node.children() {
        walk(child, toks);
    }
}

/// Reads the tokens between the "WHO TO CONTACT" section header and the next
/// section header: the first title is the name; icon values classify into
/// email / phone / address.
fn parse_contact(toks: &[Tok]) -> Contact {
    let start = match section_index(toks, "WHO TO CONTACT").or_else(|| section_index(toks, "CONTACT")) {
        Some(i) => i,
        None => return Contact::default(),
    };
    let mut c = Contact::default();
    let mut phones: Vec<String> = Vec::new();
    for t in &toks[start + 1..] {
        if t.kind == TokKind::Section {
            break;
        }
        match t.kind {
            TokKind::Title => {
                if c.name.is_empty() {
                    c.name = t.text.clone();
                }
            }
            TokKind::Icon => {
                let v = t.text.trim();
                if v.is_empty() {
                } else if v.contains('@') {
                    if c.email.is_empty() {
                        c.email = v.to_string();
                    }
                } else if is_phone(v) {
                    phones.push(v.to_string());
                } else if c.address.is_empty() {
                    c.address = v.to_string();
                }
            }
            _ => {}
        }
    }
    c.phone = phones.join(", ");
    c
}

/// Reads everything after the "ITEMS TO RESPOND" header. Each "Item No. N"
/// label starts a new item; the following title is its description;
/// label/value pairs (including the PART NO. / DELIVERY sub-sections) become
/// attributes.
fn parse_items(toks: &[Tok]) -> Vec<Item> {
    let start = match section_index(toks, "ITEMS TO RESPOND") {
        Some(i) => i,
        None => return Vec::new(),
    };
    let mut items: Vec<Item> = Vec::new();
    let mut pending = String::new();
    for t in &toks[start + 1..] {
        match t.kind {
            TokKind::Orange => {
                if let Some(m) = item_no_re().captures(&t.text) {
                    items.push(Item { no: m[1].to_string(), ..Default::default() });
                    pending.clear();
                }
            }
            TokKind::Title => {
                if let Some(cur) = items.last_mut() {
                    if cur.description.is_empty() {
                        cur.description = t.text.clone();
                    }
                }
            }
            TokKind::Label => {
                pending = t.text.clone();
            }
            TokKind::Value => {
                if let Some(cur) = items.last_mut() {
                    if !pending.is_empty() && !t.text.is_empty() {
                        cur.attributes.push(Attr { label: pending.clone(), value: t.text.clone() });
                    }
                }
                pending.clear();
            }
            _ => {}
        }
    }
    items
}

fn section_index(toks: &[Tok], needle: &str) -> Option<usize> {
    toks.iter()
        .position(|t| t.kind == TokKind::Section && t.text.to_uppercase().contains(needle))
}

/// isPhone: digits (with spaces/dashes/parens) and at least 6 of them.
fn is_phone(s: &str) -> bool {
    let mut digits = 0;
    for r in s.chars() {
        match r {
            '0'..='9' => digits += 1,
            ' ' | '-' | '(' | ')' | '+' | ',' => {}
            _ => return false,
        }
    }
    digits >= 6
}

/// Concatenates all descendant text, collapsing whitespace.
fn text_of(node: NodeRef<'_, Node>) -> String {
    let mut parts: Vec<String> = Vec::new();
    collect_text(node, &mut parts);
    parts
        .join(" ")
        .split_whitespace()
        .collect::<Vec<_>>()
        .join(" ")
}

fn collect_text(node: NodeRef<'_, Node>, parts: &mut Vec<String>) {
    if let Node::Text(t) = node.value() {
        parts.push(t.to_string());
    }
    for child in node.children() {
        collect_text(child, parts);
    }
}

/// Climbs from an icon node to the closest ancestor that carries text (the
/// icon's value sits as a sibling text node a couple of levels up).
fn nearest_text(node: NodeRef<'_, Node>) -> String {
    let mut ancestor = node.parent();
    let mut depth = 0;
    while let Some(a) = ancestor {
        if depth >= 5 {
            break;
        }
        let t = text_of(a);
        if !t.is_empty() {
            return t;
        }
        ancestor = a.parent();
        depth += 1;
    }
    String::new()
}

#[cfg(test)]
mod tests {
    use super::parse;

    // Mirrors the real GeBIZ JSF markup: contact values sit as text after an
    // icon div, items are delimited by an "Item No. N" orange label + a title
    // + label/value pairs. Same fixture as the Go parse_test.go.
    const FIXTURE: &str = r#"<html><body>
<div class="formSectionHeader1_TEXT">WHO TO CONTACT</div>
<div class="formOutputText_HIDDEN-LABEL outputText_TITLE-BLACK">TAN TIONG LEI</div>
<div><div style="padding-right:5px;display:inline-block;"><div style="background-image: url('/ptn/desktop/images/imageicons/fontawesome/envelope_20_434343.png');"></div></div>TTIONGLE@dsta.gov.sg</div>
<div><div style="display:inline-block;"><div style="background-image: url('/ptn/desktop/images/imageicons/fontawesome/phone_20_434343.png');"></div></div>68194165</div>
<div><div style="display:inline-block;"><div style="background-image: url('/ptn/desktop/images/imageicons/fontawesome/location_20.png');"></div></div>1 Depot Road, Singapore 109679</div>
<div class="formSectionHeader1_TEXT">AWARDING AGENCY</div>
<div class="formSectionHeader1_TEXT">ITEMS TO RESPOND</div>
<span class="outputText_LABEL-ORANGE">Item No. 1<br/>Mandatory to Bid: No</span>
<div class="formOutputText_HIDDEN-LABEL outputText_TITLE-BLACK">VALVE SHUTOFF</div>
<label><span>Unit of Measurement</span></label>
<div class="formOutputText_VALUE-DIV">Each</div>
<label><span>Required Quantity</span></label>
<div class="formOutputText_VALUE-DIV">1.0</div>
<div class="formSectionHeader2_TEXT">PART NO. INFORMATION</div>
<label><span>NSN</span></label>
<div class="formOutputText_VALUE-DIV">3291268-1</div>
</body></html>"#;

    #[test]
    fn test_parse() {
        let d = parse(FIXTURE.as_bytes());

        assert_eq!(d.contact.name, "TAN TIONG LEI");
        assert_eq!(d.contact.email, "TTIONGLE@dsta.gov.sg");
        assert_eq!(d.contact.phone, "68194165");
        assert!(!d.contact.address.is_empty(), "address empty");

        assert_eq!(d.items.len(), 1, "items = {}, want 1", d.items.len());
        let it = &d.items[0];
        assert_eq!(it.no, "1");
        assert_eq!(it.description, "VALVE SHUTOFF");
        // Unit of Measurement, Required Quantity, NSN (the sub-section attr counts too).
        assert_eq!(it.attributes.len(), 3, "attributes = {:?}", it.attributes);
    }
}
