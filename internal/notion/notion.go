package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jomei/notionapi"

	"github.com/yourname/tabikake/internal/model"
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

// --- Trip page + Records database setup ---

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
			"Store": notionapi.TitlePropertyConfig{
				Type: "title", Title: struct{}{},
			},
			"Date": notionapi.DatePropertyConfig{
				Type: "date", Date: struct{}{},
			},
			"Amount_JPY": notionapi.NumberPropertyConfig{
				Type:   "number",
				Number: notionapi.NumberFormat{Format: "number"},
			},
			"Amount_TWD": notionapi.NumberPropertyConfig{
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
			"PaidBy": notionapi.RichTextPropertyConfig{
				Type: "rich_text", RichText: struct{}{},
			},
			"PaidByName": notionapi.RichTextPropertyConfig{
				Type: "rich_text", RichText: struct{}{},
			},
			"PaidByMemberID": notionapi.RichTextPropertyConfig{
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

// --- Records CRUD ---

// ListRecords returns all expense records from the given database.
func (c *Client) ListRecords(ctx context.Context, dbID string) ([]model.Record, error) {
	resp, err := c.api.Database.Query(ctx, notionapi.DatabaseID(dbID), &notionapi.DatabaseQueryRequest{})
	if err != nil {
		return nil, fmt.Errorf("notion list records: %w", err)
	}

	records := make([]model.Record, 0, len(resp.Results))
	for _, page := range resp.Results {
		rec, err := pageToRecord(page)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}

// CreateRecord creates a new expense record page in the given database.
func (c *Client) CreateRecord(ctx context.Context, dbID string, req model.CreateRecordRequest) (*model.Record, error) {
	itemsJSON, _ := json.Marshal(req.Items)
	splitWithJSON, _ := json.Marshal(req.SplitWith)

	properties := notionapi.Properties{
		"Store": notionapi.TitleProperty{
			Title: []notionapi.RichText{{Text: &notionapi.Text{Content: req.Store}}},
		},
		"Date": notionapi.DateProperty{
			Date: &notionapi.DateObject{Start: toNotionDate(req.Date)},
		},
		"Amount_JPY": notionapi.NumberProperty{Number: req.AmountJPY},
		"Amount_TWD": notionapi.NumberProperty{Number: req.AmountTWD},
		"Tax_JPY":    notionapi.NumberProperty{Number: req.TaxJPY},
		"Category":   notionapi.SelectProperty{Select: notionapi.Option{Name: req.Category}},
		"Payment":    notionapi.SelectProperty{Select: notionapi.Option{Name: req.Payment}},
		"PaidBy":     notionapi.RichTextProperty{RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: req.PaidBy}}}},
		"PaidByName": notionapi.RichTextProperty{RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: req.PaidByName}}}},
		"PaidByMemberID": notionapi.RichTextProperty{RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: req.PaidByMemberID}}}},
		"SplitWith":      notionapi.RichTextProperty{RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: string(splitWithJSON)}}}},
		"Items":          notionapi.RichTextProperty{RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: string(itemsJSON)}}}},
	}

	page, err := c.api.Page.Create(ctx, &notionapi.PageCreateRequest{
		Parent:     notionapi.Parent{Type: notionapi.ParentTypeDatabaseID, DatabaseID: notionapi.DatabaseID(dbID)},
		Properties: properties,
	})
	if err != nil {
		return nil, fmt.Errorf("notion create record: %w", err)
	}

	rec, err := pageToRecord(*page)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// UpdateRecord updates an existing expense record page in Notion.
