package odt

import (
	"bytes"
	_ "embed"
	"testing"
)

//go:embed testdata/demo.odt
var demoOdt []byte

func TestNew(t *testing.T) {
	r := bytes.NewReader(demoOdt)
	d := New(r, int64(len(demoOdt)))
	if d == nil {
		t.Fatal("New returned nil")
	}
	if d.Size != int64(len(demoOdt)) {
		t.Errorf("expected size %d, got %d", len(demoOdt), d.Size)
	}
}

func TestLoad(t *testing.T) {
	r := bytes.NewReader(demoOdt)
	d := New(r, int64(len(demoOdt)))

	if err := d.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if d.Text == "" {
		t.Error("expected non-empty text")
	}

	expected := "This is a word document.\n\nThis is a link."
	if d.Text != expected {
		t.Errorf("content mismatch:\nwant: %q\ngot:  %q", expected, d.Text)
	}
}

func TestLoadInvalidData(t *testing.T) {
	data := []byte("not a real odt file")
	r := bytes.NewReader(data)
	d := New(r, int64(len(data)))

	if err := d.Load(); err == nil {
		t.Error("expected error for invalid ODT data")
	}
}
