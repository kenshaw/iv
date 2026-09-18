package ffmpeg

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"regexp"
	"strconv"
	"time"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/cmd"
	"github.com/kenshaw/iv/ivctx"
)

// The rate a video is sampled and played back at.
//
// What plays is a sampling rather than a playback -- see [animate] -- and
// there is nothing in one that twelve frames a second does not carry as well
// as thirty. What the terminal gets out of the difference is a steadier
// picture: every frame is a clear and an encode of a whole image, and the
// fewer of those there are the less the screen flickers under them.
const (
	sampleFPS   = 12
	sampleDelay = time.Second / sampleFPS
)

// How the frames are exported.
//
// A video sampled closely enough that most of it is played is read through
// once, with ffmpeg picking the frames out as it goes -- one decode of the
// file, which is as little work as sampling it can be done in.
//
// Past dense, that stops being true: an hour of video played in a minute is
// one frame in every five seconds, and reading the four fifths of it that will
// never be seen costs far more than seeking to the fifth that will. Those are
// seeked to instead, several at a time, each its own ffmpeg decoding forward
// from the keyframe it lands on. More than a handful of those only has them
// competing for the cores ffmpeg is already threading itself across.
const (
	dense   = time.Second
	workers = 4
)

// animate samples the video into the frames of an animation.
//
// The video is not played. A minute of it at [sampleFPS] is as much as a pass
// shows, so a video longer than that is sampled at even intervals across the
// whole of its length -- what plays is the shape of the thing from beginning
// to end rather than its first minute at the right speed -- and a shorter one
// is sampled densely enough to come out at its own length.
//
// The frames are exported while the animation is being displayed rather than
// ahead of it, so the first frame is on screen after one seek instead of after
// the whole video has been walked.
func animate(ctx context.Context, r io.Reader) (any, error) {
	src, cleanup, err := cmd.Source(ctx, r, ivctx.FileExt(ivctx.PathName(ctx)))
	if err != nil {
		return nil, err
	}
	v, err := probe(ctx, src)
	if err != nil {
		cleanup()
		return nil, err
	}
	delay := ivctx.AnimateDelay(ctx, sampleDelay)
	n := frameCount(v.Duration, delay)
	if n == 0 {
		cleanup()
		return nil, errors.New("cannot animate a video of unknown length")
	}
	// the frames are exported at the size they are displayed at: ffmpeg
	// scales as part of an export that is already running and already
	// parallel, where scaling them afterwards would be work on the way to the
	// screen
	sample := sampling{
		src:   src,
		scale: scaleArg(ctx, v.Width, v.Height),
		count: n,
		step:  v.Duration / time.Duration(n),
		delay: delay,
	}
	ivctx.Logf(ctx, "sampling %d frames every %v of %v at %v a frame (%s)", n, sample.step, v.Duration, delay, sample.scale)
	export := seek
	if sample.step < dense {
		export = read
	}
	return &decoder.Anim{
		Frames: decoder.Produce(func(ctx context.Context, s *decoder.Sink) error {
			return export(ctx, s, sample)
		}),
		Count:    n,
		Duration: time.Duration(n) * delay,
		Close: func() error {
			cleanup()
			return nil
		},
	}, nil
}

// sampling is the set of frames taken from a video: where they come from, how
// many there are, how far apart in the video they sit, and how long each is
// displayed for.
type sampling struct {
	src   string
	scale string
	count int
	step  time.Duration
	delay time.Duration
}

// frameCount returns how many frames are sampled from a video of the given
// length: enough to fill its own running time at the frame delay, capped at
// what [ivctx.AnimateMax] leaves room for. Returns 0 for a video whose length
// could not be read, there being nothing to spread the sampling over.
func frameCount(dur, delay time.Duration) int {
	if dur <= 0 || delay <= 0 {
		return 0
	}
	return max(1, min(int(dur/delay), int(ivctx.AnimateMax/delay)))
}

