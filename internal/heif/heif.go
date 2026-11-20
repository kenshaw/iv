// Package heif reads the top-level images of a HEIF file, such as a heic or
// an avif, and makes one of them the primary image (D58). It also reads the
// metadata of a dynamic wallpaper of macOS (D59).
//
// A HEIF file can hold several images, such as the photos of a burst. The
// decoders of heic and avif decode only the primary image, which the pitm box
// names. [WithPrimary] returns a copy of the file whose pitm box names another
// image, so that the decoder decodes it.
//
// An image item is top level, as libheif reads it, when it is not hidden, and
// is not a thumbnail, an auxiliary image, such as an alpha or a depth map, or
// a tile of a grid. The top-level images are in the order of the item
// information box.
//
// See: ISO/IEC 23008-12
package heif

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/kenshaw/iv/ivctx"
)

// ErrFormat is the error of a file whose boxes are not valid.
var ErrFormat = errors.New("not a valid heif file")

// imageTypes are the item types of an image.
var imageTypes = []string{"hvc1", "av01", "grid", "iden", "iovl", "jpeg", "j2k1", "unci", "avc1", "vvc1"}

// File holds the items of a HEIF file that select its images.
type File struct {
	// Primary is the item ID of the primary image.
	Primary uint32
	// Images are the item IDs of the top-level images, in the order of the
	// file.
	Images []uint32

	buf []byte
	// pitm is the offset of the item ID in the pitm box, and wide is true
	// when it has 4 bytes, not 2
	pitm int
	wide bool
}

// Parse reads the meta box of the HEIF file in buf.
func Parse(buf []byte) (*File, error) {
	meta, ok := child(buf, 0, len(buf), "meta")
	if !ok {
		return nil, fmt.Errorf("meta box: %w", ErrFormat)
	}
	// meta is a full box, with 4 bytes of version and flags
	start, end := meta.body+4, meta.end
	if start > end {
		return nil, fmt.Errorf("meta box: %w", ErrFormat)
	}
	f := &File{buf: buf}
	if err := f.readPrimary(start, end); err != nil {
		return nil, err
	}
	items, err := readItems(buf, start, end)
	if err != nil {
		return nil, err
	}
	excluded, err := readReferences(buf, start, end)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		if it.hidden || excluded[it.id] || !slices.Contains(imageTypes, it.typ) {
			continue
		}
		f.Images = append(f.Images, it.id)
	}
	if !slices.Contains(f.Images, f.Primary) {
		return nil, fmt.Errorf("primary item %d: %w", f.Primary, ErrFormat)
	}
	return f, nil
}

// WithPrimary returns a copy of the file whose primary image is the item id.
func (f *File) WithPrimary(id uint32) ([]byte, error) {
	if !slices.Contains(f.Images, id) {
		return nil, fmt.Errorf("item %d: not a top-level image: %w", id, ErrFormat)
	}
	buf := slices.Clone(f.buf)
	switch {
	case f.wide:
		binary.BigEndian.PutUint32(buf[f.pitm:], id)
	case id <= 0xffff:
		binary.BigEndian.PutUint16(buf[f.pitm:], uint16(id))
	default:
		return nil, fmt.Errorf("item %d: too large for the pitm box: %w", id, ErrFormat)
	}
	return buf, nil
}

// readPrimary reads the pitm box.
func (f *File) readPrimary(start, end int) error {
	b, ok := child(f.buf, start, end, "pitm")
	if !ok || b.body+6 > b.end {
		return fmt.Errorf("pitm box: %w", ErrFormat)
	}
	f.pitm, f.wide = b.body+4, f.buf[b.body] != 0
	switch {
	case f.wide && f.pitm+4 <= b.end:
		f.Primary = binary.BigEndian.Uint32(f.buf[f.pitm:])
	case f.wide:
		return fmt.Errorf("pitm box: %w", ErrFormat)
	default:
		f.Primary = uint32(binary.BigEndian.Uint16(f.buf[f.pitm:]))
	}
	return nil
}

// item is an entry of the item information box.
type item struct {
	id     uint32
	typ    string
	hidden bool
}

