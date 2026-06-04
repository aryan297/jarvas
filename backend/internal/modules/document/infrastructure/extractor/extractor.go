package extractor

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/ledongthuc/pdf"
)

// Extractor pulls plain text from a document binary.
type Extractor interface {
	Extract(r io.ReadSeeker, size int64) (string, error)
}

// ForMIME returns the right extractor for the given MIME type.
func ForMIME(mime string) Extractor {
	switch mime {
	case "application/pdf":
		return &PDFExtractor{}
	default:
		return &PlainTextExtractor{}
	}
}

// ── Plain text (TXT, MD, HTML, CSV) ──────────────────────────────────────────

type PlainTextExtractor struct{}

func (e *PlainTextExtractor) Extract(r io.ReadSeeker, _ int64) (string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ── PDF ───────────────────────────────────────────────────────────────────────

type PDFExtractor struct{}

func (e *PDFExtractor) Extract(r io.ReadSeeker, size int64) (string, error) {
	// ledongthuc/pdf needs a ReaderAt; wrap ReadSeeker with a buffer.
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("pdf read: %w", err)
	}

	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("pdf parse: %w", err)
	}

	var sb strings.Builder
	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		sb.WriteString(text)
		sb.WriteString("\n")
	}
	return sb.String(), nil
}
