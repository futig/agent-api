package formatter

import (
	"bytes"

	"github.com/unidoc/unioffice/document"
)

const (
	docxContentType   = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	docxFileExtension = ".docx"
)

type DOCXFormatter struct{}

func NewDOCXFormatter() *DOCXFormatter {
	return &DOCXFormatter{}
}

func (mf *DOCXFormatter) Format(text string) ([]byte, error) {
	doc := document.New()
	defer doc.Close()

	titlePar := doc.AddParagraph()
	titlePar.SetStyle("Heading1")
	titleRun := titlePar.AddRun()
	titleRun.AddText(baseTitle)

	doc.AddParagraph()

	bodyPar := doc.AddParagraph()
	bodyRun := bodyPar.AddRun()
	bodyRun.AddText(text)

	var buf bytes.Buffer
	if err := doc.Save(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (mf *DOCXFormatter) ContentType() string {
	return docxContentType
}

func (mf *DOCXFormatter) FileExtension() string {
	return docxFileExtension
}
