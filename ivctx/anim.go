package ivctx

import (
	"context"
	"time"
)

// AnimateMax is how long a single pass of an animation plays for. An animation
// shorter than this plays at its own length, and a longer one is cut off here.
// A video plays for its own length, and is not cut off (D43).
const AnimateMax = time.Minute

// Animated playback runs against the wall clock.
//
// Drawing a frame means clearing the terminal and encoding an image into it,
// which is work enough that how long it takes is not something the animation
// can be timed against: it varies with the terminal, the graphics protocol,
// and the size of the image. So each frame has a moment it belongs at, and a
// frame whose moment has gone by while the one before it was still being drawn
// is dropped rather than played late.
//
// What an animation is played at is its own: a gif has a delay recorded with
// each frame, and a lottie and a video a frame rate. A video plays at its
// frame rate up to 30 frames a second (D43).

// AnimateDelay returns how long one frame of an animation is displayed for:
// the delay the animation gives it, or the one --fps forces when it was
// passed.
//
// A forced rate is the same for every frame, which for a video is the only
// kind there is, and for a gif or a lottie replaces timing the animation
// recorded for itself.
func AnimateDelay(ctx context.Context, delay time.Duration) time.Duration {
	if fps := Get(ctx).AnimateFPS; fps != 0 {
		return time.Second / time.Duration(fps)
	}
	return delay
}

// Animate reports whether the pipeline is decoding for animated playback,
// which is what a decoder for an animated format checks to know whether it is
// being asked for the animation or for a still of it.
func Animate(ctx context.Context) bool {
	return Get(ctx).Animate
}
