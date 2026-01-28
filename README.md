# document-crack

A Go library for extracting text content from various document formats.

## Supported Formats

- **PDF** - Portable Document Format
- **DOCX** - Microsoft Word (Open XML)
- **DOC** - Microsoft Word (Legacy)
- **PPTX** - Microsoft PowerPoint (Open XML)
- **ODT** - OpenDocument Text
- **TXT** - Plain text

## Installation

```bash
go get github.com/taigrr/document-crack
```

## Usage

### From a file path

```go
package main

import (
    "fmt"
    "log"

    crack "github.com/taigrr/document-crack"
)

func main() {
    doc, err := crack.FromFile("/path/to/document.pdf")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Type: %s\n", doc.Type)
    fmt.Printf("Title: %s\n", doc.Title)
    fmt.Printf("Content: %v\n", doc.Content)
}
```

### From bytes

```go
doc, err := crack.FromBytes(data)
if err != nil {
    log.Fatal(err)
}
```

### From io.ReaderAt

```go
doc, err := crack.FromReader(reader, size)
if err != nil {
    log.Fatal(err)
}
```

### From URL

```go
doc, err := crack.FromURL(ctx, "https://example.com/document.pdf")
if err != nil {
    log.Fatal(err)
}
```

## Document Structure

```go
type Document struct {
    Type    FileType  // PDF, DOCX, DOC, PPTX, ODT, TXT
    Title   string    // Document title (if available)
    Content []string  // Text content (per page for PDFs, single string for others)
}
```

## License

0BSD
