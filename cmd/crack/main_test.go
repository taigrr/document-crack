package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunCrack(t *testing.T) {
	var stdout bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)

	err := runCrack(cmd, []string{"../../testdata/demo.txt"})
	if err != nil {
		t.Fatalf("runCrack: %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "Sample txt file") {
		t.Fatalf("expected output to contain sample text, got %q", got)
	}
}

func TestRunCrackMissingFile(t *testing.T) {
	var stdout bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)

	err := runCrack(cmd, []string{"../../testdata/missing.pdf"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to crack") {
		t.Fatalf("expected crack error context, got %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no output on error, got %q", stdout.String())
	}
}

// failOutput rejects one write and records any attempted writes after failure.
type failOutput struct {
	writes int
	failAt int
	err    error
}

func (w *failOutput) Write(data []byte) (int, error) {
	w.writes++
	if w.writes == w.failAt {
		return 0, w.err
	}
	return len(data), nil
}

func TestRunCrackOutputFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "titled.docx")
	var data bytes.Buffer
	archive := zip.NewWriter(&data)
	for name, content := range map[string]string{
		"word/_rels/document.xml.rels": `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"/>`,
		"word/document.xml":            `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Document body</w:t></w:r></w:p></w:body></w:document>`,
		"docProps/core.xml":            `<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Document title</dc:title></cp:coreProperties>`,
	} {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(entry, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	if err := runCrack(cmd, []string{path}); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); !strings.HasPrefix(got, "# Document title\n\n") || !strings.Contains(got, "Document body") {
		t.Fatalf("missing title or body: %q", got)
	}

	for _, test := range []struct {
		name, path string
		failAt     int
	}{
		{"plain body", "../../testdata/demo.txt", 1},
		{"title", path, 1},
		{"titled body", path, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			outputErr := errors.New("output unavailable")
			writer := &failOutput{failAt: test.failAt, err: outputErr}
			cmd := &cobra.Command{}
			cmd.SetOut(writer)
			if err := runCrack(cmd, []string{test.path}); !errors.Is(err, outputErr) {
				t.Fatalf("error = %v, want %v", err, outputErr)
			}
			if writer.writes != test.failAt {
				t.Fatalf("writes = %d, want %d", writer.writes, test.failAt)
			}
		})
	}
}
