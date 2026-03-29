package notion

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jomei/notionapi"
)

// Client wraps the Notion API using the server's integration token.
type Client struct {
	api        *notionapi.Client
	rootPageID string
}

// New creates a Notion client.
func New(integrationToken, rootPageID string) *Client {
	return &Client{
		api:        notionapi.NewClient(notionapi.Token(integrationToken)),
		rootPageID: rootPageID,
	}
}

// --- helpers shared across files ---

func richText(s string) []notionapi.RichText {
	return []notionapi.RichText{{Text: &notionapi.Text{Content: s}}}
}

func paragraph(s string) *notionapi.ParagraphBlock {
	return &notionapi.ParagraphBlock{
		BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeParagraph},
		Paragraph:  notionapi.Paragraph{RichText: richText(s)},
	}
}

func toNotionDate(s string) *notionapi.Date {
	t, _ := time.Parse("2006-01-02", s)
	d := notionapi.Date(t)
	return &d
}

func formatDate(d *notionapi.Date) string {
	return time.Time(*d).Format("2006-01-02")
}

func formatJPY(v float64) string {
	return fmt.Sprintf("%.0f", v)
}

func marshalJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
