package youtube

import "testing"

func TestExtractVideoID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantID  string
		wantErr bool
	}{
		{"Raw 11-char ID", "dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"Standard URL", "https://www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"Standard URL with extras", "https://www.youtube.com/watch?v=dQw4w9WgXcQ&feature=shared", "dQw4w9WgXcQ", false},
		{"Shortened URL", "https://youtu.be/dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"Embed URL", "https://www.youtube.com/embed/dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"Shorts URL", "https://www.youtube.com/shorts/dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"Invalid URL", "https://example.com/video", "", true},
		{"Empty String", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractVideoID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ExtractVideoID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.wantID {
				t.Errorf("ExtractVideoID(%q) = %q, want %q", tt.input, got, tt.wantID)
			}
		})
	}
}
