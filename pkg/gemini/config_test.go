package gemini

import "testing"

func TestApplyToPrompt(t *testing.T) {
	tests := []struct {
		name   string
		cfg    *ImageGenConfig
		prompt string
		want   string
	}{
		{
			name:   "nil config passthrough",
			cfg:    nil,
			prompt: "A cat.",
			want:   "A cat.",
		},
		{
			name:   "prompt only no defaults",
			cfg:    &ImageGenConfig{},
			prompt: "A cat.",
			want:   "A cat.",
		},
		{
			name: "prompt plus style",
			cfg: &ImageGenConfig{Defaults: ImageGenDefaults{
				Style: "flat vector illustration",
			}},
			prompt: "A cat.",
			want:   "A cat. Rendered in flat vector illustration.",
		},
		{
			name: "prompt style and colors",
			cfg: &ImageGenConfig{Defaults: ImageGenDefaults{
				Style:       "flat vector illustration",
				ColorScheme: "muted pastels",
			}},
			prompt: "A cat.",
			want:   "A cat. Rendered in flat vector illustration. Color palette: muted pastels.",
		},
		{
			name: "all four fields",
			cfg: &ImageGenConfig{Defaults: ImageGenDefaults{
				Style:             "flat vector illustration",
				ColorScheme:       "muted pastels",
				AdditionalContext: "Aim for a friendly, approachable tone.",
			}},
			prompt: "A cat.",
			want:   "A cat. Rendered in flat vector illustration. Color palette: muted pastels. Aim for a friendly, approachable tone.",
		},
		{
			name: "prompt without trailing punctuation",
			cfg: &ImageGenConfig{Defaults: ImageGenDefaults{
				Style: "watercolor",
			}},
			prompt: "A cat",
			want:   "A cat. Rendered in watercolor.",
		},
		{
			name: "context already terminated no double period",
			cfg: &ImageGenConfig{Defaults: ImageGenDefaults{
				AdditionalContext: "Clean white background.",
			}},
			prompt: "A cat.",
			want:   "A cat. Clean white background.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.ApplyToPrompt(tt.prompt)
			if got != tt.want {
				t.Errorf("ApplyToPrompt()\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestEnsureTerminated(t *testing.T) {
	cases := map[string]string{
		"":              "",
		"hello":         "hello.",
		"hello.":        "hello.",
		"hello!":        "hello!",
		"hello?":        "hello?",
		"hello  ":       "hello.",
		"hello.\n":      "hello.",
	}
	for in, want := range cases {
		if got := ensureTerminated(in); got != want {
			t.Errorf("ensureTerminated(%q) = %q, want %q", in, got, want)
		}
	}
}
