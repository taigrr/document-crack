package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"charm.land/fang/v2"
	"github.com/spf13/cobra"
	crack "github.com/taigrr/document-crack/v2"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "crack <file>",
		Short: "Extract text from documents",
		Long:  "Crack extracts text content from PDF, DOCX, DOC, PPTX, ODT, and TXT files.",
		Args:  cobra.ExactArgs(1),
		RunE:  runCrack,
	}

	if err := fang.Execute(context.Background(), rootCmd, fang.WithVersion(version)); err != nil {
		os.Exit(1)
	}
}

func runCrack(cmd *cobra.Command, args []string) error {
	path := args[0]

	doc, err := crack.FromFile(path)
	if err != nil {
		return fmt.Errorf("failed to crack %s: %w", path, err)
	}

	if doc.Title != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "# %s\n\n", doc.Title)
	}

	fmt.Fprintln(cmd.OutOrStdout(), strings.Join(doc.Content, "\n"))

	return nil
}
