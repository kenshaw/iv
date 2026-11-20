// Package jpeg2000 supplies a jpeg 2000 decoder for iv, in pure Go. It reads
// a jp2 file and a raw codestream.
//
// See: https://github.com/mrjoshuak/go-jpeg2000
package jpeg2000

import (
	"github.com/kenshaw/iv/decoder"
	"github.com/mrjoshuak/go-jpeg2000"
)

func init() {
	decoder.RegisterBuiltin(
		"jpeg2000",
		decoder.Desc("JPEG 2000"),
		decoder.Extension("jp2", "jpf", "jpx", "j2k", "j2c", "jpc"),
		decoder.MimeType("image/jp2", "image/jpx", "image/x-jp2-codestream"),
		decoder.ImageDecoder(jpeg2000.Decode),
	)
}
