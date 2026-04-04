package pptx

import (
	"bytes"
	_ "embed"
	"testing"
)

//go:embed testdata/demo.pptx
var demoPptx []byte

func TestNew(t *testing.T) {
	r := bytes.NewReader(demoPptx)
	p := New(r, int64(len(demoPptx)))
	if p == nil {
		t.Fatal("New returned nil")
	}
	if p.Size != int64(len(demoPptx)) {
		t.Errorf("expected size %d, got %d", len(demoPptx), p.Size)
	}
}

func TestLoad(t *testing.T) {
	r := bytes.NewReader(demoPptx)
	p := New(r, int64(len(demoPptx)))

	if err := p.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if p.Text == "" {
		t.Error("expected non-empty text")
	}

	expected := "This is a pptx document.\n\nThis is a link."
	if p.Text != expected {
		t.Errorf("content mismatch:\nwant: %q\ngot:  %q", expected, p.Text)
	}
}

func TestLoadInvalidData(t *testing.T) {
	data := []byte("not a real pptx file")
	r := bytes.NewReader(data)
	p := New(r, int64(len(data)))

	if err := p.Load(); err == nil {
		t.Error("expected error for invalid PPTX data")
	}
}

func TestMapZipFiles(t *testing.T) {
	// Test with nil/empty input
	result := mapZipFiles(nil)
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}
