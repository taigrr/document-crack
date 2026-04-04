package doc

import (
	"bytes"
	_ "embed"
	"testing"
)

//go:embed testdata/demo.doc
var demoDoc []byte

func TestNew(t *testing.T) {
	r := bytes.NewReader(demoDoc)
	d := New(r)
	if d == nil {
		t.Fatal("New returned nil")
	}
}

func TestLoad(t *testing.T) {
	r := bytes.NewReader(demoDoc)
	d := New(r)

	if err := d.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	// DOC extraction may return empty for some files, just verify no crash
}

func TestLoadInvalidData(t *testing.T) {
	data := []byte("not a real doc file")
	r := bytes.NewReader(data)
	d := New(r)

	if err := d.Load(); err == nil {
		t.Error("expected error for invalid DOC data")
	}
}