// Only non-nil fields in req are written; nil fields are left unchanged.
func (c *Client) UpdateRecord(ctx context.Context, pageID string, req model.UpdateRecordRequest) (*model.Record, error) {
	props := notionapi.Properties{}

	if req.Store != nil {
		props["Store"] = notionapi.TitleProperty{
			Title: []notionapi.RichText{{Text: &notionapi.Text{Content: *req.Store}}},
		}
	}
	if req.Date != nil {
		props["Date"] = notionapi.DateProperty{
			Date: &notionapi.DateObject{Start: toNotionDate(*req.Date)},
		}
	}
	if req.AmountJPY != nil {
		props["Amount_JPY"] = notionapi.NumberProperty{Number: *req.AmountJPY}
	}
	if req.AmountTWD != nil {
		props["Amount_TWD"] = notionapi.NumberProperty{Number: *req.AmountTWD}
	}
	if req.TaxJPY != nil {
		props["Tax_JPY"] = notionapi.NumberProperty{Number: *req.TaxJPY}
	}
	if req.Category != nil {
		props["Category"] = notionapi.SelectProperty{Select: notionapi.Option{Name: *req.Category}}
	}
	if req.Payment != nil {
		props["Payment"] = notionapi.SelectProperty{Select: notionapi.Option{Name: *req.Payment}}
	}
	if req.PaidBy != nil {
		props["PaidBy"] = notionapi.RichTextProperty{RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: *req.PaidBy}}}}
	}
	if req.PaidByName != nil {
		props["PaidByName"] = notionapi.RichTextProperty{RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: *req.PaidByName}}}}
	}
	if req.PaidByMemberID != nil {
		props["PaidByMemberID"] = notionapi.RichTextProperty{RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: *req.PaidByMemberID}}}}
	}
	if req.SplitWith != nil {
		b, _ := json.Marshal(req.SplitWith)
		props["SplitWith"] = notionapi.RichTextProperty{RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: string(b)}}}}
	}
	if req.Items != nil {
		b, _ := json.Marshal(req.Items)
		props["Items"] = notionapi.RichTextProperty{RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: string(b)}}}}
	}

	page, err := c.api.Page.Update(ctx, notionapi.PageID(pageID), &notionapi.PageUpdateRequest{
		Properties: props,
	})
	if err != nil {
		return nil, fmt.Errorf("notion update record: %w", err)
	}

	rec, err := pageToRecord(*page)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// DeleteRecord archives a Notion page (moves it to trash).
func (c *Client) DeleteRecord(ctx context.Context, pageID string) error {
	_, err := c.api.Page.Update(ctx, notionapi.PageID(pageID), &notionapi.PageUpdateRequest{
		Archived: true,
	})
	if err != nil {
		return fmt.Errorf("notion delete record: %w", err)
	}
	return nil
}

// --- Settlement export ---

// SettlementData holds the data needed to render the settlement page.
type SettlementData struct {
	Settlements []model.Settlement
	Balances    []model.MemberBalance
	TotalJPY    float64
}

// CreateSettlementPage creates a rich settlement summary page under parentPageID.
// Returns the URL of the created page.
func (c *Client) CreateSettlementPage(ctx context.Context, parentPageID, tripName string, data SettlementData) (string, error) {
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

func buildSettlementBlocks(data SettlementData) []notionapi.Block {
	blocks := []notionapi.Block{}

	// Heading: 結算明細
	blocks = append(blocks, &notionapi.Heading2Block{
		BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeHeading2},
		Heading2:   notionapi.Heading{RichText: richText("結算明細")},
	})

	if len(data.Settlements) == 0 {
		blocks = append(blocks, paragraph("（無需轉帳）"))
	} else {
		for _, s := range data.Settlements {
			line := fmt.Sprintf("%s → %s　¥%s", s.FromUserName, s.ToUserName, formatJPY(s.Amount))
			blocks = append(blocks, &notionapi.BulletedListItemBlock{
				BasicBlock:       notionapi.BasicBlock{Type: notionapi.BlockTypeBulletedListItem},
				BulletedListItem: notionapi.ListItem{RichText: richText(line)},
			})
		}
	}

	blocks = append(blocks, &notionapi.DividerBlock{
		BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeDivider},
		Divider:    notionapi.Divider{},
	})

	// Heading: 各人花費
	blocks = append(blocks, &notionapi.Heading2Block{
		BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeHeading2},
		Heading2:   notionapi.Heading{RichText: richText("各人花費")},
	})

	for _, b := range data.Balances {
		blocks = append(blocks, &notionapi.Heading3Block{
			BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeHeading3},
			Heading3:   notionapi.Heading{RichText: richText(b.UserName)},
		})

		paidLine := fmt.Sprintf("實際付出：¥%s", formatJPY(b.TotalPaid))
		shouldLine := fmt.Sprintf("應付份額：¥%s", formatJPY(b.ShouldPay))
		balanceSign := "+"
		if b.Balance < 0 {
			balanceSign = ""
		}
		balanceLine := fmt.Sprintf("差額：%s¥%s", balanceSign, formatJPY(b.Balance))

		for _, line := range []string{paidLine, shouldLine, balanceLine} {
			blocks = append(blocks, &notionapi.BulletedListItemBlock{
				BasicBlock:       notionapi.BasicBlock{Type: notionapi.BlockTypeBulletedListItem},
				BulletedListItem: notionapi.ListItem{RichText: richText(line)},
			})
		}
	}

	return blocks
}

