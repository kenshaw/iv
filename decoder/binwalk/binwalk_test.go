package binwalk

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kenshaw/iv/decoder"
	"github.com/ulikunitz/xz"
	// registers the decoders of the images that the walk keeps
	_ "github.com/kenshaw/iv/decoder/jpeg"
	_ "github.com/kenshaw/iv/decoder/png"
	"github.com/xo/magic/walk"
)

// pngSig is the signature that starts a png.
var pngSig = []byte("\x89PNG\r\n\x1a\n")

// TestDecode checks that the png in testdata/binwalk/icon.afdesign is found
// once, at its offset, as the binwalk command found it (D47).
func TestDecode(t *testing.T) {
	buf, err := os.ReadFile(filepath.Join("..", "..", "testdata", "binwalk", "icon.afdesign"))
	if err != nil {
		t.Fatalf("no test data: %v", err)
	}
	files := testDecode(t, buf)
	if !slices.Equal(files.Names, []string{"0001D279.png"}) {
		t.Fatalf("expected [0001D279.png], got %q", files.Names)
	}
	data, err := fs.ReadFile(files.FS, files.Names[0])
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !bytes.HasPrefix(data, pngSig) || len(data) != 13390 {
		t.Errorf("expected the 13390 bytes of the png, got %d bytes", len(data))
	}
}

// TestDecodeArchive checks that a png in a file of a zip is found, under the
// offset of the zip and the name of the file, as --extract of the binwalk
// command finds it.
func TestDecodeArchive(t *testing.T) {
	var img bytes.Buffer
	src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	src.Set(1, 1, color.NRGBA{R: 0xff, A: 0xff})
	if err := png.Encode(&img, src); err != nil {
		t.Fatal(err)
	}
	// the zip follows 256 bytes of nothing, as a file in a firmware image
	// does
	buf := bytes.NewBuffer(make([]byte, 256))
	zw := zip.NewWriter(buf)
	w, err := zw.Create("pics/a.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(img.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	files := testDecode(t, buf.Bytes())
	if !slices.Equal(files.Names, []string{"00000100/pics/a.png/00000000.png"}) {
		t.Fatalf("expected [00000100/pics/a.png/00000000.png], got %q", files.Names)
	}
	data, err := fs.ReadFile(files.FS, files.Names[0])
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !bytes.Equal(data, img.Bytes()) {
		t.Error("expected the bytes of the png in the zip")
	}
}

// TestDecodeXZ checks that a png in an xz stream is found, which needs the
// stream decoder of compress/xz. The stream follows 256 bytes, more than its
// own length: magic v0.6.0 missed an xz stream there, and v0.6.1 fixed it.
func TestDecodeXZ(t *testing.T) {
	var img bytes.Buffer
	if err := png.Encode(&img, image.NewNRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	buf := bytes.NewBuffer(make([]byte, 256))
	w, err := xz.NewWriter(buf)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(img.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	files := testDecode(t, buf.Bytes())
	if !slices.Equal(files.Names, []string{"00000100/decompressed.bin/00000000.png"}) {
		t.Errorf("expected [00000100/decompressed.bin/00000000.png], got %q", files.Names)
	}
}

// TestDecodeNoImages checks that a file with no image is an error.
func TestDecodeNoImages(t *testing.T) {
	if _, err := decode(context.Background(), bytes.NewReader(make([]byte, 4096))); !errors.Is(err, ErrNoImages) {
		t.Errorf("expected %v, got: %v", ErrNoImages, err)
	}
}

// TestImageExt checks the matches that the walk keeps: an image that a
// builtin decoder reads, and a RIFF only when it holds a WebP.
func TestImageExt(t *testing.T) {
	for _, test := range []struct {
		format string
		desc   string
		exp    string
		ok     bool
	}{
		{"png", "PNG image", "png", true},
		{"jpeg", "JPEG image", "jpg", true},
		{"riff", "RIFF image, encoding type: WAVE, total size: 44 bytes", "", false},
		{"svg", "SVG image", "", false},
		{"zip", "ZIP archive", "", false},
	} {
		t.Run(test.format, func(t *testing.T) {
			ext, ok := imageExt(&walk.Entry{Format: test.format, Description: test.desc})
			if ext != test.exp || ok != test.ok {
				t.Errorf("expected %q %t, got %q %t", test.exp, test.ok, ext, ok)
			}
		})
	}
}

// TestEntryName checks that the name of a member, which comes from the file,
// stays in the file set.
func TestEntryName(t *testing.T) {
	parent := &walk.Entry{Format: "zip", Offset: 0x10}
	e := &walk.Entry{Format: "png", Offset: 0x20, Parent: parent, Member: "../../x/a.png"}
	if name := entryName(e, "png"); name != "00000010/x/a.png/00000020.png" || !fs.ValidPath(name) {
		t.Errorf("expected 00000010/x/a.png/00000020.png, got %q", name)
	}
}

// testDecode decodes buf, and returns the file set of the images.
func testDecode(t *testing.T, buf []byte) *decoder.Files {
	t.Helper()
	res, err := decode(context.Background(), bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	files, ok := res.(*decoder.Files)
	if !ok {
		t.Fatalf("expected *decoder.Files, got %T", res)
	}
	return files
}
