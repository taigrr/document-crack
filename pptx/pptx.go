// Package pptx provides PPTX text extraction.
package pptx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

const maxSize = 50 << 20 // 50 MB

type typeOverride struct {
	XMLName     xml.Name `xml:"Override"`
	ContentType string   `xml:"ContentType,attr"`
	PartName    string   `xml:"PartName,attr"`
}

type contentTypeDefinition struct {
	XMLName   xml.Name       `xml:"Types"`
	Overrides []typeOverride `xml:"Override"`
}

// PPTX represents a PPTX document for text extraction.
type PPTX struct {
	Title  string
	Size   int64
	Text   string
	Reader io.ReaderAt
}

// New creates a new PPTX extractor.
func New(r io.ReaderAt, size int64) *PPTX {
	return &PPTX{
		Reader: r,
		Size:   size,
	}
}

// Load extracts text content from the PPTX.
func (p *PPTX) Load() error {
	zr, err := zip.NewReader(p.Reader, p.Size)
	if err != nil {
		return fmt.Errorf("could not unzip: %v", err)
	}

	zipFiles := mapZipFiles(zr.File)
	zipFile := zipFiles["[Content_Types].xml"]
	if zipFile == nil {
		return fmt.Errorf("zipFile type definition not found")
	}
	contentTypeDef, err := getContentTypeDefinition(zipFile)
	if err != nil {
		return err
	}

	var textBody string
	for _, override := range contentTypeDef.Overrides {
		f := zipFiles[override.PartName]
		switch override.ContentType {
		case "application/vnd.openxmlformats-officedocument.presentationml.slide+xml",
			"application/vnd.openxmlformats-officedocument.drawingml.diagramData+xml":
			body, err := parseSlideText(f)
			if err != nil {
				return fmt.Errorf("could not parse pptx: %v", err)
			}
			textBody += body + "\n"
		}
	}
	p.Text = strings.TrimSpace(textBody)
	if p.Text == "" {
		return fmt.Errorf("pptx is empty")
	}
	return nil
}

func getContentTypeDefinition(zf *zip.File) (*contentTypeDefinition, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	x := &contentTypeDefinition{}
	if err := xml.NewDecoder(io.LimitReader(f, maxSize)).Decode(x); err != nil {
		return nil, err
	}
	return x, nil
}

func parseSlideText(f *zip.File) (string, error) {
	r, err := f.Open()
	if err != nil {
		return "", fmt.Errorf("error opening '%v' from archive: %v", f.Name, err)
	}
	defer r.Close()

	text, err := xmlToText(r, []string{"br", "p", "tab"}, []string{"instrText", "script"}, true)
	if err != nil {
		return "", fmt.Errorf("error parsing '%v': %v", f.Name, err)
	}
	return text, nil
}

func mapZipFiles(files []*zip.File) map[string]*zip.File {
	filesMap := make(map[string]*zip.File, 2*len(files))
	for _, f := range files {
		filesMap[f.Name] = f
		filesMap["/"+f.Name] = f
	}
	return filesMap
}

func xmlToText(r io.Reader, breaks []string, skip []string, strict bool) (string, error) {
	var result string
	dec := xml.NewDecoder(io.LimitReader(r, maxSize))
	dec.Strict = strict
	for {
		t, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		switch v := t.(type) {
		case xml.CharData:
			result += string(v)
		case xml.StartElement:
			for _, breakElement := range breaks {
				if v.Name.Local == breakElement {
					result += "\n"
				}
			}
			for _, skipElement := range skip {
				if v.Name.Local == skipElement {
					depth := 1
					for {
						t, err := dec.Token()
						if err != nil {
							return "", err
						}
						switch t.(type) {
						case xml.StartElement:
							depth++
						case xml.EndElement:
							depth--
						}
						if depth == 0 {
							break
						}
					}
				}
			}
		}
	}
	return result, nil
}