// readItems reads the entries of the iinf box.
func readItems(buf []byte, start, end int) ([]item, error) {
	iinf, ok := child(buf, start, end, "iinf")
	if !ok {
		return nil, fmt.Errorf("iinf box: %w", ErrFormat)
	}
	// the version says whether the count has 2 bytes or 4
	off := iinf.body + 4 + 2
	if iinf.body < len(buf) && buf[iinf.body] != 0 {
		off += 2
	}
	var items []item
	for b := range boxes(buf, off, iinf.end) {
		if b.typ != "infe" || b.body+4 > b.end {
			continue
		}
		version, hidden := buf[b.body], buf[b.body+3]&1 != 0
		p := b.body + 4
		var it item
		switch version {
		case 2:
			if p+8 > b.end {
				return nil, fmt.Errorf("infe box: %w", ErrFormat)
			}
			it.id, p = uint32(binary.BigEndian.Uint16(buf[p:])), p+2
		case 3:
			if p+10 > b.end {
				return nil, fmt.Errorf("infe box: %w", ErrFormat)
			}
			it.id, p = binary.BigEndian.Uint32(buf[p:]), p+4
		default:
			// an entry of version 0 or 1 has no item type, and is no image
			continue
		}
		// the index of the item protection, then the item type
		it.typ, it.hidden = string(buf[p+2:p+6]), hidden
		items = append(items, it)
	}
	return items, nil
}

// readReferences returns the item IDs that the iref box shows are not top
// level: a thumbnail or an auxiliary image refers to its image, and a grid
// refers to its tiles.
func readReferences(buf []byte, start, end int) (map[uint32]bool, error) {
	excluded := make(map[uint32]bool)
	iref, ok := child(buf, start, end, "iref")
	if !ok {
		return excluded, nil
	}
	if iref.body+4 > iref.end {
		return nil, fmt.Errorf("iref box: %w", ErrFormat)
	}
	size := 2
	if buf[iref.body] != 0 {
		size = 4
	}
	id := func(p int) uint32 {
		if size == 4 {
			return binary.BigEndian.Uint32(buf[p:])
		}
		return uint32(binary.BigEndian.Uint16(buf[p:]))
	}
	for b := range boxes(buf, iref.body+4, iref.end) {
		p := b.body
		if p+size+2 > b.end {
			return nil, fmt.Errorf("iref box %s: %w", b.typ, ErrFormat)
		}
		from, count := id(p), int(binary.BigEndian.Uint16(buf[p+size:]))
		p += size + 2
		if p+count*size > b.end {
			return nil, fmt.Errorf("iref box %s: %w", b.typ, ErrFormat)
		}
		switch b.typ {
		case "thmb", "auxl":
			excluded[from] = true
		case "dimg", "base":
			for i := range count {
				excluded[id(p+i*size)] = true
			}
		}
	}
	return excluded, nil
}

// box is a box of the file: its type, the offset of its body, and its end.
type box struct {
	typ  string
	body int
	end  int
}

// child returns the first box of the type among the boxes of buf[start:end].
func child(buf []byte, start, end int, typ string) (box, bool) {
	for b := range boxes(buf, start, end) {
		if b.typ == typ {
			return b, true
		}
	}
	return box{}, false
}

// boxes yields the boxes of buf[start:end], at one level. It stops at a box
// whose size does not fit.
func boxes(buf []byte, start, end int) func(func(box) bool) {
	return func(yield func(box) bool) {
		end = min(end, len(buf))
		for i := start; i+8 <= end; {
			n, head := int(binary.BigEndian.Uint32(buf[i:])), 8
			switch n {
			case 0:
				n = end - i
			case 1:
				if i+16 > end {
					return
				}
				// a box of more than 2 GiB, such as the media data of a
				// long sequence, ends the scan. The meta box is small
				m := binary.BigEndian.Uint64(buf[i+8:])
				if m > math.MaxInt32 {
					return
				}
				n, head = int(m), 16
			}
			if n < head || i+n > end {
				return
			}
			if !yield(box{typ: string(buf[i+4 : i+8]), body: i + head, end: i + n}) {
				return
			}
			i += n
		}
	}
}

// Select returns buf with the image that --page selects as its primary image,
// with the index of that image among the top-level images of the file, from
// 0, and the count of the top-level images.
//
// Without --page, or with a page past the last image, buf is returned as it
// is, and the decoder decodes the primary image. A file that the package
// cannot read is returned as it is too, with the index -1, and the decoder
// reports its error.
func Select(ctx context.Context, buf []byte) ([]byte, int, int) {
	f, err := Parse(buf)
	if err != nil {
		ivctx.Logf(ctx, "heif: %v", err)
		return buf, -1, 0
	}
	primary := slices.Index(f.Images, f.Primary)
	page := int(ivctx.Get(ctx).Page)
	switch {
	case page == 0:
		return buf, primary, len(f.Images)
	case page > len(f.Images):
		ivctx.Logf(ctx, "heif: page %d of %d: the primary image", page, len(f.Images))
		return buf, primary, len(f.Images)
	}
	id := f.Images[page-1]
	out, err := f.WithPrimary(id)
	if err != nil {
		ivctx.Logf(ctx, "heif: %v", err)
		return buf, primary, len(f.Images)
	}
	ivctx.Logf(ctx, "heif: page %d of %d is item %d", page, len(f.Images), id)
	return out, page - 1, len(f.Images)
}
