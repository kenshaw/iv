package ffmpeg

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"regexp"
	"strconv"
	"time"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/cmd"
	"github.com/kenshaw/iv/ivctx"
)

// maxFPS is the most frames a second that a video plays at (D43). A video
// with a higher rate, such as 60 frames a second, is decoded at this rate. A
// terminal does not draw more frames than this in a second, and the player
// drops each frame that it cannot draw in time.
const maxFPS = 30

// animate plays the video at its own speed, for its whole length (D43).
//
// ffmpeg decodes the video once, from the start, at the frame rate of the
// video up to [maxFPS], or at --fps, and scales each frame to the size that
// it is shown at. It writes the frames as raw RGBA pixels, which need no
// encode in ffmpeg and no decode in iv. The frames are made while the frames
// before them are on screen, so the first frame shows after one decode.
func animate(ctx context.Context, r io.Reader) (any, error) {
	src, cleanup, err := cmd.Source(ctx, r, ivctx.FileExt(ivctx.PathName(ctx)))
	if err != nil {
		return nil, err
	}
	v, err := probe(ctx, src)
	switch {
	case err != nil:
		cleanup()
		return nil, err
	case v.Duration <= 0:
		cleanup()
		return nil, errors.New("cannot animate a video of unknown length")
	case v.Width <= 0 || v.Height <= 0:
		cleanup()
		return nil, errors.New("cannot animate a video of unknown size")
	}
	delay := ivctx.AnimateDelay(ctx, frameDelay(v.Rate))
	n := frameCount(v.Duration, delay)
	// ffmpeg turns a rotated video upright before the filters, so the size
	// is that of the upright video
	w, h := v.Width, v.Height
	if v.Rotation%180 != 0 {
		w, h = h, w
	}
	w, h = ivctx.Scale(ctx, w, h)
	ivctx.Logf(ctx, "playing %d frames of %dx%d at %v a frame, for %v", n, w, h, delay, v.Duration)
	return &decoder.Anim{
		Frames: decoder.Produce(func(ctx context.Context, s *decoder.Sink) error {
			return read(ctx, s, src, w, h, n, delay)
		}),
		Count:    n,
		Duration: v.Duration,
		Video:    true,
		Close: func() error {
			cleanup()
			return nil
		},
	}, nil
}

// frameDelay returns how long one frame is shown for a video of the frame
// rate, at most [maxFPS] frames a second. A rate that is not known plays at
// [maxFPS].
func frameDelay(rate float64) time.Duration {
	if rate <= 0 || rate > maxFPS {
		rate = maxFPS
	}
	return time.Duration(float64(time.Second) / rate)
}

// frameCount returns how many frames a video of the length plays, at the
// frame delay. It returns 0 when the length or the delay is not known.
func frameCount(dur, delay time.Duration) int {
	if dur <= 0 || delay <= 0 {
		return 0
	}
	return max(1, int(dur/delay))
}

// read decodes the video with ffmpeg, and sends each frame to the playback.
//
// The pipe paces the decode. ffmpeg fills it while the frames before are on
// screen, and waits on it when it is far enough ahead. A frame is w x h raw
// RGBA pixels, with nothing between the frames.
func read(ctx context.Context, s *decoder.Sink, src string, w, h, n int, delay time.Duration) error {
	filter := fmt.Sprintf("fps=%f,scale=%d:%d", float64(time.Second)/float64(delay), w, h)
	return ffmpeg.Stream(
		ctx,
		func(r io.Reader) error {
			br := bufio.NewReaderSize(r, 1<<20)
			for range n {
				img, err := readFrame(br, w, h)
				switch {
				case errors.Is(err, io.EOF):
					// the length that ffprobe reports can be longer than
					// the frames that the video has
					return nil
				case err != nil:
					return err
				}
				if !s.Send(ctx, img, delay) {
					return nil
				}
			}
			return nil
		},
		"-hide_banner",
		"-loglevel", "error",
		"-i", src,
		"-an", "-sn",
		"-vf", filter,
		"-frames:v", strconv.Itoa(n),
		"-pix_fmt", "rgba",
		"-f", "rawvideo",
		"-",
	)
}

