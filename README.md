# Tabikake Backend

Go backend for **Tabikake（旅掛け）** — a travel expense tracking app.
Users photograph receipts in Japan; Claude Vision OCR parses them and writes records to Notion.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Framework | [Echo v4](https://echo.labstack.com/) |
| AI / OCR | [Anthropic Claude Vision](https://docs.anthropic.com/) (`claude-sonnet-4-6`) |
| Database (records) | [Notion API](https://developers.notion.com/) — one database per trip |
| Database (trips) | SQLite via `modernc.org/sqlite` (pure Go, no CGO) |
| Auth | Notion OAuth 2.0 → JWT (HS256) |

## Architecture

```
NOTION_ROOT_PAGE_ID  (manually created once)
└── {Trip Name}                ← created on POST /trips
    ├── Records (Database)     ← schema auto-generated
    └── {Trip} 結算 - YYYY/MM/DD  ← created on POST /split/export/:trip_id
```

Trip metadata (`id`, `notion_page_id`, `notion_db_id`) is stored in local SQLite.
All record data lives in each trip's own Notion database.

## Project Structure

```
Backend/
├── cmd/server/main.go          # Entry point, Echo setup, routing
├── config/config.go            # Env var loading + validation
├── internal/
│   ├── model/model.go          # All request/response structs
│   ├── db/db.go                # SQLite (trips table)
│   ├── claude/claude.go        # Claude Vision wrapper
│   ├── notion/notion.go        # Notion API wrapper
│   ├── middleware/             # JWT auth, CORS, logging
│   ├── service/                # Business logic
│   │   ├── auth.go             # Notion OAuth exchange + JWT issuance
│   │   ├── parse.go            # Receipt OCR via Claude
│   │   ├── trip.go             # Trip CRUD (SQLite + Notion)
│   │   ├── record.go           # Expense record CRUD
│   │   ├── dashboard.go        # Aggregation + greedy settlement
│   │   └── split.go            # Settlement export to Notion
│   └── handler/                # HTTP handlers
│       ├── auth.go
│       ├── parse.go
│       ├── trip.go
│       ├── record.go
│       ├── dashboard.go
│       └── split.go
├── tabikake.postman_collection.json
├── .env.example
└── go.mod
```

## API Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/health` | — | Health check |
| `POST` | `/auth/notion/callback` | — | Notion OAuth code → JWT |
| `GET` | `/auth/me` | JWT | Current user info |
| `GET` | `/trips` | JWT | List all trips |
| `POST` | `/trips` | JWT | Create trip (auto-creates Notion page + DB) |
| `GET` | `/trips/:id` | JWT | Trip details |
| `POST` | `/parse` | JWT | OCR receipt image → JSON (does **not** write to Notion) |
| `GET` | `/records?trip_id=` | JWT | List expense records for a trip |
| `POST` | `/records` | JWT | Write confirmed record to Notion |
| `GET` | `/dashboard/:trip_id` | JWT | Spending summary + settlement |
| `POST` | `/split/export/:trip_id` | JWT | Create settlement page in Notion |

## Setup

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- A Notion account with an integration set up at [notion.so/my-integrations](https://www.notion.so/my-integrations)
- An [Anthropic API key](https://console.anthropic.com/)

### 1. Clone and install dependencies

```bash
cd Backend
go mod tidy
```

### 2. Configure environment

```bash
cp .env.example .env
```

Fill in `.env`:

```env
ANTHROPIC_API_KEY=sk-ant-...
NOTION_INTEGRATION_TOKEN=ntn_...
NOTION_OAUTH_CLIENT_ID=
NOTION_OAUTH_CLIENT_SECRET=
NOTION_ROOT_PAGE_ID=          # Notion page ID under which all trips are created
JWT_SECRET=                   # openssl rand -base64 32
PORT=8080
FRONTEND_URL=http://localhost:3000
SQLITE_PATH=tabikake.db
```

### 3. Notion setup

1. Create a root page in Notion (e.g. "Tabikake")
2. Open the page → `...` → **Connections** → add your integration
3. Copy the page ID from the URL and set `NOTION_ROOT_PAGE_ID`
4. Add `https://oauth.pstmn.io/v1/vscode-callback` to your integration's **Redirect URIs** (for Postman testing)

### 4. Run

```bash
go run ./cmd/server
```

Server starts on `http://localhost:8080`.

## Authentication Flow

```
Frontend  →  GET https://api.notion.com/v1/oauth/authorize?...
          ←  Notion redirects with ?code=...
Frontend  →  POST /auth/notion/callback  { "code": "..." }
          ←  { "access_token": "<JWT>", "user": { ... } }
Frontend     uses JWT as Bearer token for all subsequent requests
```

## POST /parse

Accepts a receipt image and returns structured JSON **without writing to Notion**.
Use the response to pre-fill the confirmation form; then call `POST /records` to persist.

**Multipart upload:**
```bash
curl -X POST http://localhost:8080/parse \
  -H "Authorization: Bearer $TOKEN" \
  -F "image=@receipt.jpg"
```

**Base64 JSON:**
```bash
curl -X POST http://localhost:8080/parse \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"image_base64": "data:image/jpeg;base64,..."}'
```

Response:
```json
{
  "store_name_jp": "すき家",
  "store_name_zh": "吉野家",
  "amount_jpy": 650,
  "tax_jpy": 59,
  "payment_method": "現金",
  "category": "餐飲",
  "items": [{ "name_jp": "牛丼", "name_zh": "牛肉飯", "price": 650 }],
  "date": "2026-03-27"
}
```

## POST /records — Field Reference

```json
{
  "trip_id":      "abc123",
  "store":        "すき家",
  "date":         "2026-03-27",
  "amount_jpy":   650,
  "amount_twd":   145,
  "tax_jpy":      59,
  "category":     "餐飲",
  "payment":      "現金",
  "paid_by":      "notion-user-id",
  "paid_by_name": "劉冠呈",
  "split_with":   ["other-user-id"],
  "items": [
    { "name_jp": "牛丼", "name_zh": "牛肉飯", "price": 650 }
  ]
}
```

Valid `category` values: `餐飲` `交通` `購物` `住宿` `其他`
Valid `payment` values: `現金` `Suica` `PayPay` `信用卡`

## Settlement Algorithm

`GET /dashboard/:trip_id` and `POST /split/export/:trip_id` both use a greedy algorithm to minimize the number of transfers:

1. Calculate each member's `total_paid - (total / members)`
2. Positive balance → creditor; negative → debtor
3. Greedily match largest creditor with largest debtor until all settled

## Testing

Import `tabikake.postman_collection.json` into Postman (VSCode extension or desktop app).

The collection's **Authorization** tab is pre-configured for OAuth 2.0:
- Auth URL: `https://api.notion.com/v1/oauth/authorize?owner=user`
- Access Token URL: `http://localhost:8080/auth/notion/callback`
- Grant Type: Authorization Code