// read exports the frames by reading the video through once, ffmpeg picking
// them out at the sampling rate as it goes and writing them down a pipe.
//
// The pipe is what paces the read: ffmpeg fills it while the frames already
// through it are on screen, and is left waiting on it whenever it gets far
// enough ahead -- so a video that decodes faster than it plays, which most
// do, is read no faster than it is watched.
func read(ctx context.Context, s *decoder.Sink, v sampling) error {
	filter := fmt.Sprintf("fps=%f", float64(time.Second)/float64(v.step))
	if v.scale != "" {
		filter += "," + v.scale
	}
	return ffmpeg.Stream(
		ctx,
		func(r io.Reader) error {
			// ffmpeg writes the frames one after another with nothing between
			// them, and a png decode stops at the end of the image it read,
			// which leaves the reader at the start of the next one
			br := bufio.NewReader(r)
			for range v.count {
				img, err := png.Decode(br)
				if err != nil {
					return err
				}
				if !s.Send(img, v.delay) {
					return nil
				}
			}
			return nil
		},
		"-hide_banner",
		"-i", v.src,
		"-vf", filter,
		"-frames:v", strconv.Itoa(v.count),
		"-f", "image2pipe",
		"-c:v", "png",
		"-",
	)
}

// seek exports the frames by seeking to each of them, in order, several at a
// time.
//
// The frames are handed out to be exported ahead of where the display has got
// to, and collected back in the order they are played: a frame that took
// longer to reach than the ones after it holds them at the end of the queue
// rather than arriving out of sequence.
func seek(ctx context.Context, s *decoder.Sink, v sampling) error {
	type result struct {
		img image.Image
		err error
	}
	// a queue of pending frames, which is what bounds the export to workers
	// running at once: the frame after the last of them cannot be handed out
	// until room is made by one being displayed
	queue := make(chan chan result, workers)
	go func() {
		defer close(queue)
		for i := range v.count {
			ch := make(chan result, 1)
			select {
			case queue <- ch:
			case <-ctx.Done():
				return
			}
			go func(at time.Duration) {
				img, err := grab(ctx, v.src, v.scale, at)
				ch <- result{img, err}
			}(time.Duration(i) * v.step)
		}
	}()
	for ch := range queue {
		var res result
		select {
		case res = <-ch:
		case <-ctx.Done():
			return nil
		}
		if res.err != nil {
			return res.err
		}
		if !s.Send(res.img, v.delay) {
			return nil
		}
	}
	return nil
}

// grab exports the single frame at the timecode.
func grab(ctx context.Context, src, scale string, at time.Duration) (image.Image, error) {
	// -ss ahead of -i seeks the input rather than reading up to the timecode,
	// which is what makes a frame an hour in cost the same as the first
	params := []string{"-hide_banner", "-ss", seconds(at), "-i", src, "-vframes", "1", "-q:v", "1"}
	if scale != "" {
		params = append(params, "-vf", scale)
	}
	buf, err := ffmpeg.Run(ctx, append(params, "-f", "apng", "-")...)
	if err != nil {
		return nil, err
	}
	return png.Decode(bytes.NewReader(buf))
}

// scaleArg returns the ffmpeg filter scaling a w x h video to the size it is
// displayed at, and an empty string when it is displayed at its own size.
func scaleArg(ctx context.Context, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	if dw, dh := ivctx.Scale(ctx, w, h); dw != w || dh != h {
		return fmt.Sprintf("scale=%d:%d", dw, dh)
	}
	return ""
}

// seconds formats a timecode as the seconds ffmpeg takes for -ss. The MM:SS
// form the snapshot uses cannot say where in a second a frame is, and at
// thirty of them to the second that is the whole of the difference between one
// frame and the next.
func seconds(d time.Duration) string {
	return strconv.FormatFloat(max(d, 0).Seconds(), 'f', 3, 64)
}

// video is what ffprobe reports about a video.
type video struct {
	Duration      time.Duration
	Width, Height int
}

// probe reads the length and size of the video.
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
	}
	ivctx.Logf(ctx, "ffprobe: %dx%d, %v", v.Width, v.Height, v.Duration)
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
)