// readFrame reads one frame of w x h raw RGBA pixels from r. It returns
// [io.EOF] when r ends before the frame.
func readFrame(r io.Reader, w, h int) (*image.NRGBA, error) {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	switch _, err := io.ReadFull(r, img.Pix); {
	case errors.Is(err, io.ErrUnexpectedEOF):
		return nil, io.EOF
	case err != nil:
		return nil, err
	}
	return img, nil
}

// video is what ffprobe reports about a video.
type video struct {
	Duration      time.Duration
	Width, Height int
	// Rate is the frame rate, or 0 when it is not known.
	Rate float64
	// Rotation is the rotation in degrees, which ffmpeg undoes when it
	// decodes the video.
	Rotation int
}

// probe reads the length, the size, the frame rate and the rotation of the
// video.
//
// A container carries the length in its format header and each stream may
// carry one of its own, and neither is always there -- a stream copied from a
// live source has no header to have written one into -- so every duration
// reported is looked at and the first that reads as one is taken.
func probe(ctx context.Context, src string) (video, error) {
	if !ffprobe.Available() {
		_, err := ffprobe.Path()
		return video{}, err
	}
	buf, err := ffprobe.CombinedOutput(
		ctx,
		"-loglevel", "quiet",
		"-select_streams", "v:0",
		"-show_format",
		"-show_streams",
		src,
	)
	if err != nil {
		return video{}, err
	}
	v := video{
		Duration: duration(buf),
		Width:    number(widthRE, buf),
		Height:   number(heightRE, buf),
		Rate:     frameRate(buf),
		Rotation: rotation(buf),
	}
	ivctx.Logf(ctx, "ffprobe: %dx%d, %v, %g fps, rotation %d", v.Width, v.Height, v.Duration, v.Rate, v.Rotation)
	return v, nil
}

// duration returns the first duration in ffprobe's output that reads as one.
func duration(buf []byte) time.Duration {
	for _, m := range durationRE.FindAllSubmatch(buf, -1) {
		f, err := strconv.ParseFloat(string(m[1]), 64)
		if err == nil && f > 0 {
			return time.Duration(f * float64(time.Second))
		}
	}
	return 0
}

// frameRate returns the frame rate in ffprobe's output: the average rate,
// or the base rate when the average is not known. ffprobe writes a rate as a
// fraction, such as 24000/1001, and 0/0 when it is not known.
func frameRate(buf []byte) float64 {
	for _, re := range []*regexp.Regexp{avgRateRE, rateRE} {
		m := re.FindSubmatch(buf)
		if m == nil {
			continue
		}
		num, err1 := strconv.ParseFloat(string(m[1]), 64)
		den, err2 := strconv.ParseFloat(string(m[2]), 64)
		if err1 == nil && err2 == nil && num > 0 && den > 0 {
			return num / den
		}
	}
	return 0
}

// rotation returns the rotation in ffprobe's output, in degrees from 0 to
// 359, or 0 when there is none.
func rotation(buf []byte) int {
	m := rotationRE.FindSubmatch(buf)
	if m == nil {
		return 0
	}
	f, err := strconv.ParseFloat(string(m[1]), 64)
	if err != nil {
		return 0
	}
	return ((int(math.Round(f)) % 360) + 360) % 360
}

// number returns the value of the first field the regexp matches.
func number(re *regexp.Regexp, buf []byte) int {
	m := re.FindSubmatch(buf)
	if m == nil {
		return 0
	}
	n, err := strconv.Atoi(string(m[1]))
	if err != nil {
		return 0
	}
	return n
}

// the video stream fields read out of ffprobe's output.
var (
	widthRE  = regexp.MustCompile(`(?m)^width=(\d+)$`)
	heightRE = regexp.MustCompile(`(?m)^height=(\d+)$`)
	// the rates are fractions, and the rotation is a number of degrees that
	// can be negative or have a fraction
	avgRateRE  = regexp.MustCompile(`(?m)^avg_frame_rate=(\d+)/(\d+)$`)
	rateRE     = regexp.MustCompile(`(?m)^r_frame_rate=(\d+)/(\d+)$`)
	rotationRE = regexp.MustCompile(`(?m)^rotation=(-?[\d.]+)$`)
)
