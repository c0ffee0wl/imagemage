package cmd

import (
	"encoding/base64"
	"fmt"
	"imagemage/pkg/filehandler"
	"imagemage/pkg/gemini"
	"imagemage/pkg/metadata"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// Gemini 3 Pro Image accepts more, but 4 keeps inline payloads well under the
// ~20MB threshold where Google recommends the Files API instead.
const maxReferenceImages = 4

const refPayloadWarnBytes = 20 * 1024 * 1024

const refStyleHint = "Use the attached reference image(s) as the authoritative visual style guide: match their color palette, iconography, line weight, layout density, and overall aesthetic. Do not copy specific content — only style.\n\n"

var (
	generateCount       int
	generateOutput      string
	generateStyle       string
	generatePreview     bool
	generateAspectRatio string
	generateResolution  string
	generateFrugal      bool
	generateSlide       bool
	generateConfig      string
	generateForce       bool
	generateStorePrompt bool
	generateRefs        []string
)

var generateCmd = &cobra.Command{
	Use:   "generate [prompt]",
	Short: "Generate images from text descriptions",
	Long: `Generate one or more images from a text prompt using Google's Gemini image models.

By default, uses Gemini 3 Pro Image (gemini-3-pro-image-preview) for high-quality 4K generation.
Use --frugal flag to switch to Nano Banana 2 (gemini-3.1-flash-image-preview) for Pro quality at Flash speed.

Examples:
  imagemage generate "watercolor painting of a fox in snowy forest"
  imagemage generate "mountain landscape" --count=3 --output=./images
  imagemage generate "cyberpunk city" --style="neon, futuristic"
  imagemage generate "wide cinematic shot" --aspect-ratio="21:9"
  imagemage generate "phone wallpaper" --aspect-ratio="9:16"
  imagemage generate "concept art" --frugal
  imagemage generate "process flow diagram" --slide --ref ./refs/house-style.png`,
	Args: cobra.MinimumNArgs(1),
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().IntVarP(&generateCount, "count", "c", 1, "Number of images to generate")
	generateCmd.Flags().StringVarP(&generateOutput, "output", "o", ".", "Output directory for generated images")
	generateCmd.Flags().StringVarP(&generateStyle, "style", "s", "", "Additional style guidance (e.g., 'watercolor', 'pixel-art')")
	generateCmd.Flags().BoolVarP(&generatePreview, "preview", "p", false, "Show preview information")
	generateCmd.Flags().StringVarP(&generateAspectRatio, "aspect-ratio", "a", "", "Aspect ratio (1:1, 16:9, 9:16, 4:3, 3:4, 3:2, 2:3, 21:9, 5:4, 4:5)")
	generateCmd.Flags().StringVarP(&generateResolution, "resolution", "r", "", "Image resolution (512px, 1K, 2K, 4K). Defaults to 4K")
	generateCmd.Flags().BoolVarP(&generateFrugal, "frugal", "f", false, "Use Nano Banana 2 (faster, cheaper, still supports 4K)")
	generateCmd.Flags().BoolVar(&generateSlide, "slide", false, "Optimize for presentation slides (4K, 16:9, with theme from config)")
	generateCmd.Flags().StringVar(&generateConfig, "config", "", "Path to config file (JSON) with style, colorScheme, additionalContext")
	generateCmd.Flags().BoolVar(&generateForce, "force", false, "Overwrite existing files without confirmation")
	generateCmd.Flags().BoolVar(&generateStorePrompt, "store-prompt", false, "Store prompt in PNG metadata for reproducibility")
	generateCmd.Flags().StringSliceVar(&generateRefs, "ref", nil, "Reference image(s) for style grounding (repeatable, or comma-separated). PNG/JPG/WebP.")
}

func refMimeForPath(path string) (string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png", nil
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	case ".webp":
		return "image/webp", nil
	default:
		return "", fmt.Errorf("unsupported reference image format %q (supported: .png, .jpg, .jpeg, .webp)", filepath.Ext(path))
	}
}

func loadReferenceImages(paths []string) ([]gemini.RefImage, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	if len(paths) > maxReferenceImages {
		return nil, fmt.Errorf("too many reference images: %d (max %d)", len(paths), maxReferenceImages)
	}
	refs := make([]gemini.RefImage, 0, len(paths))
	var total int
	for _, p := range paths {
		mime, err := refMimeForPath(p)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("reference image %q: %w", p, err)
		}
		refs = append(refs, gemini.RefImage{
			MimeType: mime,
			Base64:   base64.StdEncoding.EncodeToString(data),
		})
		total += len(data)
		fmt.Fprintf(os.Stderr, "[refs] attached %s (%s, %d bytes)\n", p, mime, len(data))
	}
	if total > refPayloadWarnBytes {
		fmt.Fprintf(os.Stderr, "[refs] warning: total reference payload is %d bytes (>%d); consider fewer/smaller images\n", total, refPayloadWarnBytes)
	}
	return refs, nil
}

