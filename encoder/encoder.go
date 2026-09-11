// Package encoder provides the encoder registration mechanisms for iv.
//
// An encoder writes a decoded image to a writer, either as a file format (png,
// jpeg, webp) or as terminal graphics.
package encoder

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"slices"
	"sort"
	"sync"
)

var (
	// ErrNotRegistered is returned when no encoder is registered under a name.
	ErrNotRegistered = errors.New("encoder not registered")
	// ErrUnsupportedFormat is returned when an encoder is registered but the
	// underlying library was not built with support for its output format.
	ErrUnsupportedFormat = errors.New("format not supported by this build")
)

// EncodeFunc encodes an image to a writer.
type EncodeFunc func(context.Context, io.Writer, image.Image) error

var (
	mu       sync.RWMutex
	encoders = make(map[string]*Entry)
)

// Entry is a registered iv image encoder.
type Entry struct {
	// Name is the unique encoder name.
	Name string
	// Desc is the human readable description.
	Desc string
	// Ext is the output file extension, without a leading dot. Empty for
	// terminal encoders.
	Ext string
	// Term indicates the encoder writes terminal graphics rather than a file
	// format.
	Term bool

	encode EncodeFunc
}

// String satisfies the [fmt.Stringer] interface.
func (e *Entry) String() string {
	if e.Desc != "" {
		return e.Name + " (" + e.Desc + ")"
	}
	return e.Name
}

// Encode encodes the image to the writer.
func (e *Entry) Encode(ctx context.Context, w io.Writer, img image.Image) error {
	if e.encode == nil {
		return fmt.Errorf("encoder %q: has no encode func", e.Name)
	}
	return e.encode(ctx, w, img)
}

// Register registers an encoder. It panics when name is empty or already
// registered.
func Register(name string, opts ...Option) *Entry {
	if name == "" {
		panic("encoder: name must not be empty")
	}
	e := &Entry{
		Name: name,
		Ext:  name,
	}
	for _, o := range opts {
		o(e)
	}
	mu.Lock()
	defer mu.Unlock()
	if _, ok := encoders[name]; ok {
		panic("encoder: " + name + " already registered")
	}
	encoders[name] = e
	return e
}

// Unregister removes the named encoder. Returns whether it was registered. Not
// used by iv itself; provided for tests.
func Unregister(name string) bool {
	mu.Lock()
	defer mu.Unlock()
	_, ok := encoders[name]
	delete(encoders, name)
	return ok
}

// Get returns the named encoder.
func Get(name string) (*Entry, bool) {
	mu.RLock()
	defer mu.RUnlock()
	e, ok := encoders[name]
	return e, ok
}

// All returns every registered encoder, ordered by name.
func All() []*Entry {
	mu.RLock()
	defer mu.RUnlock()
	res := make([]*Entry, 0, len(encoders))
	for _, e := range encoders {
		res = append(res, e)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	return res
}

// Names returns the sorted names of every registered encoder.
func Names() []string {
	res := All()
	names := make([]string, len(res))
	for i, e := range res {
		names[i] = e.Name
	}
	return names
}

// Encode encodes the image to the writer using the named encoder.
func Encode(ctx context.Context, w io.Writer, name string, img image.Image) error {
	e, ok := Get(name)
	if !ok {
		return fmt.Errorf("%q: %w", name, ErrNotRegistered)
	}
	return e.Encode(ctx, w, img)
}

// ForExt returns the encoder producing the file extension, without a leading
// dot.
func ForExt(ext string) (*Entry, bool) {
	for _, e := range All() {
		if !e.Term && e.Ext == ext {
			return e, true
		}
	}
	return nil, false
}

// TermEncoders returns the terminal graphics encoders, ordered by name.
func TermEncoders() []*Entry {
	return slices.DeleteFunc(All(), func(e *Entry) bool {
		return !e.Term
	})
}
