package formatter

import (
	"bytes"
	"fmt"
)

const (
	markdownContentType   = "text/markdown; charset=utf-8"
	markdownFileExtension = ".md"
)

type MarkdownFormatter struct{}

func NewMarkdownFormatter() *MarkdownFormatter {
	return &MarkdownFormatter{}
}

func (mf *MarkdownFormatter) Format(text string) ([]byte, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# %s\n\n%s\n", baseTitle, text)
	return buf.Bytes(), nil
}

func (mf *MarkdownFormatter) ContentType() string {
	return markdownContentType
}

func (mf *MarkdownFormatter) FileExtension() string {
	return markdownFileExtension
}
