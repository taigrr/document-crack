package pdf

import (
	"bytes"
	_ "embed"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

//go:embed testdata/twopage.pdf
var twoPagePDF []byte

var errForcedGo = errors.New("forced pure-Go failure")

// forceFallback makes the pure-Go extractor fail for the duration of the test.
func forceFallback(t *testing.T) {
	t.Helper()
	orig := goExtract
	goExtract = func(*PDF) error { return errForcedGo }
	t.Cleanup(func() { goExtract = orig })
}

func requirePdftotext(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext not installed (poppler-utils); skipping fallback test")
	}
}

func TestLoadFallbackFromReader(t *testing.T) {
	requirePdftotext(t)
	forceFallback(t)

	p := New(bytes.NewReader(twoPagePDF), int64(len(twoPagePDF)))
	if err := p.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(p.Contents) != 2 {
		t.Fatalf("expected 2 pages, got %d: %q", len(p.Contents), p.Contents)
	}
	if !strings.Contains(p.Contents[0], "Page one text") {
		t.Errorf("page 1 = %q, want it to contain %q", p.Contents[0], "Page one text")
	}
	if !strings.Contains(p.Contents[1], "Page two text") {
		t.Errorf("page 2 = %q, want it to contain %q", p.Contents[1], "Page two text")
	}
}

func TestLoadFallbackFromFilePath(t *testing.T) {
	requirePdftotext(t)
	forceFallback(t)

	// Reader is empty: the fallback must read from FilePath, not the reader.
	p := New(bytes.NewReader(nil), 0)
	p.FilePath = "testdata/twopage.pdf"
	if err := p.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(p.Contents) != 2 || !strings.Contains(p.Contents[1], "Page two text") {
		t.Fatalf("unexpected contents: %q", p.Contents)
	}
}

func TestLoadFallbackUnavailableReturnsGoError(t *testing.T) {
	forceFallback(t)
	t.Setenv("PATH", t.TempDir())

	p := New(bytes.NewReader(twoPagePDF), int64(len(twoPagePDF)))
	err := p.Load()
	if !errors.Is(err, errForcedGo) {
		t.Fatalf("expected pure-Go error, got %v", err)
	}
	if p.Contents != nil {
		t.Errorf("expected no contents, got %q", p.Contents)
	}
}

func TestLoadPdftotextUnavailable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	p := New(bytes.NewReader(twoPagePDF), int64(len(twoPagePDF)))
	if err := p.loadPdftotext(); !errors.Is(err, errPdftotextUnavailable) {
		t.Fatalf("expected errPdftotextUnavailable, got %v", err)
	}
}

func TestLoadInvalidDataFallbackFails(t *testing.T) {
	requirePdftotext(t)

	data := []byte("%PDF-1.4 truncated garbage")
	p := New(bytes.NewReader(data), int64(len(data)))
	err := p.Load()
	if err == nil {
		t.Fatal("expected error for invalid PDF data")
	}
	if strings.Contains(err.Error(), "pdftotext") {
		t.Errorf("expected the pure-Go error to be returned, got %v", err)
	}
}

func TestLoadPureGoStillPreferred(t *testing.T) {
	// With no override, the pure-Go path handles a simple PDF on its own.
	if err := goExtract(New(bytes.NewReader(twoPagePDF), int64(len(twoPagePDF)))); err != nil {
		t.Fatalf("pure-Go extraction failed on fixture: %v", err)
	}
}
