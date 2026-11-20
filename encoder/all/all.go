// Package all registers every iv encoder.
package all

import (
	// each package registers its encoder from init
	_ "github.com/kenshaw/iv/encoder/avif"
	_ "github.com/kenshaw/iv/encoder/gif"
	_ "github.com/kenshaw/iv/encoder/heic"
	_ "github.com/kenshaw/iv/encoder/jpeg"
	_ "github.com/kenshaw/iv/encoder/jpeg2000"
	_ "github.com/kenshaw/iv/encoder/jxl"
	_ "github.com/kenshaw/iv/encoder/nativewebp"
	_ "github.com/kenshaw/iv/encoder/pdf"
	_ "github.com/kenshaw/iv/encoder/png"
	_ "github.com/kenshaw/iv/encoder/rasterm"
	_ "github.com/kenshaw/iv/encoder/tiff"
	_ "github.com/kenshaw/iv/encoder/vpx"
)
