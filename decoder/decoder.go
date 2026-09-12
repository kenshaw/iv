// Package decoder provides the decoder registration and resolution mechanisms
// for iv.
//
// A decoder converts some input (a byte stream, or a bare string such as a
// data: URL) into something the pipeline knows how to finish: an image, a set
// of images, or another input to feed back through the pipeline.
package decoder

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/kenshaw/iv/ivctx"
)

// decoders are the registered decoders, in registration order.
var (
	mu       sync.RWMutex
	decoders []*Entry
	ordered  []*Entry // cached result of order(decoders)
)

// DecodeFunc decodes from a reader, returning a pipeline result. See [Resolve]
// for the recognized result types.
type DecodeFunc func(context.Context, io.Reader) (any, error)

// StringDecodeFunc decodes a bare string (a data: URL, a WIFI: code, a http
// URL), returning a pipeline result.
type StringDecodeFunc func(context.Context, string) (any, error)

// MatchFunc reports whether a decoder handles the mime type / file extension.
type MatchFunc func(ctx context.Context, mime, ext string) bool

// StringMatchFunc reports whether a string decoder handles the string.
type StringMatchFunc func(ctx context.Context, s string) bool

// DetectFunc sniffs a reader for a mime type, returning an empty string when
// the content is not recognized. It must not consume the reader.
type DetectFunc func(context.Context, io.Reader) (string, error)

// Entry is a registered iv image decoder.
type Entry struct {
	// Name is the unique decoder name.
	Name string
	// Desc is the human readable description.
	Desc string
	// Builtin indicates the decoder's formats are in Go's [image.Decode]
	// registry, and so can be rendered from inside archives and other
	// containers. The decoder may still supply a decode func of its own --
	// see [RegisterBuiltin].
	Builtin bool
	// IsString indicates a string decoder, matched against a bare command
	// line argument instead of a byte stream.
	IsString bool
	// Exts are the file extensions, without a leading dot.
	Exts []string
	// Mimes are the mime types, optionally with a trailing /* wildcard.
	Mimes []string
	// Fallback indicates a decoder of last resort, tried only once every
	// other decoder has declined or failed.
	Fallback bool

	before, after []string
	matchers      []MatchFunc
	strMatchers   []StringMatchFunc
	decode        DecodeFunc
	decodeString  StringDecodeFunc
	detect        DetectFunc

	initOnce  sync.Once
	initFunc  func(context.Context) (any, error)
	closeFunc func(context.Context) error
	state     any
	stateErr  error
	started   bool
	closed    bool
}

// String satisfies the [fmt.Stringer] interface.
func (d *Entry) String() string {
	if d.Desc != "" {
		return d.Name + " (" + d.Desc + ")"
	}
	return d.Name
}

// Register registers a decoder. It panics when name is empty or already
// registered.
func Register(name string, opts ...Option) *Entry {
	if name == "" {
		panic("decoder: name must not be empty")
	}
	d := &Entry{
		Name: name,
	}
	for _, o := range opts {
		o(d)
	}
	mu.Lock()
	defer mu.Unlock()
	if slices.ContainsFunc(decoders, func(v *Entry) bool { return v.Name == name }) {
		panic("decoder: " + name + " already registered")
	}
	ordered = nil
	decoders = append(decoders, d)
	return d
}

// RegisterBuiltin registers a decoder for a format that is in Go's
// [image.Decode] registry, which is what lets container formats such as comic
// archives render it directly -- see [BuiltinExt]. The format's package must
// be imported for its side effects.
//
// It decodes through [image.Decode] unless a [Decoder], [ImageDecoder], or
// [ImagesDecoder] option supplies a decode func of its own, which formats
// needing more than one image or a fallback do.
func RegisterBuiltin(name string, opts ...Option) *Entry {
	return Register(name, append([]Option{builtin()}, opts...)...)
}

// Unregister removes the named decoder. Returns whether it was registered. Not
// used by iv itself; provided for tests.
func Unregister(name string) bool {
	mu.Lock()
	defer mu.Unlock()
	i := slices.IndexFunc(decoders, func(v *Entry) bool { return v.Name == name })
	if i == -1 {
		return false
	}
	decoders, ordered = slices.Delete(decoders, i, i+1), nil
	return true
}

// Get returns the named decoder.
func Get(name string) (*Entry, bool) {
	mu.RLock()
	defer mu.RUnlock()
	i := slices.IndexFunc(decoders, func(v *Entry) bool { return v.Name == name })
	if i == -1 {
		return nil, false
	}
	return decoders[i], true
}

// All returns all registered decoders in resolution order.
func All() []*Entry {
	mu.Lock()
	defer mu.Unlock()
	if ordered == nil {
		ordered = order(decoders)
	}
	return slices.Clone(ordered)
}

// Match returns the decoders matching the mime type and file extension, in
// resolution order.
func Match(ctx context.Context, mime, ext string) []*Entry {
	var m []*Entry
	for _, d := range All() {
		if !d.IsString && d.match(ctx, mime, ext) {
			m = append(m, d)
		}
	}
	return m
}

// MatchString returns the string decoders matching s, in resolution order.
func MatchString(ctx context.Context, s string) []*Entry {
	var m []*Entry
	for _, d := range All() {
		if !d.IsString {
			continue
		}
		for _, f := range d.strMatchers {
			if f(ctx, s) {
				m = append(m, d)
				break
			}
		}
	}
	return m
}

