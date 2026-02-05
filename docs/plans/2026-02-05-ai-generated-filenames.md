# AI-Generated Filenames Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make Gemini suggest short, evocative filenames for generated images by injecting a naming request into prompts.

**Architecture:** Append filename request to prompts, return both image and suggested name from API, validate/fallback in filehandler.

**Tech Stack:** Go, Gemini API

---

### Task 1: Add GenerateResult struct and update extractImageData

**Files:**
- Modify: `pkg/gemini/client.go:78-93` (add struct after existing types)
- Modify: `pkg/gemini/client.go:339-360` (rename and update extraction function)

**Step 1: Add GenerateResult struct after ErrorInfo**

In `pkg/gemini/client.go`, add after line 93:

```go
// GenerateResult contains both image data and AI-suggested filename
type GenerateResult struct {
	ImageData     string // base64 encoded image
	SuggestedName string // AI-suggested filename (may be empty)
}
```

**Step 2: Rename extractImageData to extractResult and return GenerateResult**

Replace the `extractImageData` function (lines 339-360) with:

```go
// extractResult extracts base64 image data and suggested filename from the response
func (c *Client) extractResult(result *GenerateResponse) GenerateResult {
	var res GenerateResult
	if len(result.Candidates) == 0 {
		return res
	}

	for _, part := range result.Candidates[0].Content.Parts {
		// Check for inline data (image)
		if part.InlineData != nil && part.InlineData.Data != "" {
			res.ImageData = part.InlineData.Data
		}

		// Check for text (suggested filename)
		if part.Text != "" && len(part.Text) < 100 {
			// Only use text if it looks like a filename suggestion (short)
			res.SuggestedName = part.Text
		}
	}

	return res
}
```

**Step 3: Run tests to verify no breakage yet**

Run: `cd /home/cquinn/src/imagemage && go build ./...`
Expected: Build succeeds (tests will fail until we update callers)

**Step 4: Commit**

```bash
git add pkg/gemini/client.go
git commit -m "feat: Add GenerateResult struct for image+filename extraction"
```

---

### Task 2: Add filename suffix constant and update GenerateContentWithFullOptions

**Files:**
- Modify: `pkg/gemini/client.go:14-18` (add constant)
- Modify: `pkg/gemini/client.go:215-336` (update main generation function)

**Step 1: Add filename suffix constant**

In `pkg/gemini/client.go`, add after line 18 (after BaseURL):

```go
	FilenameSuffix = "\n\nAfter generating the image, respond with a short (2-4 word) evocative filename for it. Just the words, no extension."
```

**Step 2: Update GenerateContentWithFullOptions to append suffix and return GenerateResult**

Change the function signature on line 215 from:
```go
func (c *Client) GenerateContentWithFullOptions(prompt string, imagesBase64 []string, resolution string, aspectRatio string) (string, error) {
```
to:
```go
func (c *Client) GenerateContentWithFullOptions(prompt string, imagesBase64 []string, resolution string, aspectRatio string) (GenerateResult, error) {
```

On line 220-222, change:
```go
	parts := []Part{
		{Text: prompt},
	}
```
to:
```go
	fullPrompt := prompt + FilenameSuffix
	parts := []Part{
		{Text: fullPrompt},
	}
```

On lines 329-335, change:
```go
	// Extract image data from response
	imageData := c.extractImageData(&result)
	if imageData == "" {
		return "", fmt.Errorf("no image data found in response")
	}

	return imageData, nil
```
to:
```go
	// Extract image data and suggested filename from response
	res := c.extractResult(&result)
	if res.ImageData == "" {
		return GenerateResult{}, fmt.Errorf("no image data found in response")
	}

	return res, nil
```

Also update the error returns on lines 217-218, 267-268, 294-295, 316-317, 325-327 to return `GenerateResult{}` instead of `""`.

**Step 3: Run build to check syntax**

Run: `cd /home/cquinn/src/imagemage && go build ./...`
Expected: Fails (other functions still return string)

**Step 4: Commit**

```bash
git add pkg/gemini/client.go
git commit -m "feat: Update GenerateContentWithFullOptions to inject filename request"
```

---

### Task 3: Update all other GenerateContent* wrapper functions

