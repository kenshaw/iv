// Package tag supplies an audio metadata decoder for iv, rendering a card of
// the track's cover art, tags and waveform.
//
// See: https://github.com/dhowden/tag
package tag

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dhowden/tag"
	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/cmd"
	"github.com/kenshaw/iv/ivctx"
)

func init() {
	decoder.Register(
		"tag",
		decoder.Desc("Audio metadata (cover art, tags, waveform)"),
		decoder.Extension("mp3", "m4a", "m4b", "m4p", "flac", "ogg", "oga", "dsf", "aac"),
		decoder.MimeType("audio/*"),
		decoder.Decoder(decode),
	)
}

// decode renders the track as a card: cover art, tags, and the waveform
// ffmpeg decodes from the audio. The waveform is the only part that needs an
// external tool, so a missing ffmpeg costs the bars and nothing else.
func decode(ctx context.Context, r io.Reader) (any, error) {
	rs, ok := r.(io.ReadSeeker)
	if !ok {
		return nil, fmt.Errorf("%T: not seekable", r)
	}
	md, err := tag.ReadFrom(rs)
	if err != nil {
		return nil, err
	}
	ivctx.Logf(ctx, "tag format: %s file type: %s", md.Format(), md.FileType())
	c := &card{
		title:  md.Title(),
		artist: firstOf(md.Artist(), md.AlbumArtist(), md.Composer()),
		album:  album(md),
	}
	if pic := md.Picture(); pic != nil {
		ivctx.Logf(ctx, "tag picture: %s %s %d bytes", pic.Type, pic.MIMEType, len(pic.Data))
		c.art, c.artMime = pic.Data, pic.MIMEType
	}
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek: %w", err)
	}
	pathName := ivctx.PathName(ctx)
	if c.art == nil {
		// nothing embedded, so fall back to the album's cover sitting next to
		// the audio, which is where a ripper leaves it
		c.art, c.artMime = sidecar(ctx, pathName)
	}
	if c.title == "" {
		c.title = strings.TrimSuffix(filepath.Base(pathName), filepath.Ext(pathName))
	}
	src, cleanup, err := cmd.Source(ctx, rs, ivctx.FileExt(pathName))
	if err != nil {
		return nil, err
	}
	defer cleanup()
	s := probe(ctx, src)
	c.duration = s.duration
	c.meta = meta(md, s)
	c.accent = accentFor(c.art, c.artist+c.title)
	c.peaks = peaks(ctx, src, s.duration, bars)
	ivctx.Logf(ctx, "tag card: accent %s waveform %t", c.accent.hex(1, 1), c.peaks != nil)
	return decoder.NewBytes("image/svg+xml", c.svg()).WithExt("svg"), nil
}

// album returns the album line: the album, with the year and track position
// when the tags carry them.
func album(md tag.Metadata) string {
	s := md.Album()
	if y := md.Year(); y != 0 {
		s = join(s, strconv.Itoa(y))
	}
	if n, total := md.Track(); n != 0 {
		t := "track " + strconv.Itoa(n)
		if total != 0 {
			t += " of " + strconv.Itoa(total)
		}
		s = join(s, t)
	}
	return s
}

// meta returns the technical line: what the file is, and how it was encoded.
func meta(md tag.Metadata, s stream) string {
	var parts []string
	if t := string(md.FileType()); t != "" {
		parts = append(parts, t)
	}
	if s.codec != "" && !strings.EqualFold(s.codec, string(md.FileType())) {
		parts = append(parts, s.codec)
	}
	if s.sampleRate != 0 {
		parts = append(parts, strings.TrimSuffix(fmt.Sprintf("%.1f", float64(s.sampleRate)/1000), ".0")+" kHz")
	}
	switch s.channels {
	case 1:
		parts = append(parts, "mono")
	case 2:
		parts = append(parts, "stereo")
	case 0:
	default:
		parts = append(parts, strconv.Itoa(s.channels)+" channels")
	}
	if s.bitRate != 0 {
		parts = append(parts, strconv.Itoa(s.bitRate/1000)+" kbps")
	}
	if g := md.Genre(); g != "" {
		parts = append(parts, g)
	}
	return join(parts...)
}

// join joins the non-empty parts with a middot.
func join(parts ...string) string {
	var out []string
	for _, s := range parts {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return strings.Join(out, " · ")
}

// firstOf returns the first non-empty string.
func firstOf(s ...string) string {
	for _, v := range s {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}
