// Package docx provides DOCX text extraction.
package docx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/nguyenthenguyen/docx"
)

// Docx represents a DOCX document for text extraction.
type Docx struct {
	Reader io.ReaderAt
	Text   string
	Title  string
	Size   int64
}

type coreProperties struct {
	XMLName xml.Name `xml:"http://schemas.openxmlformats.org/package/2006/metadata/core-properties coreProperties"`
	Title   string   `xml:"title"`
}

// New creates a new DOCX extractor.
func New(r io.ReaderAt, size int64) *Docx {
	return &Docx{
		Reader: r,
		Size:   size,
	}
}

func (d *Docx) getTitle() error {
	r, err := zip.NewReader(d.Reader, d.Size)
	if err != nil {
		return fmt.Errorf("failed to open .docx file: %w", err)
	}

	for _, file := range r.File {
		if strings.HasSuffix(file.Name, "docProps/core.xml") {
			rc, err := file.Open()
			if err != nil {
				return fmt.Errorf("failed to open core.xml: %w", err)
			}
			defer rc.Close()

			var props coreProperties
			decoder := xml.NewDecoder(rc)
			if err := decoder.Decode(&props); err != nil {
				if err == io.EOF {
					continue
				}
				return fmt.Errorf("failed to parse XML: %w", err)
			}

			d.Title = props.Title
			return nil
		}
	}
	return fmt.Errorf("core.xml file not found")
}

// Load extracts text content from the DOCX.
func (d *Docx) Load() error {
	r, err := docx.ReadDocxFromMemory(d.Reader, d.Size)
	if err != nil {
		return err
	}
	defer r.Close()

	editable := r.Editable()
	d.Text = editable.GetContent()
	if d.Text == "" {
		return fmt.Errorf("no content found")
	}
	return d.getTitle()
}