**Files:**
- Modify: `pkg/gemini/client.go:186-212` (update wrapper functions)

**Step 1: Update all wrapper functions to return GenerateResult**

Replace lines 186-212 with:

```go
// GenerateContent sends a request to generate content
func (c *Client) GenerateContent(prompt string) (GenerateResult, error) {
	return c.GenerateContentWithOptions(prompt, "", "")
}

// GenerateContentWithImage sends a request to generate or edit content with an optional image
func (c *Client) GenerateContentWithImage(prompt string, imageBase64 string) (GenerateResult, error) {
	return c.GenerateContentWithImages(prompt, []string{imageBase64}, "")
}

// GenerateContentWithImages sends a request with multiple input images
func (c *Client) GenerateContentWithImages(prompt string, imagesBase64 []string, aspectRatio string) (GenerateResult, error) {
	return c.GenerateContentWithFullOptions(prompt, imagesBase64, "", aspectRatio)
}

// GenerateContentWithResolution sends a request with resolution and aspect ratio
func (c *Client) GenerateContentWithResolution(prompt string, resolution string, aspectRatio string) (GenerateResult, error) {
	return c.GenerateContentWithFullOptions(prompt, nil, resolution, aspectRatio)
}

// GenerateContentWithOptions sends a request to generate or edit content with full options
func (c *Client) GenerateContentWithOptions(prompt string, imageBase64 string, aspectRatio string) (GenerateResult, error) {
	var images []string
	if imageBase64 != "" {
		images = []string{imageBase64}
	}
	return c.GenerateContentWithFullOptions(prompt, images, "", aspectRatio)
}
```

**Step 2: Run build**

Run: `cd /home/cquinn/src/imagemage && go build ./...`
Expected: Fails (cmd files still expect string)

**Step 3: Commit**

```bash
git add pkg/gemini/client.go
git commit -m "feat: Update all GenerateContent wrappers to return GenerateResult"
```

---

### Task 4: Update filehandler.GenerateFilename to accept suggestedName

**Files:**
- Modify: `pkg/filehandler/filehandler.go:44-88` (update GenerateFilename and add validation)

**Step 1: Add validateSuggestedName helper function**

Add after `cleanPrompt` function (after line 88):

```go
// validateSuggestedName checks if a suggested name is usable
func validateSuggestedName(name string) string {
	// Clean the name using same rules as prompts
	cleaned := cleanPrompt(name)

	// Must have at least one letter
	hasLetter := false
	for _, c := range cleaned {
		if c >= 'a' && c <= 'z' {
			hasLetter = true
			break
		}
	}

	// Valid if 1-50 chars and has a letter
	if len(cleaned) >= 1 && len(cleaned) <= 50 && hasLetter {
		return cleaned
	}

	return ""
}
```

**Step 2: Update GenerateFilename to accept suggestedName parameter**

Replace lines 44-69 with:

```go
// GenerateFilename creates a descriptive filename from a prompt or suggested name
func GenerateFilename(prompt, suggestedName, prefix string, count int) string {
	// Try to use AI-suggested name first
	cleaned := validateSuggestedName(suggestedName)

	// Fall back to prompt-based name
	if cleaned == "" {
		cleaned = cleanPrompt(prompt)
		// Truncate if too long
		maxLen := 50
		if len(cleaned) > maxLen {
			cleaned = cleaned[:maxLen]
		}
	}

	// Build filename
	var filename string
	if prefix != "" {
		filename = fmt.Sprintf("%s_%s", prefix, cleaned)
	} else {
		filename = cleaned
	}

	// Add counter if specified
	if count > 0 {
		filename = fmt.Sprintf("%s_%d", filename, count)
	}

	return filename + ".png"
}
```

**Step 3: Run build**

Run: `cd /home/cquinn/src/imagemage && go build ./...`
Expected: Fails (cmd files call with wrong number of args)

**Step 4: Commit**

```bash
git add pkg/filehandler/filehandler.go
git commit -m "feat: Update GenerateFilename to prefer AI-suggested names"
```

---

### Task 5: Update cmd/generate.go

**Files:**
- Modify: `cmd/generate.go:174-196`

