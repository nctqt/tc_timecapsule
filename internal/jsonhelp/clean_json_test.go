package jsonhelp

import "testing"

func TestCleanJSONOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Clean JSON Object",
			input:    `{"summary": "test", "category": "crime"}`,
			expected: `{"summary": "test", "category": "crime"}`,
		},
		{
			name:     "Markdown json Code Block",
			input:    "```json\n{\n  \"summary\": \"test\"\n}\n```",
			expected: "{\n  \"summary\": \"test\"\n}",
		},
		{
			name:     "Preamble text before JSON",
			input:    `Here is the result: {"summary": "test"} Thank you!`,
			expected: `{"summary": "test"}`,
		},
		{
			name:     "Raw refusal text (no JSON braces)",
			input:    "Unable to process request",
			expected: "Unable to process request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanJSONOutput(tt.input)
			if got != tt.expected {
				t.Errorf("CleanJSONOutput() = %q, want %q", got, tt.expected)
			}
		})
	}
}
