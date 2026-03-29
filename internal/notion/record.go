package notion

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jomei/notionapi"

	"github.com/yourname/tabikake/internal/model"
)

// ListRecords returns all expense records from the given database.
func (c *Client) ListRecords(ctx context.Context, dbID string) ([]model.Record, error) {
	resp, err := c.api.Database.Query(ctx, notionapi.DatabaseID(dbID), &notionapi.DatabaseQueryRequest{})
	if err != nil {
		return nil, fmt.Errorf("notion list records: %w", err)
	}

	records := make([]model.Record, 0, len(resp.Results))
	for _, page := range resp.Results {
		rec := pageToRecord(page)
		records = append(records, rec)
	}
	return records, nil
}

// CreateRecord creates a new expense record page in the given database.
func (c *Client) CreateRecord(ctx context.Context, dbID string, req model.CreateRecordRequest) (*model.Record, error) {
	props := notionapi.Properties{
		"StoreNameZH": notionapi.TitleProperty{
			Title: []notionapi.RichText{{Text: &notionapi.Text{Content: req.StoreNameZH}}},
		},
		"StoreNameJP":  notionapi.RichTextProperty{RichText: richText(req.StoreNameJP)},
		"Date":         notionapi.DateProperty{Date: &notionapi.DateObject{Start: toNotionDate(req.Date)}},
		"Amount_JPY":   notionapi.NumberProperty{Number: req.AmountJPY},
		"Tax_JPY":      notionapi.NumberProperty{Number: req.TaxJPY},
		"Category":     notionapi.SelectProperty{Select: notionapi.Option{Name: req.Category}},
		"Payment":      notionapi.SelectProperty{Select: notionapi.Option{Name: req.Payment}},
		"PaidByUserID": notionapi.RichTextProperty{RichText: richText(req.PaidByUserID)},
		"SplitWith":    notionapi.RichTextProperty{RichText: richText(marshalJSON(req.SplitWith))},
		"Items":        notionapi.RichTextProperty{RichText: richText(marshalJSON(req.Items))},
	}

	page, err := c.api.Page.Create(ctx, &notionapi.PageCreateRequest{
		Parent:     notionapi.Parent{Type: notionapi.ParentTypeDatabaseID, DatabaseID: notionapi.DatabaseID(dbID)},
		Properties: props,
	})
	if err != nil {
		return nil, fmt.Errorf("notion create record: %w", err)
	}

	rec := pageToRecord(*page)
	return &rec, nil
}

// UpdateRecord updates non-nil fields of an existing record page.
func (c *Client) UpdateRecord(ctx context.Context, pageID string, req model.UpdateRecordRequest) (*model.Record, error) {
	props := notionapi.Properties{}

	if req.StoreNameZH != nil {
		props["StoreNameZH"] = notionapi.TitleProperty{
			Title: []notionapi.RichText{{Text: &notionapi.Text{Content: *req.StoreNameZH}}},
		}
	}
	if req.StoreNameJP != nil {
		props["StoreNameJP"] = notionapi.RichTextProperty{RichText: richText(*req.StoreNameJP)}
	}
	if req.Date != nil {
		props["Date"] = notionapi.DateProperty{Date: &notionapi.DateObject{Start: toNotionDate(*req.Date)}}
	}
	if req.AmountJPY != nil {
		props["Amount_JPY"] = notionapi.NumberProperty{Number: *req.AmountJPY}
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
	if req.PaidByUserID != nil {
		props["PaidByUserID"] = notionapi.RichTextProperty{RichText: richText(*req.PaidByUserID)}
	}
	if req.SplitWith != nil {
		props["SplitWith"] = notionapi.RichTextProperty{RichText: richText(marshalJSON(req.SplitWith))}
	}
	if req.Items != nil {
		props["Items"] = notionapi.RichTextProperty{RichText: richText(marshalJSON(req.Items))}
	}

	page, err := c.api.Page.Update(ctx, notionapi.PageID(pageID), &notionapi.PageUpdateRequest{
		Properties: props,
	})
	if err != nil {
		return nil, fmt.Errorf("notion update record: %w", err)
	}
	rec := pageToRecord(*page)
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

func pageToRecord(page notionapi.Page) model.Record {
	props := page.Properties
	rec := model.Record{ID: string(page.ID)}

	if p, ok := props["StoreNameZH"].(*notionapi.TitleProperty); ok && len(p.Title) > 0 {
		rec.StoreNameZH = p.Title[0].PlainText
	}
	if p, ok := props["StoreNameJP"].(*notionapi.RichTextProperty); ok && len(p.RichText) > 0 {
		rec.StoreNameJP = p.RichText[0].PlainText
	}
	if p, ok := props["Date"].(*notionapi.DateProperty); ok && p.Date != nil && p.Date.Start != nil {
		rec.Date = formatDate(p.Date.Start)
	}
	if p, ok := props["Amount_JPY"].(*notionapi.NumberProperty); ok {
		rec.AmountJPY = p.Number
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
	if p, ok := props["PaidByUserID"].(*notionapi.RichTextProperty); ok && len(p.RichText) > 0 {
		rec.PaidByUserID = p.RichText[0].PlainText
	}
	if p, ok := props["SplitWith"].(*notionapi.RichTextProperty); ok && len(p.RichText) > 0 {
		_ = json.Unmarshal([]byte(p.RichText[0].PlainText), &rec.SplitWith)
	}
	if p, ok := props["Items"].(*notionapi.RichTextProperty); ok && len(p.RichText) > 0 {
		_ = json.Unmarshal([]byte(p.RichText[0].PlainText), &rec.Items)
	}
	return rec
}
