package vips

import (
	"strings"
	"testing"

	"github.com/cshum/vipsgen/v2/vips"
)

func TestLevelString(t *testing.T) {
	for _, level := range []vips.LogLevel{
		vips.LogLevelError,
		vips.LogLevelCritical,
		vips.LogLevelWarning,
		vips.LogLevelMessage,
		vips.LogLevelInfo,
		vips.LogLevelDebug,
	} {
		if got := levelString(level); len(got) != 3 {
			t.Errorf("levelString(%v) = %q, want a 3 character label", level, got)
		}
	}
	if got := levelString(vips.LogLevel(9999)); !strings.HasPrefix(got, "(") {
		t.Errorf("expected an unknown level to be formatted numerically, got %q", got)
	}
}
