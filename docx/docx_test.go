package docx

import (
	"bytes"
	_ "embed"
	"testing"
)

//go:embed testdata/demo.docx
var demoDocx []byte

//go:embed testdata/expected.xml
var expectedContent []byte

func TestNew(t *testing.T) {
	r := bytes.NewReader(demoDocx)
	d := New(r, int64(len(demoDocx)))
	if d == nil {
		t.Fatal("New returned nil")
	}
	if d.Size != int64(len(demoDocx)) {
		t.Errorf("expected size %d, got %d", len(demoDocx), d.Size)
	}
}

func TestLoad(t *testing.T) {
	r := bytes.NewReader(demoDocx)
	d := New(r, int64(len(demoDocx)))

	if err := d.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if d.Text == "" {
		t.Error("expected non-empty text")
	}
	if d.Text != string(expectedContent) {
		t.Errorf("content mismatch:\nwant: %q\ngot:  %q", string(expectedContent), d.Text)
	}
}

func TestLoadInvalidData(t *testing.T) {
	data := []byte("not a real docx")
	r := bytes.NewReader(data)
	d := New(r, int64(len(data)))

	if err := d.Load(); err == nil {
		t.Error("expected error for invalid DOCX data")
	}
}
