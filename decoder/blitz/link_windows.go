//go:build windows

package blitz

// Two rust static archives end up in the same binary, blitz and resvg, and
// each carries its own copy of the rust runtime: the compiler-builtins soft
// float and i128 helpers, the libm shims, the win32 wake-by-address stubs,
// and rust_eh_personality. 151 symbols overlap, every one of them a runtime
// helper with a single meaning rather than two libraries disagreeing, and
// none of them from either library's own code.
//
// The PE linker rejects a duplicate definition outright where ELF and mach-o
// quietly keep the first, so linking the two is a windows-only failure. Tell
// it to keep the first of each, which is what the other two platforms do
// already.
//
// This belongs here rather than in blitz or resvg: neither library is broken
// on its own, and the flag turns off a real linker check for the whole link,
// which no consumer of just one of them should have to wear.

/*
#cgo LDFLAGS: -Wl,--allow-multiple-definition
*/
import "C"
