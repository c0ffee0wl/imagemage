# AI-Generated Filenames Design

## Problem

Current image naming truncates prompts to 50 characters, producing:
- Long, unwieldy filenames
- Nearly identical names for similar prompts
- No semantic understanding of image content
- No differentiation between generated images

## Solution

Inject a filename request into the image generation prompt. Gemini returns both the image and a suggested name in the same API call - zero additional cost.

## Design

### Prompt Injection

Append to every prompt sent to Gemini:

```
[user's original prompt]

After generating the image, respond with a short (2-4 word) evocative filename for it. Just the words, no extension.
```

### Response Structure

New return type for generation functions:

```go
type GenerateResult struct {
    ImageData     string  // base64 encoded image
    SuggestedName string  // AI-suggested filename (may be empty)
}
```

### Filename Validation

**Valid name criteria:**
- 1-50 characters after cleaning
- Contains at least one letter

**Cleaning:**
- Lowercase
- Replace spaces with underscores
- Strip special characters (keep a-z, 0-9, underscore, hyphen)
- Trim leading/trailing underscores

**Fallback chain:**
1. Use AI-suggested name if valid
2. Otherwise, fall back to truncated/cleaned prompt (current behavior)

### Updated Function Signature

```go
func GenerateFilename(prompt, suggestedName, prefix string, count int) string
```

## Files to Modify

| File | Changes |
|------|---------|
| `pkg/gemini/client.go` | Add `GenerateResult` struct, update `extractImageData` → `extractResult`, append filename instruction to prompts |
| `pkg/filehandler/filehandler.go` | Update `GenerateFilename` to accept `suggestedName`, add `validateSuggestedName` helper |
| `cmd/generate.go` | Handle `GenerateResult`, pass `SuggestedName` to filename generation |
| `cmd/icon.go` | Handle `GenerateResult`, pass `SuggestedName` to filename generation |
| `cmd/pattern.go` | Handle `GenerateResult`, pass `SuggestedName` to filename generation |
| `cmd/story.go` | Handle `GenerateResult`, pass `SuggestedName` to filename generation |
| `cmd/diagram.go` | Handle `GenerateResult`, pass `SuggestedName` to filename generation |
| `cmd/edit.go` | Handle `GenerateResult` |
| `cmd/restore.go` | Handle `GenerateResult` |

## Non-Goals

- No new dependencies
- No new files beyond this design doc
- No changes to CLI interface or flags
