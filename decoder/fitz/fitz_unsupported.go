//go:build 386 || arm

// Package fitz supplies a mupdf (fitz) decoder for iv, covering document
// formats such as epub, xps, mobi, and fb2.
//
// This is the 32-bit stub: go-fitz vendors a prebuilt libmupdf for 64-bit
// targets only, and its bindings pass a uint32 typedef where mupdf wants a
// size_t, which only type checks when the two are the same width. Nothing
// registers, so the formats below simply aren't decodable on these platforms.
//
// See: https://github.com/gen2brain/go-fitz
package fitz
