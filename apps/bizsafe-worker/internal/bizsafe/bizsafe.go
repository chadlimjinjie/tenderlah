// Package bizsafe scrapes the WSH Council's bizSAFE Self-Help page for the
// published Excel link, then parses every row of its hidden "Data" sheet.
package bizsafe

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/midtone/bizsafe-worker/internal/store"
)

const pageURL = "https://www.tal.sg/wshc/programmes/bizsafe/bizsafe-e-services"

var xlsxHrefRe = regexp.MustCompile(`(?i)href="([^"]+\.xlsx)"`)

// Fetch discovers the current bizSAFE Self-Help Excel file, downloads it, and
// parses its "Data" sheet into upsertable records.
func Fetch(ctx context.Context) ([]store.Record, error) {
	xlsxURL, err := findXLSXURL(ctx)
	if err != nil {
		return nil, err
	}
	data, err := download(ctx, xlsxURL)
	if err != nil {
		return nil, err
	}
	return parse(data)
}

// findXLSXURL scrapes pageURL for the bizSAFE Self-Help download link.
func findXLSXURL(ctx context.Context) (string, error) {
	body, err := get(ctx, pageURL, 5<<20)
	if err != nil {
		return "", fmt.Errorf("bizsafe: fetch page: %w", err)
	}
	m := xlsxHrefRe.FindSubmatch(body)
	if m == nil {
		return "", fmt.Errorf("bizsafe: no .xlsx link found on %s", pageURL)
	}
	base, err := url.Parse(pageURL)
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(string(m[1]))
	if err != nil {
		return "", fmt.Errorf("bizsafe: parse xlsx href: %w", err)
	}
	return base.ResolveReference(ref).String(), nil
}

func download(ctx context.Context, xlsxURL string) ([]byte, error) {
	body, err := get(ctx, xlsxURL, 20<<20)
	if err != nil {
		return nil, fmt.Errorf("bizsafe: download xlsx: %w", err)
	}
	return body, nil
}

func get(ctx context.Context, target string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MidtoneBizsafeBot/1.0)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, limit))
}

// parse reads the "Data" sheet (columns: UEN, COMPANY_NAME, CERT, TYPE, ACTIVE,
// EXPIRY, STATUS, APPROVAL, LEVEL, DESCRPTION) into records, skipping the header
// row and any trailing rows without a certificate number.
func parse(data []byte) ([]store.Record, error) {
	f, err := excelize.OpenReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("bizsafe: open xlsx: %w", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Data")
	if err != nil {
		return nil, fmt.Errorf("bizsafe: read Data sheet: %w", err)
	}

	var out []store.Record
	for _, row := range rows[1:] { // skip header
		cell := func(i int) string {
			if i < len(row) {
				return strings.TrimSpace(row[i])
			}
			return ""
		}
		cert := cell(2)
		if cert == "" {
			continue
		}
		out = append(out, store.Record{
			UEN:         cell(0),
			CompanyName: cell(1),
			Cert:        cert,
			Type:        cell(3),
			Active:      cell(4),
			Expiry:      parseDate(cell(5)),
			Status:      cell(6),
			Approval:    parseDate(cell(7)),
			Level:       cell(8),
			Description: cell(9),
		})
	}
	return out, nil
}

// parseDate handles both a pre-formatted date string (what excelize.GetRows
// yields for a date-styled cell — the sheet uses numFmtID 14, "mm-dd-yy") and
// a raw Excel date serial (the fallback if the cell's number format isn't
// recognized as a date).
func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	for _, layout := range [...]string{"01-02-06", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	if serial, err := strconv.ParseFloat(s, 64); err == nil {
		if t, err := excelize.ExcelDateToTime(serial, false); err == nil {
			return &t
		}
	}
	return nil
}
