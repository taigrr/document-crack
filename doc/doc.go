// Package doc provides legacy DOC text extraction.
package doc

import (
	"io"

	doc "github.com/EndFirstCorp/doc2txt"
)

// Doc represents a legacy DOC document for text extraction.
type Doc struct {
	Reader io.Reader
	Text   string
	Title  string
	Size   int64
}

// New creates a new DOC extractor.
func New(r io.Reader) *Doc {
	return &Doc{
		Reader: r,
	}
}

func (d *Doc) getTitle() error {
	// Legacy DOC format doesn't have easily extractable title
	return nil
}

// Load extracts text content from the DOC.
func (d *Doc) Load() error {
	r, err := doc.ParseDoc(d.Reader)
	if err != nil {
		return err
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	d.Text = string(b)

	return d.getTitle()
}
