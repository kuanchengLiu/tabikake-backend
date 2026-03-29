# Tabikake Backend

Go backend for **Tabikake（旅掛け）** — a Japan travel expense tracking app.
Users photograph receipts; Claude Vision OCR parses them and writes structured records to Notion.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Framework | [Echo v4](https://echo.labstack.com/) |
| AI / OCR | [Anthropic Claude Vision](https://docs.anthropic.com/) (`claude-sonnet-4-6`) |
| Records DB | [Notion API](https://developers.notion.com/) — one database per trip |
| App DB | SQLite via `modernc.org/sqlite` (pure Go, no CGO) |
| Auth | Notion OAuth 2.0 → httpOnly cookie (JWT HS256 + server-side sessions) |

## Architecture

```
NOTION_ROOT_PAGE_ID  (manually created once)
└── {Trip Name}                  ← created on POST /trips
    ├── Records (Database)       ← schema auto-generated
    └── {Trip} 結算 - YYYY/MM/DD ← created on POST /trips/:id/settlement/export
```

SQLite stores: `users`, `sessions`, `trips`, `members`.
All expense records live in each trip's Notion database.
Notion access tokens are encrypted (AES-256-GCM) before storage.

## Project Structure

```
Backend/
├── cmd/server/main.go          # Entry point, Echo setup, routing
├── config/config.go            # Env var loading + validation
├── internal/
│   ├── model/                  # Shared types (user, trip, member, record, …)
│   ├── store/                  # SQLite queries (users, sessions, trips, members)
│   ├── claude/claude.go        # Claude Vision OCR wrapper
│   ├── notion/                 # Notion API (page/DB creation, records, settlement page)
│   ├── middleware/             # Cookie JWT auth, CORS, logging
│   ├── service/                # Business logic
│   │   ├── auth.go             # Notion OAuth → session + encrypted token + JWT cookie
│   │   ├── trip.go             # Trip CRUD (SQLite + Notion)
│   │   ├── member.go           # Member management
│   │   ├── record.go           # Expense record CRUD + receipt OCR
│   │   ├── dashboard.go        # Spending aggregation
│   │   └── settlement.go       # Greedy settlement algorithm + Notion export
│   └── handler/                # HTTP handlers
│       ├── auth.go
│       ├── trip.go
│       ├── member.go
│       ├── record.go
│       ├── dashboard.go
│       └── settlement.go
├── tabikake.postman_collection.json
├── .env.example
└── go.mod
```

## API Routes

### Public (no auth)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/auth/notion/url` | Get Notion OAuth authorization URL |
| `GET` | `/auth/notion/callback?code=` | Exchange OAuth code → set httpOnly cookie |
| `GET` | `/trips/join-info?code=` | Preview trip info before joining |

### Protected (requires auth cookie)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/auth/me` | Current user profile |
| `POST` | `/auth/logout` | Delete session + clear cookie |
| `GET` | `/trips` | List trips where the user is a member |
| `POST` | `/trips` | Create trip (auto-creates Notion page + DB, adds creator as owner) |
| `GET` | `/trips/:id` | Trip details + `is_member` for current user |
| `PATCH` | `/trips/:id` | Update trip fields (owner only) |
| `DELETE` | `/trips/:id` | Delete trip (owner only) |
| `POST` | `/trips/join` | Join a trip via invite code |
| `GET` | `/trips/:id/members` | List trip members with user profiles |
| `DELETE` | `/trips/:id/members/:user_id` | Remove member by Notion user ID (owner only) |
| `GET` | `/trips/:id/settlement` | Calculate settlement (no write) |
| `POST` | `/trips/:id/settlement/export` | Calculate + create Notion summary page |
| `GET` | `/records?trip_id=` | List expense records for a trip |
| `POST` | `/records` | Write confirmed expense record to Notion |
| `PATCH` | `/records/:id` | Update a Notion record page |
| `DELETE` | `/records/:id` | Archive a Notion record page |
| `POST` | `/parse` | OCR receipt image → structured JSON (does **not** write to Notion) |
| `GET` | `/dashboard/:trip_id` | Spending summary by member, category, and date |

## Setup

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- A Notion account with an OAuth integration at [notion.so/my-integrations](https://www.notion.so/my-integrations)
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
NOTION_OAUTH_CLIENT_ID=          # from your Notion integration
NOTION_OAUTH_CLIENT_SECRET=      # from your Notion integration
NOTION_OAUTH_REDIRECT_URI=http://localhost:8080/auth/notion/callback
NOTION_ROOT_PAGE_ID=             # Notion page ID under which all trips are created
JWT_SECRET=                      # openssl rand -base64 32
TOKEN_ENCRYPT_KEY=               # openssl rand -base64 32  (encrypts stored access tokens)
PORT=8080
FRONTEND_URL=http://localhost:3000
SQLITE_PATH=tabikake.db
```

### 3. Notion setup

1. Create a root page in Notion (e.g. "Tabikake")
2. Open the page → `...` → **Connections** → add your integration
3. Copy the page ID from the URL and set `NOTION_ROOT_PAGE_ID`
4. In your integration's OAuth settings, add `http://localhost:8080/auth/notion/callback` to **Redirect URIs**

### 4. Run

```bash
go run ./cmd/server
```

Server starts on `http://localhost:8080`.

## Authentication Flow

```
1. Frontend   →  GET /auth/notion/url
              ←  { "url": "https://api.notion.com/v1/oauth/authorize?..." }

2. Browser opens the URL → user authorizes in Notion
   Notion redirects to: GET http://localhost:8080/auth/notion/callback?code=...

3. Backend exchanges code → creates session → signs JWT
   Response: 302 redirect to FRONTEND_URL
             Set-Cookie: auth_token=<jwt>; HttpOnly; SameSite=Lax

4. All subsequent requests send the cookie automatically (browser) or via
   Postman's cookie jar (Postman).
```

## POST /parse

Accepts a receipt image and returns structured JSON **without writing to Notion**.
Use the response to pre-fill the confirmation form; then call `POST /records` to persist.

**Multipart upload:**
```bash
curl -X POST http://localhost:8080/parse \
  -b "auth_token=$TOKEN" \
  -F "image=@receipt.jpg"
```

**Base64 JSON:**
```bash
curl -X POST http://localhost:8080/parse \
  -b "auth_token=$TOKEN" \
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
  "items": [{ "name_zh": "牛肉飯", "price": 650 }],
  "date": "2026-03-27"
}
```

## POST /records — Field Reference

```json
{
  "trip_id":         "uuid",
  "store_name_zh":   "吉野家",
  "store_name_jp":   "すき家",
  "date":            "2026-03-27",
  "amount_jpy":      650,
  "tax_jpy":         59,
  "category":        "餐飲",
  "payment":         "現金",
  "paid_by_user_id": "notion-user-id",
  "split_with":      ["notion-user-id-2"],
  "items": [
    { "name_zh": "牛肉飯", "price": 650 }
  ]
}
```

- `paid_by_user_id` — defaults to the authenticated user if omitted
- `split_with` — empty array means split equally among **all trip members** (AA制)

Valid `category`: `餐飲` `交通` `購物` `住宿` `其他`
Valid `payment`: `現金` `Suica` `PayPay` `信用卡`

## Settlement Algorithm

`GET /trips/:id/settlement` and `POST /trips/:id/settlement/export` use a greedy algorithm to minimise the number of transfers:

1. For each record: add `amount_jpy` to payer's "paid" total; divide equally (or per `split_with`) among the responsible members
2. `balance = paid − owe` → positive = creditor, negative = debtor
3. Sort both sides descending; greedily match until all settled

**Export** additionally creates a Notion page under the trip with a per-user record breakdown.

## Testing with Postman

Import `tabikake.postman_collection.json` into Postman.

**Auth flow:**
1. Call `GET /auth/notion/url` → copy the returned URL
2. Open that URL in a browser → authorize with your Notion account
3. Notion redirects to `http://localhost:8080/auth/notion/callback?code=...`
   The server sets the `auth_token` cookie and redirects to the frontend
4. Back in Postman: call `GET /auth/notion/callback` with the `code` query param — Postman's cookie jar captures the `auth_token` cookie automatically
5. All subsequent Postman requests to `localhost:8080` will include the cookie
