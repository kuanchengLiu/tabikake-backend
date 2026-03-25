package claude

import (
	"context"
	"encoding/json"
	"fmt"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/yourname/tabikake/internal/model"
)

const (
	receiptParseModel = anthropic.ModelClaudeSonnet4_6
	maxTokens         = 1024
)

const receiptPrompt = `你是一個日本收據解析助手，請從圖片中擷取資訊並只回傳以下 JSON，不要其他文字：
{
  "store_name_jp": "店名（日文原文）",
  "store_name_zh": "店名（繁體中文翻譯）",
  "amount_jpy": 1234,
  "tax_jpy": 111,
  "payment_method": "現金|Suica|PayPay|信用卡",
  "category": "餐飲|交通|購物|住宿|其他",
  "items": [
    { "name_jp": "品項日文", "name_zh": "品項繁中", "price": 123 }
  ],
  "date": "2026-03-23"
}`

// Client wraps the Anthropic SDK client.
type Client struct {
	api anthropic.Client
}

// New creates a new Claude client with the given API key.
func New(apiKey string) *Client {
	api := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Client{api: api}
}

// ParseReceipt sends a base64-encoded image to Claude Vision and returns
// the structured receipt data. imageBase64 should be raw base64 (no data URI prefix).
// mediaType should be e.g. "image/jpeg", "image/png".
func (c *Client) ParseReceipt(ctx context.Context, imageBase64, mediaType string) (*model.ParseReceiptResult, error) {
	msg, err := c.api.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     receiptParseModel,
		MaxTokens: maxTokens,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewImageBlockBase64(mediaType, imageBase64),
				anthropic.NewTextBlock(receiptPrompt),
			),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude api error: %w", err)
	}

	if len(msg.Content) == 0 {
		return nil, fmt.Errorf("claude returned empty response")
	}

	textBlock := msg.Content[0].AsText()
	if textBlock.Text == "" {
		return nil, fmt.Errorf("unexpected claude response content type")
	}

	var result model.ParseReceiptResult
	if err := json.Unmarshal([]byte(textBlock.Text), &result); err != nil {
		return nil, fmt.Errorf("failed to parse claude response as JSON: %w (raw: %s)", err, textBlock.Text)
	}

	return &result, nil
}
