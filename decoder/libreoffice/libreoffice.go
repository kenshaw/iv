// Package libreoffice supplies an office document decoder for iv, converting
// documents to pdf with the `soffice` command.
package libreoffice

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/cmd"
	"github.com/kenshaw/iv/ivctx"
)

// soffice is the LibreOffice headless command.
var soffice = cmd.New("soffice")

func init() {
	decoder.Register(
		"libreoffice",
		decoder.Desc("Office documents (via soffice)"),
		decoder.Extension(exts...),
		decoder.Matcher(match),
		decoder.MimeTypeExtensionMatch(
			"text/plain", "csv",
			"text/plain", "tsv",
		),
		decoder.Decoder(decode),
	)
}

// match reports whether the mime type is an office document.
func match(mime, ext string) bool {
	switch {
	case strings.HasPrefix(mime, "application/vnd.openxmlformats-officedocument."), // pptx, xlsx, ...
		strings.HasPrefix(mime, "application/vnd.ms-"),                 // ppt, xls, ...
		strings.HasPrefix(mime, "application/vnd.oasis.opendocument."), // odt, otp, odg, ...
		mime == "application/x-ole-storage",
		mime == "application/msword",
		mime == "text/rtf":
		return true
	case mime == "text/csv",
		mime == "text/tab-separated-values":
		// content sniffing reports these for any plain text whose lines
		// happen to parse with a consistent field count, which ordinary
		// markdown does often enough, so an extension naming something else
		// overrules the guess
		return ext == "" || slices.Contains(exts, ext)
	}
	return false
}

// exts are the extensions the decoder is registered for.
var exts = []string{
	"doc", "docx", "dot", "dotx",
	"xls", "xlsx", "xlt", "xltx",
	"ppt", "pptx", "pot", "potx", "pps", "ppsx",
	"odt", "ods", "odp", "odg", "odf", "odc",
	"ott", "ots", "otp", "otg",
	"rtf", "csv", "tsv", "pub", "vsd", "wpd",
}

// decode converts the document to a pdf, handing it back to the pipeline.
func decode(ctx context.Context, r io.Reader) (any, error) {
	src, cleanupSrc, err := cmd.Source(ctx, r, ivctx.FileExt(ivctx.PathName(ctx)))
	if err != nil {
		return nil, err
	}
	defer cleanupSrc()
	dir, cleanupDir, err := cmd.TempDir(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanupDir()
	if _, err := soffice.CombinedOutput(
		ctx,
		"--headless",
		"--convert-to", "pdf",
		"--outdir", dir,
		src,
	); err != nil {
		return nil, err
	}
	base := filepath.Base(src)
	pdfName := filepath.Join(dir, strings.TrimSuffix(base, filepath.Ext(base))+".pdf")
	ivctx.Logf(ctx, "soffice output: %s", pdfName)
	buf, err := os.ReadFile(pdfName)
	if err != nil {
		return nil, fmt.Errorf("soffice produced no pdf: %w", err)
	}
	return decoder.NewBytes("application/pdf", buf).WithExt("pdf"), nil
}
