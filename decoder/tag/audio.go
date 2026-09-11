package tag

import (
	"context"
	"encoding/binary"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kenshaw/iv/internal/cmd"
	"github.com/kenshaw/iv/ivctx"
)

var (
	ffmpeg  = cmd.New("ffmpeg")
	ffprobe = cmd.New("ffprobe")
)

// stream is what ffprobe reports about the audio stream.
type stream struct {
	codec      string
	duration   time.Duration
	sampleRate int
	channels   int
	bitRate    int
}

// probe asks ffprobe about the file's audio stream. Everything it reports is
// decoration on the card, so a missing ffprobe is not an error.
func probe(ctx context.Context, pathName string) stream {
	var s stream
	if !ffprobe.Available() {
		return s
	}
	buf, err := ffprobe.CombinedOutput(
		ctx,
		"-loglevel", "quiet",
		"-select_streams", "a:0",
		"-show_entries", "stream=codec_name,sample_rate,channels,bit_rate",
		"-show_entries", "format=duration,bit_rate",
		pathName,
	)
	if err != nil {
		return s
	}
	for _, m := range entryRE.FindAllStringSubmatch(string(buf), -1) {
		k, v := m[1], strings.TrimSpace(m[2])
		if v == "" || v == "N/A" {
			continue
		}
		switch n, _ := strconv.Atoi(v); k {
		case "codec_name":
			s.codec = v
		case "sample_rate":
			s.sampleRate = n
		case "channels":
			s.channels = n
		case "bit_rate":
			// the stream reports first and wins; the format total is the
			// fallback for containers that don't carry a per-stream rate
			if s.bitRate == 0 {
				s.bitRate = n
			}
		case "duration":
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				s.duration = time.Duration(f * float64(time.Second))
			}
		}
	}
	ivctx.Logf(ctx, "ffprobe: %s %d Hz %dch %d bps %v", s.codec, s.sampleRate, s.channels, s.bitRate, s.duration)
	return s
}

// entryRE matches ffprobe's key=value output.
var entryRE = regexp.MustCompile(`(?m)^(\w+)=(.*)$`)

// Bounds on the rate ffmpeg is asked to resample to. The card only needs an
// envelope, so the rate is chosen to keep the decoded pcm small whatever the
// track's length -- an hour at the floor is a few megabytes.
const (
	minRate = 600
	maxRate = 8000
	// perBar is how many samples each bar of the waveform is reduced from.
	perBar = 220
)

// rate returns the sample rate to decode at for a track of the given
// duration, so that reducing to bars has roughly [perBar] samples each.
func rate(dur time.Duration, bars int) int {
	if dur <= 0 {
		return maxRate
	}
	return min(max(int(float64(bars*perBar)/dur.Seconds()), minRate), maxRate)
}

// peaks decodes the audio with ffmpeg and reduces it to one normalized peak
// per bar. Returns nil when ffmpeg is unavailable or cannot decode the file --
// the card is still worth drawing without a waveform.
func peaks(ctx context.Context, pathName string, dur time.Duration, bars int) []float64 {
	if !ffmpeg.Available() {
		return nil
	}
	hz := rate(dur, bars)
	ivctx.Logf(ctx, "waveform: %d bars at %d Hz", bars, hz)
	buf, err := ffmpeg.Run(
		ctx,
		"-hide_banner",
		"-loglevel", "error",
		"-i", pathName,
		"-vn",
		"-ac", "1",
		"-ar", strconv.Itoa(hz),
		"-f", "s16le",
		"-",
	)
	if err != nil || len(buf) < 2 {
		return nil
	}
	return reduce(buf, bars)
}

// reduce turns signed 16-bit little endian mono pcm into one peak per bar,
// normalized so the loudest bar is 1.
func reduce(buf []byte, bars int) []float64 {
	n := len(buf) / 2
	if n < bars {
		return nil
	}
	out, maxPeak := make([]float64, bars), 0.0
	for i := range bars {
		lo, hi := i*n/bars, (i+1)*n/bars
		var peak float64
		for j := lo; j < hi; j++ {
			v := math.Abs(float64(int16(binary.LittleEndian.Uint16(buf[j*2:]))))
			peak = max(peak, v)
		}
		out[i], maxPeak = peak, max(maxPeak, peak)
	}
	if maxPeak == 0 {
		return out
	}
	for i := range out {
		// a mild curve, so quiet passages stay visible next to loud ones
		out[i] = math.Pow(out[i]/maxPeak, 0.7)
	}
	return out
}
