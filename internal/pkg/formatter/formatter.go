package formatter

import (
	"fmt"

	"github.com/futig/agent-backend/internal/entity"
)

const baseTitle = "Бизнес требования"

type Formatter interface {
	Format(plainText string) ([]byte, error)
	ContentType() string
	FileExtension() string
}

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) Create(format entity.ResultFormat) (Formatter, error) {
	switch format {
	case entity.FormatMarkdown:
		return NewMarkdownFormatter(), nil
	case entity.FormatDOCX:
		return NewDOCXFormatter(), nil
	case entity.FormatPDF:
		return NewPDFFormatter(), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}
