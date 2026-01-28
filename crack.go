// Package crack provides document text extraction for various file formats.
package crack

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/taigrr/document-crack/doc"
	"github.com/taigrr/document-crack/docx"
	"github.com/taigrr/document-crack/odt"
	"github.com/taigrr/document-crack/pdf"
	"github.com/taigrr/document-crack/pptx"
)

// FileType represents the detected document format.
type FileType string

const (
	TypePDF  FileType = "PDF"
	TypeDOC  FileType = "DOC"
	TypeDOCX FileType = "DOCX"
	TypeODT  FileType = "ODT"
	TypePPTX FileType = "PPTX"
	TypeTXT  FileType = "TXT"
)

// Document represents the extracted content from a file.
type Document struct {
	Type    FileType
	Title   string
	Content []string
}

// ErrUnknownFormat is returned when the file format cannot be determined.
var ErrUnknownFormat = errors.New("unknown file format")

// MaxDownloadSize is the maximum file size for URL downloads (100MB).
const MaxDownloadSize = 100 << 20

// FromFile extracts content from a file at the given path.
func FromFile(path string) (Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return Document{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return Document{}, fmt.Errorf("failed to stat file: %w", err)
	}

	return FromReader(f, info.Size())
}

// FromBytes extracts content from a byte slice.
func FromBytes(data []byte) (Document, error) {
	reader := bytes.NewReader(data)
	return FromReader(reader, int64(len(data)))
}

// FromReader extracts content from an io.ReaderAt with known size.
func FromReader(r io.ReaderAt, size int64) (Document, error) {
	// Create a section reader to detect file type
	sr := io.NewSectionReader(r, 0, size)

	fileType, err := detectFileType(sr)
	if err != nil {
		return Document{}, err
	}

	return crack(r, size, fileType)
}

// FromURL downloads and extracts content from a URL.
func FromURL(ctx context.Context, fileURL string) (Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return Document{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Document{}, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Document{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Limit download size to prevent OOM
	limitedReader := io.LimitReader(resp.Body, MaxDownloadSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return Document{}, fmt.Errorf("failed to read response: %w", err)
	}
	if len(data) > MaxDownloadSize {
		return Document{}, fmt.Errorf("file too large (max %d bytes)", MaxDownloadSize)
	}

	return FromBytes(data)
}

// detectFileType identifies the file format based on magic bytes.
func detectFileType(r io.Reader) (FileType, error) {
	buf := make([]byte, 8)
	n, err := r.Read(buf)
	if err != nil {
		return "", fmt.Errorf("failed to read file header: %w", err)
	}
	if n < 2 {
		return "", ErrUnknownFormat
	}

	switch {
	case bytes.HasPrefix(buf, []byte{0x25, 0x50, 0x44, 0x46, 0x2D}): // %PDF-
		return TypePDF, nil
	case bytes.HasPrefix(buf, []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}): // DOC
		return TypeDOC, nil
	case bytes.HasPrefix(buf, []byte{0x50, 0x4B}): // PK (ZIP-based: DOCX, ODT, PPTX)
		return TypeDOCX, nil // Will fallthrough to ODT/PPTX if DOCX parsing fails
	default:
		return TypeTXT, nil
	}
}

// crack extracts content based on the detected file type.
func crack(r io.ReaderAt, size int64, fileType FileType) (Document, error) {
	switch fileType {
	case TypePDF:
		return crackPDF(r, size)
	case TypeDOC:
		return crackDOC(r, size)
	case TypeDOCX:
		// DOCX, ODT, and PPTX all start with PK (ZIP)
		// Try DOCX first, fall through to PPTX, then ODT
		doc, err := crackDOCX(r, size)
		if err == nil {
			return doc, nil
		}
		doc, err = crackPPTX(r, size)
		if err == nil {
			return doc, nil
		}
		return crackODT(r, size)
	case TypeTXT:
		return crackTXT(r, size)
	default:
		return Document{}, ErrUnknownFormat
	}
}

func crackPDF(r io.ReaderAt, size int64) (Document, error) {
	p := pdf.New(r, size)

	// Need a ReadSeeker for title extraction
	sr := io.NewSectionReader(r, 0, size)
	if err := p.LoadWithTitle(sr); err != nil {
		return Document{}, fmt.Errorf("failed to parse PDF: %w", err)
	}

	return Document{
		Type:    TypePDF,
		Title:   p.Title,
		Content: p.Contents,
	}, nil
}

func crackDOC(r io.ReaderAt, size int64) (Document, error) {
	sr := io.NewSectionReader(r, 0, size)
	d := doc.New(sr)
	if err := d.Load(); err != nil {
		return Document{}, fmt.Errorf("failed to parse DOC: %w", err)
	}

	return Document{
		Type:    TypeDOC,
		Title:   d.Title,
		Content: []string{d.Text},
	}, nil
}

func crackDOCX(r io.ReaderAt, size int64) (Document, error) {
	d := docx.New(r, size)
	if err := d.Load(); err != nil {
		return Document{}, fmt.Errorf("failed to parse DOCX: %w", err)
	}

	return Document{
		Type:    TypeDOCX,
		Title:   d.Title,
		Content: []string{d.Text},
	}, nil
}

func crackPPTX(r io.ReaderAt, size int64) (Document, error) {
	p := pptx.New(r, size)
	if err := p.Load(); err != nil {
		return Document{}, fmt.Errorf("failed to parse PPTX: %w", err)
	}

	return Document{
		Type:    TypePPTX,
		Title:   p.Title,
		Content: []string{p.Text},
	}, nil
}

func crackODT(r io.ReaderAt, size int64) (Document, error) {
	d := odt.New(r, size)
	if err := d.Load(); err != nil {
		return Document{}, fmt.Errorf("failed to parse ODT: %w", err)
	}

	return Document{
		Type:    TypeODT,
		Title:   d.Title,
		Content: []string{d.Text},
	}, nil
}

func crackTXT(r io.ReaderAt, size int64) (Document, error) {
	sr := io.NewSectionReader(r, 0, size)
	data, err := io.ReadAll(sr)
	if err != nil {
		return Document{}, fmt.Errorf("failed to read text file: %w", err)
	}

	return Document{
		Type:    TypeTXT,
		Title:   "",
		Content: []string{string(data)},
	}, nil
}
