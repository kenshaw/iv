// Package cmd provides lazily resolved external commands for iv decoders that
// shell out (ffmpeg, soffice, mmdc, binwalk).
package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kenshaw/iv/ivctx"
)

// errLen is how much of a command's stderr is included in an error.
const errLen = 200

// Cmd is an external command, resolved in $PATH on first use.
type Cmd struct {
	name string
	once sync.Once
	path string
	err  error
}

// New creates a lazily resolved external command.
func New(name string) *Cmd {
	return &Cmd{
		name: name,
	}
}

// Name returns the command name.
func (c *Cmd) Name() string {
	return c.name
}

// Path returns the resolved path of the command.
func (c *Cmd) Path() (string, error) {
	c.once.Do(func() {
		if c.path, c.err = exec.LookPath(c.name); c.err != nil {
			c.err = fmt.Errorf("%s not in path: %w", c.name, c.err)
		}
	})
	return c.path, c.err
}

// Available reports whether the command is in $PATH.
func (c *Cmd) Available() bool {
	_, err := c.Path()
	return err == nil
}

// Run executes the command, returning its standard output. Any error includes
// the leading portion of standard error.
func (c *Cmd) Run(ctx context.Context, params ...string) ([]byte, error) {
	path, err := c.Path()
	if err != nil {
		return nil, err
	}
	ivctx.Logf(ctx, "executing: %s %s", path, strings.Join(params, " "))
	start := time.Now()
	cmd := exec.CommandContext(ctx, path, params...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	for s := range strings.SplitSeq(strings.TrimSpace(stderr.String()), "\n") {
		if s != "" {
			ivctx.Logf(ctx, "%s: %s", c.name, s)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w: %s", c.name, err, elide(stderr.String()))
	}
	ivctx.Logf(ctx, "%s: %v", c.name, time.Since(start))
	return stdout.Bytes(), nil
}

// CombinedOutput executes the command, returning its combined output.
func (c *Cmd) CombinedOutput(ctx context.Context, params ...string) ([]byte, error) {
	path, err := c.Path()
	if err != nil {
		return nil, err
	}
	ivctx.Logf(ctx, "executing: %s %s", path, strings.Join(params, " "))
	start := time.Now()
	buf, err := exec.CommandContext(ctx, path, params...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %w: %s", c.name, err, elide(string(buf)))
	}
	ivctx.Logf(ctx, "%s: %v", c.name, time.Since(start))
	return buf, nil
}

// elide trims output for inclusion in an error message.
func elide(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > errLen {
		return s[:errLen] + "..."
	}
	return s
}

// Source returns a path on disk holding the contents of r, along with a
// cleanup func. When the decode target is already a file, its path is used
// as-is and cleanup is a no-op; otherwise the stream is spooled to a temp file
// with the given extension.
func Source(ctx context.Context, r io.Reader, ext string) (string, func(), error) {
	if pathName := ivctx.PathName(ctx); pathName != "" {
		if _, err := os.Stat(pathName); err == nil {
			return pathName, func() {}, nil
		}
	}
	dir, err := os.MkdirTemp("", "iv.")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() {
		ivctx.Logf(ctx, "removing: %s", dir)
		_ = os.RemoveAll(dir)
	}
	name := filepath.Join(dir, "source")
	if ext != "" {
		name += "." + ext
	}
	f, err := os.Create(name)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	_, err = io.Copy(f, r)
	if err := errors.Join(err, f.Close()); err != nil {
		cleanup()
		return "", nil, err
	}
	ivctx.Logf(ctx, "spooled to: %s", name)
	return name, cleanup, nil
}

// TempDir creates a temp directory and returns it with a cleanup func.
func TempDir(ctx context.Context) (string, func(), error) {
	dir, err := os.MkdirTemp("", "iv.")
	if err != nil {
		return "", nil, err
	}
	ivctx.Logf(ctx, "temp dir: %s", dir)
	return dir, func() {
		ivctx.Logf(ctx, "removing: %s", dir)
		_ = os.RemoveAll(dir)
	}, nil
}
