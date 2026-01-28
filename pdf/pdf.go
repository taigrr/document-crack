// Package pdf provides PDF text extraction.
package pdf

import (
	"fmt"
	"io"
	"strings"

	"github.com/ledongthuc/pdf"
	pdfcpu "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// PDF represents a PDF document for text extraction.
type PDF struct {
	Reader   io.ReaderAt
	Size     int64
	Title    string
	Contents []string
}

// New creates a new PDF extractor.
func New(r io.ReaderAt, size int64) *PDF {
	return &PDF{
		Reader: r,
		Size:   size,
	}
}

func (p *PDF) getTitle(rs io.ReadSeeker) error {
	info, err := pdfcpu.PDFInfo(rs, "", []string{}, false, model.NewDefaultConfiguration())
	if err != nil {
		// Title extraction is optional, don't fail if we can't get it
		return nil
	}
	p.Title = info.Title
	return nil
}

// Load extracts text content from all pages.
func (p *PDF) Load() error {
	reader, err := pdf.NewReader(p.Reader, p.Size)
	if err != nil {
		return fmt.Errorf("failed to create PDF reader: %w", err)
	}

	numPages := reader.NumPage()
	if numPages == 0 {
		return fmt.Errorf("PDF has no pages")
	}

	contents := make([]string, 0, numPages)
	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			contents = append(contents, "")
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			contents = append(contents, "")
			continue
		}
		contents = append(contents, strings.TrimSpace(text))
	}

	p.Contents = contents

	hasContent := false
	for _, content := range contents {
		if content != "" {
			hasContent = true
			break
		}
	}
	if !hasContent {
		return fmt.Errorf("no text content found in PDF")
	}

	return nil
}

// LoadWithTitle loads the PDF and also extracts the title metadata.
func (p *PDF) LoadWithTitle(rs io.ReadSeeker) error {
	if err := p.Load(); err != nil {
		return err
	}
	return p.getTitle(rs)
}