**Step 1: Update generate command to use GenerateResult**

Replace lines 174-196 with:

```go
		// Generate image with resolution support
		result, err := client.GenerateContentWithResolution(fullPrompt, generateResolution, generateAspectRatio)
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
```

**Step 2: Run build**

Run: `cd /home/cquinn/src/imagemage && go build ./...`
Expected: Fails (other cmd files still broken)

**Step 3: Commit**

```bash
git add cmd/generate.go
git commit -m "feat: Update generate command to use AI-suggested filenames"
```

---

### Task 6: Update cmd/pattern.go

**Files:**
- Modify: `cmd/pattern.go:60-74`

**Step 1: Update pattern command**

Replace lines 60-74 with:

```go
	// Generate pattern
	result, err := client.GenerateContent(prompt)
	if err != nil {
		return fmt.Errorf("failed to generate pattern: %w", err)
	}

	// Generate filename (prefer AI-suggested name)
	filename := filehandler.GenerateFilename(description, result.SuggestedName, "pattern", 0)
	outputPath := filepath.Join(patternOutput, filename)
	outputPath = filehandler.EnsureUniqueFilename(outputPath)

	// Save pattern
	if err := filehandler.SaveImage(result.ImageData, outputPath); err != nil {
		return fmt.Errorf("failed to save pattern: %w", err)
	}
```

**Step 2: Commit**

```bash
git add cmd/pattern.go
git commit -m "feat: Update pattern command to use AI-suggested filenames"
```

---

### Task 7: Update cmd/diagram.go

**Files:**
- Modify: `cmd/diagram.go:53-68`

**Step 1: Update diagram command**

Replace lines 53-68 with:

```go
	// Generate diagram
	result, err := client.GenerateContent(prompt)
	if err != nil {
		return fmt.Errorf("failed to generate diagram: %w", err)
	}

	// Generate filename (prefer AI-suggested name)
	filename := filehandler.GenerateFilename(description, result.SuggestedName, diagramType, 0)
	outputPath := filepath.Join(diagramOutput, filename)
	outputPath = filehandler.EnsureUniqueFilename(outputPath)

	// Save diagram
	if err := filehandler.SaveImage(result.ImageData, outputPath); err != nil {
		return fmt.Errorf("failed to save diagram: %w", err)
	}
```

**Step 2: Commit**

```bash
git add cmd/diagram.go
git commit -m "feat: Update diagram command to use AI-suggested filenames"
```

---

### Task 8: Update cmd/story.go

**Files:**
- Modify: `cmd/story.go:81-97`

**Step 1: Update story command**

Replace lines 81-97 with:

```go
		// Generate image
		result, err := client.GenerateContent(prompt)
		if err != nil {
			fmt.Printf("Error generating frame %d: %v\n", i, err)
			continue
		}

		// Generate filename (prefer AI-suggested name, with frame prefix)
		filename := filehandler.GenerateFilename(narrative, result.SuggestedName, fmt.Sprintf("story_frame_%02d", i), 0)
		outputPath := filepath.Join(storyOutput, filename)
		outputPath = filehandler.EnsureUniqueFilename(outputPath)

		// Save image
		if err := filehandler.SaveImage(result.ImageData, outputPath); err != nil {
			fmt.Printf("Error saving frame %d: %v\n", i, err)
			continue
		}
```

**Step 2: Commit**

```bash
git add cmd/story.go
git commit -m "feat: Update story command to use AI-suggested filenames"
```

---

### Task 9: Update cmd/icon.go

**Files:**
- Modify: `cmd/icon.go:92-116`

**Step 1: Update icon command**

Replace lines 92-116 with:

```go
	var result gemini.GenerateResult
	if inputImageBase64 != "" {
		result, err = client.GenerateContentWithImages(prompt, []string{inputImageBase64}, "1:1")
	} else {
		result, err = client.GenerateContentWithImages(prompt, nil, "1:1")
	}
	if err != nil {
		return fmt.Errorf("failed to generate icon: %w", err)
	}

	// Resize and save icons at each requested size
	successCount := 0
	for _, size := range sizes {
		filename := filehandler.GenerateFilename(description, result.SuggestedName, fmt.Sprintf("icon_%dx%d", size, size), 0)
		outputPath := filepath.Join(iconOutput, filename)
		outputPath = filehandler.EnsureUniqueFilename(outputPath)

		if err := filehandler.ResizeAndSaveImage(result.ImageData, size, outputPath); err != nil {
			fmt.Printf("Error saving %dx%d icon: %v\n", size, size, err)
			continue
		}

		fmt.Printf("✓ Saved %dx%d icon to: %s\n", size, size, outputPath)
		successCount++
	}
```

