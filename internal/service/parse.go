package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/yourname/tabikake/internal/claude"
	"github.com/yourname/tabikake/internal/model"
)

// ParseService handles receipt image parsing via Claude Vision.
type ParseService struct {
	claude *claude.Client
}

// NewParseService creates a new ParseService.
func NewParseService(claudeClient *claude.Client) *ParseService {
	return &ParseService{claude: claudeClient}
}

// ParseReceiptFile reads a multipart file, base64-encodes it,
// and sends it to Claude Vision for structured extraction.
func (s *ParseService) ParseReceiptFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*model.ParseReceiptResult, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	mediaType := detectMediaType(data, header.Filename)
	imageBase64 := base64.StdEncoding.EncodeToString(data)

	result, err := s.claude.ParseReceipt(ctx, imageBase64, mediaType)
	if err != nil {
		return nil, fmt.Errorf("claude parse: %w", err)
	}

	return result, nil
}

// ParseReceiptBase64 accepts a raw base64 string with optional data URI prefix.
func (s *ParseService) ParseReceiptBase64(ctx context.Context, dataURI string) (*model.ParseReceiptResult, error) {
	imageBase64, mediaType, err := stripDataURI(dataURI)
	if err != nil {
		return nil, err
	}

	result, err := s.claude.ParseReceipt(ctx, imageBase64, mediaType)
	if err != nil {
		return nil, fmt.Errorf("claude parse: %w", err)
	}

	return result, nil
}

// detectMediaType sniffs the media type from file bytes, falling back to extension.
func detectMediaType(data []byte, filename string) string {
	detected := http.DetectContentType(data)
	if detected != "application/octet-stream" {
		return detected
	}
	// fallback by extension
	lower := filename
	if len(lower) > 4 {
		ext := lower[len(lower)-4:]
		switch ext {
		case ".jpg", "jpeg":
			return "image/jpeg"
		case ".png":
			return "image/png"
		case "webp":
			return "image/webp"
		}
	}
	return "image/jpeg"
}

// stripDataURI parses "data:image/jpeg;base64,<data>" or returns input as-is with jpeg assumed.
func stripDataURI(input string) (imageBase64, mediaType string, err error) {
	if len(input) > 5 && input[:5] == "data:" {
		// data:<mediaType>;base64,<data>
		rest := input[5:]
		semiIdx := -1
		for i, c := range rest {
			if c == ';' {
				semiIdx = i
				break
			}
		}
		if semiIdx < 0 {
			return "", "", fmt.Errorf("invalid data URI format")
		}
		mediaType = rest[:semiIdx]
		rest = rest[semiIdx+1:]
		if len(rest) < 7 || rest[:7] != "base64," {
			return "", "", fmt.Errorf("data URI must be base64 encoded")
		}
		imageBase64 = rest[7:]
		return imageBase64, mediaType, nil
	}
	return input, "image/jpeg", nil
}
