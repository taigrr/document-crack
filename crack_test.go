package crack

import (
	"context"
	_ "embed"
	"testing"
)

//go:embed testdata/demo.pdf
var demoPDF []byte

//go:embed testdata/demo.txt
var demoTxt []byte

//go:embed testdata/demo.doc
var demoDoc []byte

//go:embed testdata/demo.docx
var demoDocx []byte

//go:embed testdata/demo.pptx
var demoPptx []byte

func TestFromBytes_PDF(t *testing.T) {
	doc, err := FromBytes(demoPDF)
	if err != nil {
		t.Fatalf("FromBytes PDF: %v", err)
	}
	if doc.Type != TypePDF {
		t.Errorf("expected type %s, got %s", TypePDF, doc.Type)
	}
	if len(doc.Content) == 0 {
		t.Error("expected content, got empty")
	}
}

func TestFromBytes_TXT(t *testing.T) {
	doc, err := FromBytes(demoTxt)
	if err != nil {
		t.Fatalf("FromBytes TXT: %v", err)
	}
	if doc.Type != TypeTXT {
		t.Errorf("expected type %s, got %s", TypeTXT, doc.Type)
	}
	if len(doc.Content) == 0 || doc.Content[0] == "" {
		t.Error("expected content, got empty")
	}
}

func TestFromBytes_DOC(t *testing.T) {
	doc, err := FromBytes(demoDoc)
	if err != nil {
		t.Fatalf("FromBytes DOC: %v", err)
	}
	if doc.Type != TypeDOC {
		t.Errorf("expected type %s, got %s", TypeDOC, doc.Type)
	}
}

func TestFromBytes_DOCX(t *testing.T) {
	doc, err := FromBytes(demoDocx)
	if err != nil {
		t.Fatalf("FromBytes DOCX: %v", err)
	}
	if doc.Type != TypeDOCX {
		t.Errorf("expected type %s, got %s", TypeDOCX, doc.Type)
	}
}

func TestFromBytes_PPTX(t *testing.T) {
	doc, err := FromBytes(demoPptx)
	if err != nil {
		t.Fatalf("FromBytes PPTX: %v", err)
	}
	if doc.Type != TypePPTX {
		t.Errorf("expected type %s, got %s", TypePPTX, doc.Type)
	}
	if len(doc.Content) == 0 || doc.Content[0] == "" {
		t.Error("expected content, got empty")
	}
}

func TestDetectFileType(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    FileType
		wantErr bool
	}{
		{"PDF", demoPDF, TypePDF, false},
		{"DOC", demoDoc, TypeDOC, false},
		{"DOCX", demoDocx, TypeDOCX, false},
		{"PPTX", demoPptx, TypePPTX, false},
		{"TXT", []byte("plain text"), TypeTXT, false},
		{"Empty", []byte{}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := FromBytes(tt.data)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("FromBytes: %v", err)
			}
			if doc.Type != tt.want {
				t.Errorf("expected type %s, got %s", tt.want, doc.Type)
			}
		})
	}
}

func TestFromURL_Invalid(t *testing.T) {
	_, err := FromURL(context.Background(), "invalid://url")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}