**Step 2: Add import for gemini package**

At line 6, add `"imagemage/pkg/gemini"` to imports.

**Step 3: Commit**

```bash
git add cmd/icon.go
git commit -m "feat: Update icon command to use AI-suggested filenames"
```

---

### Task 10: Update cmd/edit.go

**Files:**
- Modify: `cmd/edit.go:192-206`

**Step 1: Update edit command**

Replace lines 192-206 with:

```go
	// Generate with all images
	var result gemini.GenerateResult
	if editResolution != "" || editAspectRatio != "" {
		result, err = client.GenerateContentWithFullOptions(instruction, allImagesBase64, editResolution, editAspectRatio)
	} else {
		result, err = client.GenerateContentWithImages(instruction, allImagesBase64, "")
	}

	if err != nil {
		return fmt.Errorf("failed to edit image: %w", err)
	}

	// Save edited image
	if err := filehandler.SaveImage(result.ImageData, outputPath); err != nil {
		return fmt.Errorf("failed to save edited image: %w", err)
	}
```

**Step 2: Commit**

```bash
git add cmd/edit.go
git commit -m "feat: Update edit command to handle GenerateResult"
```

---

### Task 11: Update cmd/restore.go

**Files:**
- Modify: `cmd/restore.go:56-75`

**Step 1: Update restore command**

Replace lines 56-75 with:

```go
	// Generate restored image
	result, err := client.GenerateContentWithImage(prompt, imageBase64)
	if err != nil {
		return fmt.Errorf("failed to restore image: %w", err)
	}

	// Determine output path
	outputPath := restoreOutput
	if outputPath == "" {
		ext := filepath.Ext(imagePath)
		base := strings.TrimSuffix(imagePath, ext)
		outputPath = base + "_restored" + ext
	}

	outputPath = filehandler.EnsureUniqueFilename(outputPath)

	// Save restored image
	if err := filehandler.SaveImage(result.ImageData, outputPath); err != nil {
		return fmt.Errorf("failed to save restored image: %w", err)
	}
```

**Step 2: Commit**

```bash
git add cmd/restore.go
git commit -m "feat: Update restore command to handle GenerateResult"
```

---

### Task 12: Update tests and verify build

**Files:**
- Modify: `pkg/gemini/client_test.go`

**Step 1: Update tests to handle GenerateResult**

In `client_test.go`, update all occurrences where `GenerateContent*` functions are called. Change:
```go
_, err := client.GenerateContent("test prompt")
```
to:
```go
result, err := client.GenerateContent("test prompt")
_ = result // silence unused variable warning
```

And similar for `GenerateContentWithResolution` calls.

**Step 2: Run full test suite**

Run: `cd /home/cquinn/src/imagemage && go test ./...`
Expected: All tests pass

**Step 3: Run build**

Run: `cd /home/cquinn/src/imagemage && go build ./...`
Expected: Build succeeds

**Step 4: Run pre-commit hooks**

Run: `cd /home/cquinn/src/imagemage && pre-commit run --all-files`
Expected: All hooks pass

**Step 5: Commit**

```bash
git add pkg/gemini/client_test.go
git commit -m "test: Update tests for GenerateResult return type"
```

---

### Task 13: Final verification

**Step 1: Run full test suite again**

Run: `cd /home/cquinn/src/imagemage && go test ./... -v`
Expected: All tests pass

**Step 2: Build binary**

Run: `cd /home/cquinn/src/imagemage && go build -o imagemage .`
Expected: Binary builds successfully

**Step 3: Verify help still works**

Run: `cd /home/cquinn/src/imagemage && ./imagemage --help`
Expected: Help displays correctly
