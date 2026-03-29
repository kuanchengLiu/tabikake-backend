package notion

import (
	"context"
	"fmt"
	"time"

	"github.com/jomei/notionapi"

	"github.com/yourname/tabikake/internal/model"
)

// UserRecords groups a user with their records and total for the settlement page.
type UserRecords struct {
	User     model.User
	Records  []model.Record
	TotalJPY int64
}

// SettlementExportData holds everything needed to render the settlement Notion page.
type SettlementExportData struct {
	Settlements []model.SettlementItem
	ByUser      []UserRecords
	TotalJPY    int64
}

// CreateSettlementPage creates a rich settlement summary page under parentPageID.
// Returns the URL of the created page.
func (c *Client) CreateSettlementPage(ctx context.Context, parentPageID, tripName string, data SettlementExportData) (string, error) {
	title := fmt.Sprintf("%s 結算 - %s", tripName, time.Now().Format("2006/01/02"))

	blocks := buildSettlementBlocks(data)

	page, err := c.api.Page.Create(ctx, &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:   notionapi.ParentTypePageID,
			PageID: notionapi.PageID(parentPageID),
		},
		Properties: notionapi.Properties{
			"title": notionapi.TitleProperty{
				Title: []notionapi.RichText{{Text: &notionapi.Text{Content: title}}},
			},
		},
		Children: blocks,
	})
	if err != nil {
		return "", fmt.Errorf("create settlement page: %w", err)
	}
	return page.URL, nil
}

func buildSettlementBlocks(data SettlementExportData) []notionapi.Block {
	var blocks []notionapi.Block

	// 結算明細
	blocks = append(blocks, heading2("結算明細"))

	if len(data.Settlements) == 0 {
		blocks = append(blocks, paragraph("（無需轉帳）"))
	} else {
		for _, s := range data.Settlements {
			line := fmt.Sprintf("%s → %s　¥%s", s.From.Name, s.To.Name, formatJPY(float64(s.AmountJPY)))
			blocks = append(blocks, paragraph(line))
		}
	}

	blocks = append(blocks, &notionapi.DividerBlock{
		BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeDivider},
		Divider:    notionapi.Divider{},
	})

	// 各人花費
	blocks = append(blocks, heading2("各人花費"))

	for _, ur := range data.ByUser {
		blocks = append(blocks, heading3(fmt.Sprintf("%s　共 ¥%s", ur.User.Name, formatJPY(float64(ur.TotalJPY)))))
		for _, r := range ur.Records {
			line := fmt.Sprintf("%s %s　¥%s　%s", r.Date, r.StoreNameZH, formatJPY(r.AmountJPY), r.Category)
			blocks = append(blocks, bulletItem(line))
		}
	}

	return blocks
}

func heading2(s string) *notionapi.Heading2Block {
	return &notionapi.Heading2Block{
		BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeHeading2},
		Heading2:   notionapi.Heading{RichText: richText(s)},
	}
}

func heading3(s string) *notionapi.Heading3Block {
	return &notionapi.Heading3Block{
		BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeHeading3},
		Heading3:   notionapi.Heading{RichText: richText(s)},
	}
}

func bulletItem(s string) *notionapi.BulletedListItemBlock {
	return &notionapi.BulletedListItemBlock{
		BasicBlock:       notionapi.BasicBlock{Type: notionapi.BlockTypeBulletedListItem},
		BulletedListItem: notionapi.ListItem{RichText: richText(s)},
	}
}
