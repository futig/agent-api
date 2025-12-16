package formatter

import (
	"bytes"
	"os"

	"github.com/jung-kurt/gofpdf"
)

const (
	pdfContentType   = "application/pdf"
	pdfFileExtension = ".pdf"

	// pdfFontName is the internal name used by gofpdf
	// for the UTF-8 capable font.
	pdfFontName = "DejaVuSans"

	// Relative paths where the TTF font may live.
	// In Docker runtime we copy fonts to /app/ttf,
	// so for the compiled binary the path is ./ttf/DejaVuSans.ttf.
	pdfFontRuntimePath = "ttf/DejaVuSans.ttf"

	// Source-relative path (useful when running from repo root with `go run`).
	pdfFontSourcePath = "internal/pkg/formatter/ttf/DejaVuSans.ttf"
)

type PDFFormatter struct{}

func NewPDFFormatter() *PDFFormatter {
	return &PDFFormatter{}
}

// resolveFontPath tries to find the DejaVuSans font in
// runtime layout (next to the binary) or source layout.
func resolveFontPath() string {
	// 1) Try runtime-relative path from current working directory.
	if _, err := os.Stat(pdfFontRuntimePath); err == nil {
		return pdfFontRuntimePath
	}

	// 2) Try source-relative path (useful in local dev).
	if _, err := os.Stat(pdfFontSourcePath); err == nil {
		return pdfFontSourcePath
	}

	return ""
}

func (mf *PDFFormatter) Format(text string) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Try to use UTF-8 capable DejaVuSans font, bundled with the project.
	fontName := "Arial"
	if fontPath := resolveFontPath(); fontPath != "" {
		// Register regular and bold styles under the same family name
		pdf.AddUTF8Font(pdfFontName, "", fontPath)
		pdf.AddUTF8Font(pdfFontName, "B", fontPath)
		fontName = pdfFontName
	}

	pdf.SetFont(fontName, "B", 20)
	pdf.Cell(0, 10, baseTitle)
	pdf.Ln(12)

	pdf.SetFont(fontName, "", 12)
	_, lineHeight := pdf.GetFontSize()
	pdf.MultiCell(0, lineHeight*1.5, text, "", "", false)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (mf *PDFFormatter) ContentType() string {
	return pdfContentType
}

func (mf *PDFFormatter) FileExtension() string {
	return pdfFileExtension
}
