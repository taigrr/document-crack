package main

import (
	"bytes"
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
