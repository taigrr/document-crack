// Package pdf provides PDF text extraction.
//
// Text extraction uses ledongthuc/pdf (pure Go) as the primary method.
// When that fails to extract text (common with CIDFont/ToUnicode PDFs
// from banks and credit card companies), it falls back to pdftotext
// from the poppler-utils package if available on the system.
package pdf

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	gopdf "github.com/ledongthuc/pdf"
	pdfcpu "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// PDF represents a PDF document for text extraction.
type PDF struct {
	Reader   io.ReaderAt
	Size     int64
	Title    string
	Contents []string

	// FilePath is an optional hint for the original file path.
	// When set and the pure-Go extractor fails, the fallback
	// can use pdftotext directly without writing a temp file.
	FilePath string
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
// It first tries the pure-Go PDF reader, then falls back to pdftotext.
func (p *PDF) Load() error {
	err := p.loadGo()
	if err == nil {
		return nil
	}

	// Pure Go extraction failed — try pdftotext fallback
	fallbackErr := p.loadPdftotext()
	if fallbackErr == nil {
		return nil
	}

	// Both failed — return the original error (more informative)
	return err
}

// loadGo extracts text using the pure-Go PDF library.
func (p *PDF) loadGo() error {
	reader, err := gopdf.NewReader(p.Reader, p.Size)
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

	p.Contents = contents
	return nil
}

// loadPdftotext extracts text using the pdftotext CLI (poppler-utils).
// This handles PDFs with CIDFont, ToUnicode maps, and other complex
// encodings that the pure-Go library cannot decode.
func (p *PDF) loadPdftotext() error {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return fmt.Errorf("pdftotext not available: install poppler-utils")
	}

	// Determine the file path to pass to pdftotext
	pdfPath := p.FilePath
	var tmpFile *os.File

	if pdfPath == "" {
		// Write reader contents to a temp file
		var err error
		tmpFile, err = os.CreateTemp("", "document-crack-*.pdf")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		sr := io.NewSectionReader(p.Reader, 0, p.Size)
		if _, err := io.Copy(tmpFile, sr); err != nil {
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		tmpFile.Close()
		pdfPath = tmpFile.Name()
	}

	// Run pdftotext with layout preservation
	cmd := exec.Command("pdftotext", "-layout", pdfPath, "-")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pdftotext failed: %w (stderr: %s)", err, stderr.String())
	}

	text := strings.TrimSpace(stdout.String())
	if text == "" {
		return fmt.Errorf("pdftotext produced no output")
	}

	// Split into pages on form-feed characters
	pages := strings.Split(text, "\f")
	contents := make([]string, 0, len(pages))
	for _, page := range pages {
		contents = append(contents, strings.TrimSpace(page))
	}

	p.Contents = contents
	return nil
}

// LoadWithTitle loads the PDF and also extracts the title metadata.
func (p *PDF) LoadWithTitle(rs io.ReadSeeker) error {
	if err := p.Load(); err != nil {
		return err
	}
	return p.getTitle(rs)
}
