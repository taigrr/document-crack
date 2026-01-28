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

type metaProperties struct {
	XMLName xml.Name `xml:"http://openxmlformats.org/officeDocument/2006/metadata/core-properties core-properties"`
	Title   string   `xml:"dc:title"`
}

type textNode struct {
	XMLName xml.Name `xml:"text:p"`
	Content string   `xml:",chardata"`
}

// New creates a new ODT extractor.
func New(r io.ReaderAt, size int64) *ODT {
	return &ODT{
		Reader: r,
		Size:   size,
	}
}

func (d *ODT) getTitle() error {
	r, err := zip.NewReader(d.Reader, d.Size)
	if err != nil {
		return fmt.Errorf("failed to open .odt from ReaderAt: %w", err)
	}

	for _, file := range r.File {
		if strings.HasSuffix(file.Name, "meta.xml") {
			rc, err := file.Open()
			if err != nil {
				return fmt.Errorf("failed to open meta.xml: %w", err)
			}
			defer rc.Close()

			var props metaProperties
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

	return fmt.Errorf("meta.xml file not found")
}

// Load extracts text content from the ODT.
func (d *ODT) Load() error {
	r, err := zip.NewReader(d.Reader, d.Size)
	if err != nil {
		return fmt.Errorf("failed to open .odt from ReaderAt: %w", err)
	}
	if err := d.getTitle(); err != nil {
		return err
	}

	for _, file := range r.File {
		if strings.HasSuffix(file.Name, "content.xml") {
			rc, err := file.Open()
			if err != nil {
				return fmt.Errorf("failed to open content.xml: %w", err)
			}
			defer rc.Close()

			content := strings.Builder{}
			decoder := xml.NewDecoder(rc)
			for {
				var node textNode
				if err := decoder.Decode(&node); err != nil {
					if err == io.EOF {
						break
					}
					return fmt.Errorf("failed to parse XML: %w", err)
				}
				content.WriteString(node.Content + "\n")
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
