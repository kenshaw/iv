package ffmpeg

import (
	"bytes"
	"errors"
	"image"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
)

// TestFrameCount checks that a video plays as many frames as fill its own
// length at the frame delay, however long it is (D43).
func TestFrameCount(t *testing.T) {
	const delay = time.Second / 24
	for _, test := range []struct {
		name  string
		dur   time.Duration
		delay time.Duration
		exp   int
	}{
		{"unknown", 0, delay, 0},
		{"negative", -time.Second, delay, 0},
		{"no rate", time.Minute, 0, 0},
		{"shorter than a frame", time.Millisecond, delay, 1},
		{"one second", time.Second, delay, 24},
		{"an hour", time.Hour, delay, 24 * 3600},
		// --fps takes as many frames as the rate it forces asks for
		{"forced to one a second", 10 * time.Second, time.Second, 10},
	} {
		t.Run(test.name, func(t *testing.T) {
			if n := frameCount(test.dur, test.delay); n != test.exp {
				t.Errorf("expected %d frames for %v at %v a frame, got %d", test.exp, test.dur, test.delay, n)
			}
		})
	}
}

// TestFrameDelay checks that a video plays at its own frame rate, up to
// [maxFPS], and at [maxFPS] when its rate is not known.
func TestFrameDelay(t *testing.T) {
	for _, test := range []struct {
		rate float64
		exp  time.Duration
	}{
		{24, time.Second / 24},
		{30, time.Second / 30},
		{60, time.Second / maxFPS},
		{0, time.Second / maxFPS},
	} {
		if d := frameDelay(test.rate); d != test.exp {
			t.Errorf("expected %v for %g fps, got %v", test.exp, test.rate, d)
		}
	}
}

// TestReadFrame checks that a frame is read as its raw RGBA pixels, and that
// a stream that ends inside a frame is the end of the video.
func TestReadFrame(t *testing.T) {
	buf := make([]byte, 2*2*4*2)
	for i := range buf {
		buf[i] = byte(i)
	}
	r := bytes.NewReader(buf[:len(buf)-1])
	img, err := readFrame(r, 2, 2)
	switch {
	case err != nil:
		t.Fatalf("expected no error, got: %v", err)
	case !bytes.Equal(img.Pix, buf[:16]) || img.Rect.Dx() != 2 || img.Rect.Dy() != 2:
		t.Errorf("expected the first 16 bytes as a 2x2 image, got %v %v", img.Rect, img.Pix)
	}
	if _, err := readFrame(r, 2, 2); !errors.Is(err, io.EOF) {
		t.Errorf("expected %v, got: %v", io.EOF, err)
	}
}

// TestProbeParsing checks that the video's length and size are read out of
// ffprobe's output, and that a stream carrying no length of its own does not
// hide the one the container recorded.
func TestProbeParsing(t *testing.T) {
	const out = `[STREAM]
width=1920
height=1080
duration=N/A
[/STREAM]
[FORMAT]
duration=95.500000
[/FORMAT]
`
	buf := []byte(out)
	switch d := duration(buf); {
	case d != 95500*time.Millisecond:
		t.Errorf("expected 95.5s, got %v", d)
	}
	switch w, h := number(widthRE, buf), number(heightRE, buf); {
	case w != 1920 || h != 1080:
		t.Errorf("expected 1920x1080, got %dx%d", w, h)
	}
	if d := duration([]byte("duration=N/A\n")); d != 0 {
		t.Errorf("expected no duration, got %v", d)
	}
}

// TestFrameRateParsing checks that the average frame rate is read as the
// fraction that ffprobe writes, and that the base rate is read when the
// average is not known.
func TestFrameRateParsing(t *testing.T) {
	for _, test := range []struct {
		out string
		exp float64
	}{
		{"r_frame_rate=30/1\navg_frame_rate=24000/1001\n", 24000.0 / 1001},
		{"r_frame_rate=25/1\navg_frame_rate=0/0\n", 25},
		{"r_frame_rate=0/0\navg_frame_rate=0/0\n", 0},
		{"", 0},
	} {
		if rate := frameRate([]byte(test.out)); rate != test.exp {
			t.Errorf("expected %g for %q, got %g", test.exp, test.out, rate)
		}
	}
}

// TestRotationParsing checks that a rotation is read in degrees from 0 to
// 359, as ffprobe writes it for a video that a phone recorded.
func TestRotationParsing(t *testing.T) {
	for _, test := range []struct {
		out string
		exp int
	}{
		{"rotation=90\n", 90},
		{"rotation=-90\n", 270},
		{"rotation=-90.000000\n", 270},
		{"rotation=180\n", 180},
		{"width=640\n", 0},
	} {
		if r := rotation([]byte(test.out)); r != test.exp {
			t.Errorf("expected %d for %q, got %d", test.exp, test.out, r)
		}
	}
}

// TestAnimate checks that a video plays every frame at its own frame rate and
// size, for its whole length (D43). testsrc2.mp4 is 10 s at 24 frames a
// second and 640x360.
func TestAnimate(t *testing.T) {
	anim := testAnimate(t, "testsrc2.mp4")
	switch {
	case !anim.Video:
		t.Error("expected a video")
	case anim.Count != 240:
		t.Errorf("expected 240 frames, got %d", anim.Count)
	case anim.Duration != 10*time.Second:
		t.Errorf("expected 10s, got %v", anim.Duration)
	}
	var n int
	for frame, err := range anim.Frames(t.Context()) {
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if size := frame.Image.Bounds().Size(); size != image.Pt(640, 360) {
			t.Fatalf("expected 640x360, got %v", size)
		}
		if frame.Delay != time.Second/24 {
			t.Fatalf("expected %v a frame, got %v", time.Second/24, frame.Delay)
		}
		n++
	}
	if n != anim.Count {
		t.Errorf("expected %d frames, got %d", anim.Count, n)
	}
}

// TestAnimateRotated checks that a video that is stored rotated by 90
// degrees, as a phone records one, plays upright. ffmpeg turns the frames
// upright, so each frame is 360x640.
func TestAnimateRotated(t *testing.T) {
	anim := testAnimate(t, "testsrc2-rotated.mp4")
	for frame, err := range anim.Frames(t.Context()) {
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if size := frame.Image.Bounds().Size(); size != image.Pt(360, 640) {
			t.Errorf("expected 360x640, got %v", size)
		}
		break
	}
}

// testAnimate returns the animation of the video in testdata/ffmpeg, with a
// display large enough that no frame is scaled. It closes the animation at
// the end of the test.
func testAnimate(t *testing.T, name string) *decoder.Anim {
	t.Helper()
	c := ivctx.New()
	c.Animate, c.Width, c.Height = true, 4096, 4096
	pathName := filepath.Join("..", "..", "testdata", "ffmpeg", name)
	ctx := ivctx.WithPathName(ivctx.WithConfig(t.Context(), c), pathName)
	f, err := os.Open(pathName)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer f.Close()
	res, err := animate(ctx, f)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	anim, ok := res.(*decoder.Anim)
	if !ok {
		t.Fatalf("expected *decoder.Anim, got %T", res)
	}
	t.Cleanup(func() {
		if err := anim.Close(); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
	return anim
}
