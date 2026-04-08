package gemini

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ensureTerminated trims trailing whitespace from s and, if the result does
// not already end in sentence-terminating punctuation, appends a period.
// An empty string is returned unchanged.
func ensureTerminated(s string) string {
	s = strings.TrimRight(s, " \t\r\n")
	if s == "" {
		return s
	}
	switch s[len(s)-1] {
	case '.', '!', '?':
		return s
	}
	return s + "."
}

// ImageConfig represents configuration for image generation
type ImageGenConfig struct {
	Defaults ImageGenDefaults `json:"defaults"`

	// configDir is the directory containing the loaded config file, used to
	// resolve relative paths (e.g. references). Not serialized.
	configDir string
}

// ImageGenDefaults represents default settings for image generation
type ImageGenDefaults struct {
	AspectRatio       string   `json:"aspectRatio"`
	Resolution        string   `json:"resolution"`
	Style             string   `json:"style"`
	ColorScheme       string   `json:"colorScheme"`
	AdditionalContext string   `json:"additionalContext"`
	References        []string `json:"references"`
}

// LoadConfig loads configuration from a file
func LoadConfig(configPath string) (*ImageGenConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ImageGenConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Record the directory so callers can resolve relative paths (e.g.
	// references) against the config file's location rather than CWD.
	if absPath, err := filepath.Abs(configPath); err == nil {
		config.configDir = filepath.Dir(absPath)
	} else {
		config.configDir = filepath.Dir(configPath)
	}

	return &config, nil
}

// FindConfig searches for config files in order of precedence:
// 1. Specified config path (if provided)
// 2. ./image-gen.config.json (per-talk config)
// 3. ~/src/talks/image-gen.defaults.json (global defaults)
func FindConfig(specifiedPath string) (*ImageGenConfig, error) {
	// Try specified path first
	if specifiedPath != "" {
		return LoadConfig(specifiedPath)
	}

	// Try current directory for per-talk config
	localConfig := "image-gen.config.json"
	if _, err := os.Stat(localConfig); err == nil {
		return LoadConfig(localConfig)
	}

	// Try global defaults
	homeDir, err := os.UserHomeDir()
	if err == nil {
		globalConfig := filepath.Join(homeDir, "src", "talks", "image-gen.defaults.json")
		if _, err := os.Stat(globalConfig); err == nil {
			return LoadConfig(globalConfig)
		}
	}

	// No config found - return nil (not an error)
	return nil, nil
}

// ApplyConfigToPrompt applies configuration settings to a prompt
func (c *ImageGenConfig) ApplyToPrompt(prompt string) string {
	if c == nil {
		return prompt
	}

	fullPrompt := ensureTerminated(prompt)

	if c.Defaults.Style != "" {
		fullPrompt = fmt.Sprintf("%s Rendered in %s", fullPrompt, ensureTerminated(c.Defaults.Style))
	}

	if c.Defaults.ColorScheme != "" {
		fullPrompt = fmt.Sprintf("%s Color palette: %s", fullPrompt, ensureTerminated(c.Defaults.ColorScheme))
	}

	if c.Defaults.AdditionalContext != "" {
		ctx := strings.TrimRight(c.Defaults.AdditionalContext, " \t\r\n")
		if ctx != "" {
			fullPrompt = fmt.Sprintf("%s %s", fullPrompt, ctx)
		}
	}

	return fullPrompt
}

// GetAspectRatio returns the aspect ratio from config, or empty string if not set
func (c *ImageGenConfig) GetAspectRatio() string {
	if c == nil {
		return ""
	}
	return c.Defaults.AspectRatio
}

// GetResolution returns the resolution from config, or empty string if not set
func (c *ImageGenConfig) GetResolution() string {
	if c == nil {
		return ""
	}
	return c.Defaults.Resolution
}

// GetReferences returns reference image paths from config, resolved against
// the config file's directory when they are relative. Returns nil if the
// config is nil or has no references.
func (c *ImageGenConfig) GetReferences() []string {
	if c == nil || len(c.Defaults.References) == 0 {
		return nil
	}
	out := make([]string, 0, len(c.Defaults.References))
	for _, ref := range c.Defaults.References {
		if ref == "" {
			continue
		}
		if filepath.IsAbs(ref) || c.configDir == "" {
			out = append(out, ref)
		} else {
			out = append(out, filepath.Join(c.configDir, ref))
		}
	}
	return out
}
