package filehandler

import (
	"testing"
)

func TestValidateSuggestedName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid short name",
			input:    "ember wyrm",
			expected: "ember_wyrm",
		},
		{
			name:     "valid name with special chars",
			input:    "Twilight Sentinel!",
			expected: "twilight_sentinel",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "only numbers",
			input:    "12345",
			expected: "",
		},
		{
			name:     "only special chars",
			input:    "!@#$%",
			expected: "",
		},
		{
			name:     "exceeds 50 chars",
			input:    "this is a very long suggested name that definitely exceeds the fifty character limit",
			expected: "",
		},
		{
			name:     "valid at 50 chars",
			input:    "aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeee",
			expected: "aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeee",
		},
		{
			name:     "mixed letters and numbers",
			input:    "dragon42",
			expected: "dragon42",
		},
		{
			name:     "single letter",
			input:    "x",
			expected: "x",
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: "",
		},
		{
			name:     "hyphens are preserved",
			input:    "hello-world",
			expected: "hello-world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateSuggestedName(tt.input)
			if result != tt.expected {
				t.Errorf("validateSuggestedName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateFilename(t *testing.T) {
	tests := []struct {
		name          string
		prompt        string
		suggestedName string
		prefix        string
		count         int
		expected      string
	}{
		{
			name:          "prefers valid suggested name",
			prompt:        "a majestic dragon breathing fire over a medieval castle",
			suggestedName: "ember wyrm",
			prefix:        "",
			count:         0,
			expected:      "ember_wyrm.png",
		},
		{
			name:          "falls back to prompt when suggested empty",
			prompt:        "cute fox",
			suggestedName: "",
			prefix:        "",
			count:         0,
			expected:      "cute_fox.png",
		},
		{
			name:          "falls back to prompt when suggested invalid",
			prompt:        "sunset beach",
			suggestedName: "12345",
			prefix:        "",
			count:         0,
			expected:      "sunset_beach.png",
		},
		{
			name:          "applies prefix with suggested name",
			prompt:        "geometric pattern",
			suggestedName: "hexagon dance",
			prefix:        "pattern",
			count:         0,
			expected:      "pattern_hexagon_dance.png",
		},
		{
			name:          "applies prefix with fallback",
			prompt:        "geometric pattern",
			suggestedName: "",
			prefix:        "pattern",
			count:         0,
			expected:      "pattern_geometric_pattern.png",
		},
		{
			name:          "adds count when specified",
			prompt:        "mountain landscape",
			suggestedName: "alpine vista",
			prefix:        "",
			count:         3,
			expected:      "alpine_vista_3.png",
		},
		{
			name:          "prefix and count together",
			prompt:        "story scene",
			suggestedName: "dawn awakening",
			prefix:        "story_frame_01",
			count:         0,
			expected:      "story_frame_01_dawn_awakening.png",
		},
		{
			name:          "truncates long prompt to 50 chars",
			prompt:        "a very long prompt that describes a beautiful scenic mountain landscape with snow",
			suggestedName: "",
			prefix:        "",
			count:         0,
			expected:      "a_very_long_prompt_that_describes_a_beautiful_scen.png",
		},
		{
			name:          "count zero not added",
			prompt:        "test",
			suggestedName: "simple",
			prefix:        "",
			count:         0,
			expected:      "simple.png",
		},
		{
			name:          "count one is added",
			prompt:        "test",
			suggestedName: "simple",
			prefix:        "",
			count:         1,
			expected:      "simple_1.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateFilename(tt.prompt, tt.suggestedName, tt.prefix, tt.count)
			if result != tt.expected {
				t.Errorf("GenerateFilename(%q, %q, %q, %d) = %q, want %q",
					tt.prompt, tt.suggestedName, tt.prefix, tt.count, result, tt.expected)
			}
		})
	}
}
