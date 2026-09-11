// Package ffmpeg supplies a video decoder for iv, snapshotting a frame with
// the `ffmpeg` command.
package ffmpeg

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/cmd"
	"github.com/kenshaw/iv/ivctx"
)

var (
	ffmpeg  = cmd.New("ffmpeg")
	ffprobe = cmd.New("ffprobe")
)

func init() {
	decoder.Register(
		"ffmpeg",
		decoder.Desc("Video snapshots (via ffmpeg)"),
		decoder.Extension(
			"mp4", "m4v", "mpeg", "mpeg2", "mpg", "mpg2", "mkv", "mov",
			"avi", "webm", "flv", "asf", "wmv", "3gp", "3g2", "mj2", "ogv",
		),
		decoder.MimeType("video/*"),
		decoder.Decoder(decode),
	)
}

// decode snapshots a single frame of the video.
func decode(ctx context.Context, r io.Reader) (any, error) {
	src, cleanup, err := cmd.Source(ctx, r, ivctx.FileExt(ivctx.PathName(ctx)))
	if err != nil {
		return nil, err
	}
	defer cleanup()
	tc := timecode(ctx, src)
	ivctx.Logf(ctx, "snapshot at %v", tc)
	buf, err := ffmpeg.Run(
		ctx,
		"-hide_banner",
		"-ss", tc,
		"-i", src,
		"-vframes", "1",
		"-q:v", "1",
		"-f", "apng",
		"-",
	)
	if err != nil {
		return nil, err
	}
	return decoder.NewBytes("image/png", buf).WithExt("png"), nil
}

// timecode returns the timecode to snapshot, either the configured time code
// or a position chosen from the video's duration.
func timecode(ctx context.Context, pathName string) string {
	if d := ivctx.Get(ctx).TimeCode; d != 0 {
		return formatTimecode(d)
	}
	if !ffprobe.Available() {
		return "00:00"
	}
	buf, err := ffprobe.CombinedOutput(ctx, "-loglevel", "quiet", "-show_format", pathName)
	if err != nil {
		return "00:00"
	}
	m := durationRE.FindStringSubmatch(string(buf))
	if m == nil {
		return "00:00"
	}
	f, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return "00:00"
	}
	dur := time.Duration(f * float64(time.Second))
	ivctx.Logf(ctx, "ffprobe duration: %v", dur)
	return durationTimecode(dur)
}

// durationTimecode returns a representative timecode for a video of the given
// duration -- far enough in to skip title cards, but well short of the end.
func durationTimecode(dur time.Duration) string {
	switch {
	case dur >= 1*time.Hour:
		return "10:00"
	case dur >= 30*time.Minute:
		return "05:00"
	case dur >= 15*time.Minute:
		return "03:00"
	case dur >= 5*time.Minute:
		return "02:00"
	case dur > 1*time.Minute:
		return "00:30"
	case dur > 30*time.Second:
		return "00:10"
	case dur > 5*time.Second:
		return "00:02"
	}
	return "00:00"
}

// durationRE matches ffprobe's duration output.
var durationRE = regexp.MustCompile(`(?m)^duration=(.*)$`)

// formatTimecode formats a duration in ffmpeg's MM:SS timecode format.
func formatTimecode(d time.Duration) string {
	if d <= 0 {
		return "00:00"
	}
	return fmt.Sprintf("%02d:%02d", int64(d/time.Minute), int64((d%time.Minute)/time.Second))
}
