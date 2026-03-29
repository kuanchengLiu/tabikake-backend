package notion

import (
	"context"
	"fmt"

	"github.com/jomei/notionapi"
)

// CreateTripPage creates a new page under NOTION_ROOT_PAGE_ID and returns its page ID.
func (c *Client) CreateTripPage(ctx context.Context, tripName string) (string, error) {
	page, err := c.api.Page.Create(ctx, &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:   notionapi.ParentTypePageID,
			PageID: notionapi.PageID(c.rootPageID),
		},
		Properties: notionapi.Properties{
			"title": notionapi.TitleProperty{
				Title: []notionapi.RichText{{Text: &notionapi.Text{Content: tripName}}},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("create trip page: %w", err)
	}
	return string(page.ID), nil
}

// CreateRecordsDatabase creates the Records database as a child of parentPageID
// with the full schema and returns the database ID.
func (c *Client) CreateRecordsDatabase(ctx context.Context, parentPageID string) (string, error) {
	db, err := c.api.Database.Create(ctx, &notionapi.DatabaseCreateRequest{
		Parent: notionapi.Parent{
			Type:   notionapi.ParentTypePageID,
			PageID: notionapi.PageID(parentPageID),
		},
		Title: []notionapi.RichText{{Text: &notionapi.Text{Content: "Records"}}},
		Properties: notionapi.PropertyConfigs{
			"StoreNameZH": notionapi.TitlePropertyConfig{
				Type: "title", Title: struct{}{},
			},
			"StoreNameJP": notionapi.RichTextPropertyConfig{
				Type: "rich_text", RichText: struct{}{},
			},
			"Date": notionapi.DatePropertyConfig{
				Type: "date", Date: struct{}{},
			},
			"Amount_JPY": notionapi.NumberPropertyConfig{
				Type:   "number",
				Number: notionapi.NumberFormat{Format: "number"},
			},
			"Tax_JPY": notionapi.NumberPropertyConfig{
				Type:   "number",
				Number: notionapi.NumberFormat{Format: "number"},
			},
			"Category": notionapi.SelectPropertyConfig{
				Type: "select",
				Select: notionapi.Select{Options: []notionapi.Option{
					{Name: "餐飲"}, {Name: "交通"}, {Name: "購物"}, {Name: "住宿"}, {Name: "其他"},
				}},
			},
			"Payment": notionapi.SelectPropertyConfig{
				Type: "select",
				Select: notionapi.Select{Options: []notionapi.Option{
					{Name: "現金"}, {Name: "Suica"}, {Name: "PayPay"}, {Name: "信用卡"},
				}},
			},
			"PaidByUserID": notionapi.RichTextPropertyConfig{
				Type: "rich_text", RichText: struct{}{},
			},
			"SplitWith": notionapi.RichTextPropertyConfig{
				Type: "rich_text", RichText: struct{}{},
			},
			"Items": notionapi.RichTextPropertyConfig{
				Type: "rich_text", RichText: struct{}{},
			},
		},
		IsInline: false,
	})
	if err != nil {
		return "", fmt.Errorf("create records database: %w", err)
	}
	return string(db.ID), nil
}
