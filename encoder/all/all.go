// Package all registers every iv encoder.
package all

import (
	_ "github.com/kenshaw/iv/encoder/jpeg"
	_ "github.com/kenshaw/iv/encoder/nativewebp"
	_ "github.com/kenshaw/iv/encoder/pdf"
	_ "github.com/kenshaw/iv/encoder/png"
	_ "github.com/kenshaw/iv/encoder/rasterm"
	_ "github.com/kenshaw/iv/encoder/vips"
)
