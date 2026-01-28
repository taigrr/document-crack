// Package odt provides ODT text extraction.
package odt

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// ODT represents an ODT document for text extraction.
type ODT struct {
	Reader io.ReaderAt
	Text   string
	Title  string
	Size   int64
}

// ODT uses ODF meta format, title can be in different locations
type officeMeta struct {
	XMLName xml.Name `xml:"document-meta"`
	Meta    metaInfo `xml:"meta"`
}

type metaInfo struct {
	Title string `xml:"title"`
}

// New creates a new ODT extractor.
func New(r io.ReaderAt, size int64) *ODT {
	return &ODT{
		Reader: r,
		Size:   size,
	}
}

func (d *ODT) getTitle(zr *zip.Reader) {
	for _, file := range zr.File {
		if file.Name == "meta.xml" {
			rc, err := file.Open()
			if err != nil {
				return // Title is optional, don't fail
			}
			defer rc.Close()

			var meta officeMeta
			decoder := xml.NewDecoder(rc)
			if err := decoder.Decode(&meta); err != nil {
				return // Title is optional, don't fail
			}

			d.Title = meta.Meta.Title
			return
		}
	}
}

// Load extracts text content from the ODT.
func (d *ODT) Load() error {
	r, err := zip.NewReader(d.Reader, d.Size)
	if err != nil {
		return fmt.Errorf("failed to open .odt from ReaderAt: %w", err)
	}
	d.getTitle(r) // Title is optional, don't fail if missing

	for _, file := range r.File {
		if file.Name == "content.xml" {
			rc, err := file.Open()
			if err != nil {
				return fmt.Errorf("failed to open content.xml: %w", err)
			}
			defer rc.Close()

			content := strings.Builder{}
			decoder := xml.NewDecoder(rc)

			// Use token-based parsing to find all text content
			for {
				tok, err := decoder.Token()
				if err != nil {
					if err == io.EOF {
						break
					}
					return fmt.Errorf("failed to parse XML: %w", err)
				}

				switch t := tok.(type) {
				case xml.StartElement:
					// Add newline for paragraph elements
					if t.Name.Local == "p" && t.Name.Space == "urn:oasis:names:tc:opendocument:xmlns:text:1.0" {
						if content.Len() > 0 {
							content.WriteString("\n")
						}
					}
				case xml.CharData:
					content.Write(t)
				}
			}

			d.Text = strings.TrimSpace(content.String())
			if d.Text == "" {
				return fmt.Errorf("no text found in content.xml")
			}
			return nil
		}
	}

	return fmt.Errorf("content.xml file not found")
}
