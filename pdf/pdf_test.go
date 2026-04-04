package pdf

import (
	"bytes"
	_ "embed"
	"io"
	"testing"
)

//go:embed testdata/demo.pdf
var demoPDF []byte

func TestNew(t *testing.T) {
	r := bytes.NewReader(demoPDF)
	p := New(r, int64(len(demoPDF)))
	if p == nil {
		t.Fatal("New returned nil")
	}
	if p.Size != int64(len(demoPDF)) {
		t.Errorf("expected size %d, got %d", len(demoPDF), p.Size)
	}
}

func TestLoad(t *testing.T) {
	r := bytes.NewReader(demoPDF)
	p := New(r, int64(len(demoPDF)))

	if err := p.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(p.Contents) == 0 {
		t.Fatal("expected at least one page of content")
	}

	found := false
	for _, c := range p.Contents {
		if c != "" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected non-empty content in at least one page")
	}
}

func TestLoadWithTitle(t *testing.T) {
	r := bytes.NewReader(demoPDF)
	p := New(r, int64(len(demoPDF)))

	sr := io.NewSectionReader(r, 0, int64(len(demoPDF)))
	if err := p.LoadWithTitle(sr); err != nil {
		t.Fatalf("LoadWithTitle: %v", err)
	}
	if len(p.Contents) == 0 {
		t.Fatal("expected at least one page of content")
	}
}

func TestLoadInvalidData(t *testing.T) {
	data := []byte("not a real PDF")
	r := bytes.NewReader(data)
	p := New(r, int64(len(data)))

	if err := p.Load(); err == nil {
		t.Error("expected error for invalid PDF data")
	}
}
