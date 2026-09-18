package ffmpeg

import (
	"testing"
	"time"

	"github.com/kenshaw/iv/ivctx"
)

// sampleFrames is the most frames one pass of a video plays at the default
// rate.
const sampleFrames = int(ivctx.AnimateMax / sampleDelay)

// TestFrameCount checks the sampling: a video shorter than a pass is sampled
// at the playback rate so that it comes out at its own length, and a longer
// one is sampled across the whole of its length into the frames a pass has
// room for.
func TestFrameCount(t *testing.T) {
	for _, test := range []struct {
		name  string
		dur   time.Duration
		delay time.Duration
		exp   int
	}{
		{"unknown", 0, sampleDelay, 0},
		{"negative", -time.Second, sampleDelay, 0},
		{"no rate", time.Minute, 0, 0},
		{"shorter than a frame", time.Millisecond, sampleDelay, 1},
		{"one second", time.Second, sampleDelay, sampleFPS},
		{"ten seconds", 10 * time.Second, sampleDelay, 10 * sampleFPS},
		{"a pass", ivctx.AnimateMax, sampleDelay, sampleFrames},
		{"an hour", time.Hour, sampleDelay, sampleFrames},
		// --fps takes as many frames as the rate it forces asks for
		{"forced to one a second", 10 * time.Second, time.Second, 10},
		{"forced past a pass", time.Hour, time.Second, int(ivctx.AnimateMax / time.Second)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if n := frameCount(test.dur, test.delay); n != test.exp {
				t.Errorf("expected %d frames for %v at %v a frame, got %d", test.exp, test.dur, test.delay, n)
			}
		})
	}
}

// TestSampling checks that the sampling of a long video is spread across the
// whole of it, and that a pass of it runs for the time a pass runs for.
func TestSampling(t *testing.T) {
	const dur = 2 * time.Hour
	n := frameCount(dur, sampleDelay)
	step := dur / time.Duration(n)
	switch play := time.Duration(n) * sampleDelay; {
	case step < dense:
		t.Errorf("expected a two hour video to be seeked rather than read, step %v", step)
	case play > ivctx.AnimateMax:
		t.Errorf("expected a pass of at most %v, got %v", ivctx.AnimateMax, play)
	case time.Duration(n-1)*step < dur-2*step:
		t.Errorf("expected the sampling to reach the end of the video, stopped at %v of %v", time.Duration(n-1)*step, dur)
	}
}

func TestSeconds(t *testing.T) {
	for _, test := range []struct {
		d   time.Duration
		exp string
	}{
		{0, "0.000"},
		{-time.Second, "0.000"},
		{sampleDelay, "0.083"},
		{90 * time.Second, "90.000"},
		{time.Hour + 90*time.Millisecond, "3600.090"},
	} {
		if s := seconds(test.d); s != test.exp {
			t.Errorf("expected %q for %v, got %q", test.exp, test.d, s)
		}
	}
}

func TestScaleArg(t *testing.T) {
	c := ivctx.New()
	c.Width, c.Height = 480, 480
	ctx := ivctx.WithConfig(t.Context(), c)
	for _, test := range []struct {
		name string
		w, h int
		exp  string
	}{
		{"unknown size", 0, 0, ""},
		{"scaled down", 1920, 1080, "scale=480:270"},
		{"shown as it is", 320, 240, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if s := scaleArg(ctx, test.w, test.h); s != test.exp {
				t.Errorf("expected %q, got %q", test.exp, s)
			}
		})
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
