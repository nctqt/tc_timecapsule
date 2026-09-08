package transcript

import (
	"testing"
)

func TestCleanVTT(t *testing.T) {
	rawVTT := `WEBVTT
Kind: captions
Language: en

1
00:00:00.500 --> 00:00:02.000
<c>Hello</c> and welcome back.

2
00:00:02.000 --> 00:00:04.000
Hello and welcome back.
To this true crime timeline breakdown.

3
00:00:04.000 --> 00:00:06.500
To this true crime timeline breakdown.
Today we cover the August 2026 updates.
`

	expected := "Hello and welcome back. To this true crime timeline breakdown. Today we cover the August 2026 updates."
	result := CleanVTT(rawVTT)

	if result != expected {
		t.Errorf("CleanVTT mismatch.\nGot:  %q\nWant: %q", result, expected)
	}
}
