package vips

import (
	"errors"
	"strings"
	"testing"

	"github.com/cshum/vipsgen/vips"
)

func TestIsEncrypted(t *testing.T) {
	for _, test := range []struct {
		err error
		exp bool
	}{
		{nil, false},
		{errors.New("pdfload: Document is encrypted"), true},
		{errors.New("vips load: document is encrypted"), true},
		{errors.New("pdfload: unable to load"), false},
	} {
		if got := IsEncrypted(test.err); got != test.exp {
			t.Errorf("IsEncrypted(%v) = %v, want %v", test.err, got, test.exp)
		}
	}
}

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