// --- helpers ---

func richText(s string) []notionapi.RichText {
	return []notionapi.RichText{{Text: &notionapi.Text{Content: s}}}
}

func paragraph(s string) *notionapi.ParagraphBlock {
	return &notionapi.ParagraphBlock{
		BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeParagraph},
		Paragraph:  notionapi.Paragraph{RichText: richText(s)},
	}
}

func formatJPY(v float64) string {
	if v < 0 {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.0f", v)
}

func pageToRecord(page notionapi.Page) (model.Record, error) {
	props := page.Properties
	rec := model.Record{ID: string(page.ID)}

	if p, ok := props["Store"].(*notionapi.TitleProperty); ok && len(p.Title) > 0 {
		rec.Store = p.Title[0].PlainText
	}
	if p, ok := props["Date"].(*notionapi.DateProperty); ok && p.Date != nil && p.Date.Start != nil {
		rec.Date = formatDate(p.Date.Start)
	}
	if p, ok := props["Amount_JPY"].(*notionapi.NumberProperty); ok {
		rec.AmountJPY = p.Number
	}
	if p, ok := props["Amount_TWD"].(*notionapi.NumberProperty); ok {
		rec.AmountTWD = p.Number
	}
	if p, ok := props["Tax_JPY"].(*notionapi.NumberProperty); ok {
		rec.TaxJPY = p.Number
	}
	if p, ok := props["Category"].(*notionapi.SelectProperty); ok {
		rec.Category = p.Select.Name
	}
	if p, ok := props["Payment"].(*notionapi.SelectProperty); ok {
		rec.Payment = p.Select.Name
	}
	if p, ok := props["PaidBy"].(*notionapi.RichTextProperty); ok && len(p.RichText) > 0 {
		rec.PaidBy = p.RichText[0].PlainText
	}
	if p, ok := props["PaidByName"].(*notionapi.RichTextProperty); ok && len(p.RichText) > 0 {
		rec.PaidByName = p.RichText[0].PlainText
	}
	if p, ok := props["PaidByMemberID"].(*notionapi.RichTextProperty); ok && len(p.RichText) > 0 {
		rec.PaidByMemberID = p.RichText[0].PlainText
	}
	if p, ok := props["SplitWith"].(*notionapi.RichTextProperty); ok && len(p.RichText) > 0 {
		_ = json.Unmarshal([]byte(p.RichText[0].PlainText), &rec.SplitWith)
	}
	if p, ok := props["Items"].(*notionapi.RichTextProperty); ok && len(p.RichText) > 0 {
		_ = json.Unmarshal([]byte(p.RichText[0].PlainText), &rec.Items)
	}

	return rec, nil
}

func toNotionDate(s string) *notionapi.Date {
	t, _ := time.Parse("2006-01-02", s)
	d := notionapi.Date(t)
	return &d
}

func formatDate(d *notionapi.Date) string {
	return time.Time(*d).Format("2006-01-02")
}
