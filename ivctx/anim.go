package ivctx

import (
	"context"
	"time"
)

// Animated playback.
//
// A frame is drawn by clearing the terminal and encoding an image into it,
// which is work enough that how long it takes is not something the animation
// can be timed against: it varies with the terminal, the graphics protocol,
// and the size of the image. So playback runs against the wall clock instead
// -- each frame has a moment it belongs at, and a frame whose moment has gone
// by while the one before it was still being drawn is dropped rather than
// played late.
const (
	// AnimateFPS is the frame rate playback aims at.
	AnimateFPS = 30
	// AnimateDelay is how long one frame occupies at [AnimateFPS].
	AnimateDelay = time.Second / AnimateFPS
	// AnimateMax is how long a single pass of an animation plays for. An
	// animation shorter than this plays at its own length; a longer one is
	// cut off here, and a video is sampled across the whole of its length to
	// fit in it rather than being truncated to its first minute.
	AnimateMax = time.Minute
	// AnimateFrames is the most frames a single pass plays.
	AnimateFrames = int(AnimateMax / AnimateDelay)
)

// Animate reports whether the pipeline is decoding for animated playback,
// which is what a decoder for an animated format checks to know whether it is
// being asked for the animation or for a still of it.
func Animate(ctx context.Context) bool {
	return Get(ctx).Animate
}