func runGenerate(cmd *cobra.Command, args []string) error {
	prompt := args[0]

	// Load config if --slide or --config is specified
	var config *gemini.ImageGenConfig
	var err error
	if generateSlide || generateConfig != "" {
		config, err = gemini.FindConfig(generateConfig)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Apply --slide defaults
	if generateSlide {
		if generateAspectRatio == "" {
			generateAspectRatio = "16:9"
		}
		if generateResolution == "" {
			generateResolution = "4K"
		}
	}

	// Override with config defaults if not specified via flags
	if config != nil {
		if generateAspectRatio == "" && config.GetAspectRatio() != "" {
			generateAspectRatio = config.GetAspectRatio()
		}
		if generateResolution == "" && config.GetResolution() != "" {
			generateResolution = config.GetResolution()
		}
	}

	// Validate aspect ratio if provided
	if generateAspectRatio != "" {
		if err := gemini.ValidateAspectRatio(generateAspectRatio); err != nil {
			return err
		}
	}

	// Build full prompt with style and config
	fullPrompt := prompt
	if generateStyle != "" {
		fullPrompt = fmt.Sprintf("%s, style: %s", prompt, generateStyle)
	}

	// Apply config theme (style, colors, context)
	if config != nil {
		fullPrompt = config.ApplyToPrompt(fullPrompt)
	}

	// CLI --ref appends to config-declared references, not replace.
	var refPaths []string
	if config != nil {
		refPaths = append(refPaths, config.GetReferences()...)
	}
	refPaths = append(refPaths, generateRefs...)

	refs, err := loadReferenceImages(refPaths)
	if err != nil {
		return err
	}
	if len(refs) > 0 {
		fullPrompt = refStyleHint + fullPrompt
	}

	// Create Gemini client (frugal or default)
	var client *gemini.Client
	if generateFrugal {
		client, err = gemini.NewFrugalClient()
		if err != nil {
			return fmt.Errorf("failed to create Gemini client: %w", err)
		}
	} else {
		client, err = gemini.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create Gemini client: %w", err)
		}
	}

	// Display generation info
	fmt.Printf("Generating %d image(s) for: %s\n", generateCount, prompt)
	if config != nil {
		fmt.Printf("Config: Loaded (theme applied to prompt)\n")
	}
	if generateStyle != "" {
		fmt.Printf("Style: %s\n", generateStyle)
	}
	if generateAspectRatio != "" {
		fmt.Printf("Aspect Ratio: %s\n", generateAspectRatio)
	}
	// Display resolution info
	resolution := generateResolution
	if resolution == "" {
		resolution = "4K"
	}
	fmt.Printf("Resolution: %s\n", resolution)
	if generateFrugal {
		fmt.Printf("Model: %s (Nano Banana 2)\n", gemini.ModelNameFrugal)
	} else {
		fmt.Printf("Model: %s\n", gemini.ModelName)
	}
	fmt.Println()

	successCount := 0
	for i := 1; i <= generateCount; i++ {
		if generateCount > 1 {
			fmt.Printf("[%d/%d] Generating image...\n", i, generateCount)
		} else {
			fmt.Println("Generating image...")
		}

		result, err := client.GenerateContentWithRefs(fullPrompt, refs, generateResolution, generateAspectRatio)
		if err != nil {
			fmt.Printf("Error generating image %d: %v\n", i, err)
			continue
		}

		// Generate filename (prefer AI-suggested name)
		var filename string
		if generateCount > 1 {
			filename = filehandler.GenerateFilename(prompt, result.SuggestedName, "", i)
		} else {
			filename = filehandler.GenerateFilename(prompt, result.SuggestedName, "", 0)
		}

		// Create output path
		outputPath := filepath.Join(generateOutput, filename)
		outputPath = filehandler.EnsureUniqueFilename(outputPath)

		// Save image
		if err := filehandler.SaveImage(result.ImageData, outputPath); err != nil {
			fmt.Printf("Error saving image %d: %v\n", i, err)
			continue
		}

		// Store prompt in metadata if requested
		if generateStorePrompt {
			if err := metadata.AddPromptToPNG(outputPath, fullPrompt); err != nil {
				fmt.Printf("⚠️  Warning: failed to store prompt in metadata: %v\n", err)
				// Don't fail the whole operation just because metadata write failed
			}
		}

		fmt.Printf("✓ Saved to: %s\n", outputPath)
		if generateStorePrompt {
			fmt.Printf("  (prompt stored in metadata)\n")
		}
		successCount++
	}

	fmt.Printf("\nSuccessfully generated %d/%d images\n", successCount, generateCount)

	return nil
}
