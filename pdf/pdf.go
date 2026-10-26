// Package pdf provides PDF text extraction.
//
// Text extraction uses ledongthuc/pdf (pure Go) as the primary method.
// When that fails to extract any text (common with CIDFont/ToUnicode PDFs
// such as bank and credit card statements), it falls back to pdftotext
// from poppler-utils if it is available on the system PATH.
package pdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
	// When set and the pure-Go extractor fails, the pdftotext fallback
	// reads the file directly instead of writing a temporary copy.
	FilePath string
}

// pdftotextTimeout bounds how long the pdftotext fallback may run.
const pdftotextTimeout = 2 * time.Minute

// errPdftotextUnavailable is returned when pdftotext is not on PATH.
var errPdftotextUnavailable = errors.New("pdftotext not available: install poppler-utils")

// goExtract is the primary extractor; it is a variable so tests can force
// the fallback path.
var goExtract = (*PDF).loadGo

// New creates a new PDF extractor.
func New(r io.ReaderAt, size int64) *PDF {
	return &PDF{
		Reader: r,
		Size:   size,
	}
}

func (p *PDF) getTitle(rs io.ReadSeeker) error {
	info, err := pdfcpu.PDFInfo(context.Background(), rs, "", []string{}, false, model.NewDefaultConfiguration())
	if err != nil {
		// Title extraction is optional, don't fail if we can't get it
		return nil
	}
	p.Title = info.Title
	return nil
}

// Load extracts text content from all pages.
// It first tries the pure-Go PDF reader, then falls back to pdftotext.
// If both fail, the error from the pure-Go reader is returned.
func (p *PDF) Load() error {
	err := goExtract(p)
	if err == nil {
		return nil
	}
	if fallbackErr := p.loadPdftotext(); fallbackErr == nil {
		return nil
	}
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
// encodings that the pure-Go library cannot decode. Pages are split on
// the form-feed characters pdftotext emits between pages.
func (p *PDF) loadPdftotext() error {
	bin, err := exec.LookPath("pdftotext")
	if err != nil {
		return errPdftotextUnavailable
	}

	pdfPath := p.FilePath
	if pdfPath != "" {
		// Absolute path so a name starting with "-" is never read as a flag.
		if pdfPath, err = filepath.Abs(pdfPath); err != nil {
			return fmt.Errorf("failed to resolve PDF path: %w", err)
		}
	} else {
		tmp, err := os.CreateTemp("", "document-crack-*.pdf")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer func() { _ = os.Remove(tmp.Name()) }()

		_, copyErr := io.Copy(tmp, io.NewSectionReader(p.Reader, 0, p.Size))
		closeErr := tmp.Close()
		if copyErr != nil {
			return fmt.Errorf("failed to write temp file: %w", copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("failed to write temp file: %w", closeErr)
		}
		pdfPath = tmp.Name()
	}

	ctx, cancel := context.WithTimeout(context.Background(), pdftotextTimeout)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, bin, "-layout", "-enc", "UTF-8", pdfPath, "-")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pdftotext failed: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	text := strings.TrimSpace(stdout.String())
	if text == "" {
		return fmt.Errorf("pdftotext produced no output")
	}

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
