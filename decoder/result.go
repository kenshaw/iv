package decoder

import (
	"bytes"
	"io"
	"io/fs"
)

// A decoder returns one of the following, which the pipeline resolves into a
// final image:
//
//   - [image.Image], the decoded image
//   - []image.Image, a set of images from which the configured page is taken
//   - [*Image], a new mime type and stream to feed back through the pipeline
//   - [*Handoff], a hand off to a specific named decoder
//   - [*Files], a set of files from which the configured page is taken

// Image is a mime type and stream to run back through the decoding pipeline.
// Used by decoders that transform their input into another format, such as
// graphviz rendering dot to svg.
type Image struct {
	Mime   string
	Ext    string
	Reader io.Reader
}

// NewImage creates an image for the decoding pipeline to retry.
func NewImage(mime string, r io.Reader) *Image {
	return &Image{
		Mime:   mime,
		Reader: r,
	}
}

// NewBytes creates an image for the decoding pipeline to retry from a byte
// slice.
func NewBytes(mime string, buf []byte) *Image {
	return NewImage(mime, bytes.NewReader(buf))
}

// WithExt sets the file extension reported to the pipeline on retry.
func (img *Image) WithExt(ext string) *Image {
	img.Ext = ext
	return img
}

// Handoff hands decoding of the same stream to a specific decoder.
type Handoff struct {
	Name string
}

// Next creates a hand off to the named decoder.
func Next(name string) *Handoff {
	return &Handoff{
		Name: name,
	}
}

// Files is a set of named files within a file system, from which the
// configured page is decoded. Used by container formats such as comic
// archives.
type Files struct {
	FS    fs.FS
	Names []string
}

// FS creates a file set for the decoding pipeline.
func FS(fsys fs.FS, names ...string) *Files {
	return &Files{
		FS:    fsys,
		Names: names,
	}
}