// Extensions returns the sorted, deduplicated set of every registered file
// extension. Used to decide which files in a directory are worth opening.
func Extensions() []string {
	seen := make(map[string]struct{})
	for _, d := range All() {
		for _, ext := range d.Exts {
			seen[ext] = struct{}{}
		}
	}
	exts := make([]string, 0, len(seen))
	for ext := range seen {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	return exts
}

// SupportedExt reports whether the extension of the name is registered by any
// decoder.
func SupportedExt(name string) bool {
	ext := fileExt(name)
	if ext == "" {
		return false
	}
	for _, d := range All() {
		if slices.Contains(d.Exts, ext) {
			return true
		}
	}
	return false
}

// BuiltinExt reports whether the extension of the name is registered by a
// builtin ([image.Decode]) decoder. Containers such as comic archives use this
// to pick the entries they can render directly.
func BuiltinExt(name string) bool {
	ext := fileExt(name)
	if ext == "" {
		return false
	}
	for _, d := range All() {
		if d.Builtin && slices.Contains(d.Exts, ext) {
			return true
		}
	}
	return false
}

// match reports whether the decoder handles the mime type and extension.
func (d *Entry) match(ctx context.Context, mime, ext string) bool {
	for _, f := range d.matchers {
		if f(ctx, mime, ext) {
			return true
		}
	}
	for _, m := range d.Mimes {
		if MimeMatch(m, mime) {
			return true
		}
	}
	// a generic mime type carries no information, so fall back to the
	// extension
	return ext != "" && isGeneric(mime) && slices.Contains(d.Exts, ext)
}

// MimeMatch reports whether the mime type matches the pattern. A pattern may
// end in /* to match an entire type, and a "+suffix" (image/svg+xml) matches
// its unsuffixed form.
func MimeMatch(pattern, mime string) bool {
	pattern, mime = strings.ToLower(pattern), strings.ToLower(mime)
	if s, ok := strings.CutSuffix(pattern, "/*"); ok {
		typ, _, _ := strings.Cut(mime, "/")
		return typ == s
	}
	if pattern == mime {
		return true
	}
	p, _, _ := strings.Cut(pattern, "+")
	m, _, _ := strings.Cut(mime, "+")
	return p == m
}

// isGeneric reports whether the mime type is too generic to identify a
// decoder on its own.
//
// text/csv and text/tab-separated-values are here because they are guesses
// rather than readings: content sniffing reports them for any plain text
// whose lines parse with a consistent field count, so they say little more
// than text/plain does.
func isGeneric(mime string) bool {
	switch mime {
	case "", "application/octet-stream", "text/plain", "application/zip", "text/xml", "application/xml",
		"text/csv", "text/tab-separated-values":
		return true
	}
	return false
}

// state returns the decoder's initialized state, running the init func exactly
// once.
func (d *Entry) init(ctx context.Context) (any, error) {
	if d.initFunc == nil {
		return nil, nil
	}
	d.initOnce.Do(func() {
		d.started = true
		d.state, d.stateErr = d.initFunc(ctx)
	})
	return d.state, d.stateErr
}

// Close closes every decoder that was initialized, returning the joined
// errors.
func Close(ctx context.Context) error {
	var errs []error
	for _, d := range All() {
		if d.closeFunc == nil || !d.started || d.closed {
			continue
		}
		d.closed = true
		if err := d.closeFunc(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", d.Name, err))
		}
	}
	if len(errs) != 0 {
		return fmt.Errorf("decoder close: %w", errors.Join(errs...))
	}
	return nil
}

// order returns the decoders sorted so that every Before/After constraint is
// satisfied, disturbing registration order as little as possible: decoders are
// emitted in registration order, with each decoder's prerequisites pulled in
// ahead of it. Fallback decoders sort after everything else.
//
// Constraints naming an unregistered decoder are ignored. Constraints forming
// a cycle cannot all hold, so one of them is dropped -- deterministically, but
// which one is not specified.
func order(decoders []*Entry) []*Entry {
	// fallbacks come last, whatever their registration order
	decoders = slices.Clone(decoders)
	sort.SliceStable(decoders, func(i, j int) bool {
		return !decoders[i].Fallback && decoders[j].Fallback
	})
	n := len(decoders)
	idx := make(map[string]int, n)
	for i, d := range decoders {
		idx[d.Name] = i
	}
	// preds[i] holds the decoders that must be tried before i
	preds := make([][]int, n)
	for i, d := range decoders {
		for _, name := range d.after {
			if j, ok := idx[name]; ok && j != i {
				preds[i] = append(preds[i], j)
			}
		}
		for _, name := range d.before {
			if j, ok := idx[name]; ok && j != i {
				preds[j] = append(preds[j], i)
			}
		}
	}
	for i := range preds {
		slices.Sort(preds[i])
		preds[i] = slices.Compact(preds[i])
	}
	const (
		unvisited = iota
		visiting
		visited
	)
	state, res := make([]int, n), make([]*Entry, 0, n)
	var visit func(int)
	visit = func(i int) {
		// a decoder already on the stack means a cycle: leave it to be
		// emitted by the call that is walking it
		if state[i] != unvisited {
			return
		}
		state[i] = visiting
		for _, j := range preds[i] {
			visit(j)
		}
		state[i] = visited
		res = append(res, decoders[i])
	}
	for i := range n {
		visit(i)
	}
	return res
}

// fileExt returns the lower case extension of the name, without the leading
// dot.
func fileExt(name string) string {
	i := strings.LastIndexByte(name, '.')
	if i == -1 {
		return ""
	}
	return strings.ToLower(name[i+1:])
}

// builtinDecode decodes through Go's [image.Decode] registry.
func builtinDecode(ctx context.Context, r io.Reader) (any, error) {
	img, format, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	ivctx.Logf(ctx, "builtin decode: %s dimensions: %dx%d", format, b.Dx(), b.Dy())
	return img, nil
}
