package crack

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"io"
	"net/http"
	"strings"
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

//go:embed testdata/demo.odt
var demoOdt []byte

//go:embed testdata/demo.pptx
var demoPptx []byte

//go:embed testdata/expected_docx.xml
var expectedDocx []byte

func TestDetectFileType(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    FileType
		wantErr bool
	}{
		{
			name:    "PDF file",
			content: []byte("%PDF-1.4"),
			want:    TypePDF,
		},
		{
			name:    "PDF file from demo",
			content: demoPDF,
			want:    TypePDF,
		},
		{
			name:    "DOC file",
			content: demoDoc,
			want:    TypeDOC,
		},
		{
			name:    "DOCX file",
			content: demoDocx,
			want:    TypeDOCX,
		},
		{
			name:    "ODT file",
			content: demoOdt,
			want:    TypeDOCX, // ODT starts with PK like DOCX
		},
		{
			name:    "PPTX file",
			content: demoPptx,
			want:    TypeDOCX, // PPTX starts with PK like DOCX
		},
		{
			name:    "Text file",
			content: []byte("plain text"),
			want:    TypeTXT,
		},
		{
			name:    "Text file from demo",
			content: demoTxt,
			want:    TypeTXT,
		},
		{
			name:    "Empty file",
			content: []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(string(tt.content))
			got, err := detectFileType(reader)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("detectFileType: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

func TestFromBytes(t *testing.T) {
	// Set up mock HTTP client for URL tests
	originalClient := http.DefaultClient
	defer func() { http.DefaultClient = originalClient }()
	http.DefaultClient = &http.Client{Transport: &mockTransport{}}

	tests := []struct {
		name        string
		data        []byte
		wantType    FileType
		wantContent string
		wantErr     string
	}{
		{
			name:        "DOC file",
			data:        demoDoc,
			wantType:    TypeDOC,
			wantContent: "",
		},
		{
			name:        "DOCX file",
			data:        demoDocx,
			wantType:    TypeDOCX,
			wantContent: string(expectedDocx),
		},
		{
			name:     "ODT file",
			data:     demoOdt,
			wantType: TypeODT,
			wantErr:  "failed to parse",
		},
		{
			name:        "PPTX file",
			data:        demoPptx,
			wantType:    TypePPTX,
			wantContent: "This is a pptx document.\n\nThis is a link.",
		},
		{
			name:        "PDF file",
			data:        demoPDF,
			wantType:    TypePDF,
			wantContent: "Dummy PDF file",
		},
		{
			name:        "Demo text file",
			data:        demoTxt,
			wantType:    TypeTXT,
			wantContent: "Sample txt file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := FromBytes(tt.data)
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
					return
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("FromBytes: %v", err)
			}
			if doc.Type != tt.wantType {
				t.Errorf("expected type %s, got %s", tt.wantType, doc.Type)
			}
			if tt.wantContent != "" {
				content := strings.Join(doc.Content, "\n")
				if content != tt.wantContent {
					t.Errorf("content mismatch:\nwant: %q\ngot:  %q", tt.wantContent, content)
				}
			}
		})
	}
}

func TestFromURL(t *testing.T) {
	originalClient := http.DefaultClient
	defer func() { http.DefaultClient = originalClient }()
	http.DefaultClient = &http.Client{Transport: &mockTransport{}}

	tests := []struct {
		name        string
		url         string
		wantType    FileType
		wantContent string
		wantErr     bool
	}{
		{
			name:        "DOC file",
			url:         "http://localhost/test.doc",
			wantType:    TypeDOC,
			wantContent: "",
		},
		{
			name:        "DOCX file",
			url:         "http://localhost/test.docx",
			wantType:    TypeDOCX,
			wantContent: string(expectedDocx),
		},
		{
			name:        "PPTX file",
			url:         "http://localhost/test.pptx",
			wantType:    TypePPTX,
			wantContent: "This is a pptx document.\n\nThis is a link.",
		},
		{
			name:        "PDF file",
			url:         "http://localhost/test.pdf",
			wantType:    TypePDF,
			wantContent: "Dummy PDF file",
		},
		{
			name:        "Demo text file",
			url:         "http://localhost/demo.txt",
			wantType:    TypeTXT,
			wantContent: "Sample txt file",
		},
		{
			name:    "Invalid URL",
			url:     "invalid://url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := FromURL(context.Background(), tt.url)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("FromURL: %v", err)
			}
			if doc.Type != tt.wantType {
				t.Errorf("expected type %s, got %s", tt.wantType, doc.Type)
			}
			if tt.wantContent != "" {
				content := strings.Join(doc.Content, "\n")
				if content != tt.wantContent {
					t.Errorf("content mismatch:\nwant: %q\ngot:  %q", tt.wantContent, content)
				}
			}
		})
	}
}

func TestFromFile(t *testing.T) {
	// Test with non-existent file
	_, err := FromFile("/nonexistent/file.pdf")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

type mockTransport struct{}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.String() {
	case "http://localhost/test.doc":
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(demoDoc)),
		}, nil
	case "http://localhost/test.docx":
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(demoDocx)),
		}, nil
	case "http://localhost/test.odt":
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(demoOdt)),
		}, nil
	case "http://localhost/test.pptx":
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(demoPptx)),
		}, nil
	case "http://localhost/test.pdf":
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(demoPDF)),
		}, nil
	case "http://localhost/demo.txt":
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(demoTxt)),
		}, nil
	}
	return nil, errors.New("mock transport error")
}
