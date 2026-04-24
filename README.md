# KYC Merchant Onboarding Workflow Service

A full-stack state-machine-driven KYC workflow system with a Go backend and React frontend.

---

## Prerequisites

| Tool       | Version  | Check               |
|------------|----------|----------------------|
| Go         | 1.22+    | `go version`         |
| Node.js    | 18+      | `node -v`            |
| npm        | 9+       | `npm -v`             |
| PostgreSQL | 14+      | `psql --version`     |

---

## 1. Database Setup

### Option A: Local PostgreSQL (Recommended)

```bash
# Start PostgreSQL if not running
sudo systemctl start postgresql
# OR on macOS: brew services start postgresql

# Create the database
createdb kyc_dev

# Verify connection
psql kyc_dev -c "SELECT 1;"
```

### Option B: Docker PostgreSQL

```bash
docker run -d \
  --name kyc-postgres \
  -e POSTGRES_USER=sarthak \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=kyc_dev \
  -p 5432:5432 \
  postgres:16-alpine

# Connection string for this:
# postgres://sarthak:postgres@localhost:5432/kyc_dev?sslmode=disable
```

> **Note:** You do NOT need to run the migration SQL manually. The server runs `migrations/001_init.sql` automatically on startup.

---

## 2. Backend Setup

```bash
cd backend

# Install Go dependencies
go mod tidy

# Create your .env file
cp .env.example .env
```

### Edit `.env`

Open `backend/.env` and set your `DATABASE_URL` to match your PostgreSQL setup:

```env
# If your local postgres user has no password:
DATABASE_URL=postgres://sarthak:@localhost:5432/kyc_dev?sslmode=disable

# If using Docker setup from above:
DATABASE_URL=postgres://sarthak:postgres@localhost:5432/kyc_dev?sslmode=disable

# Generate a random JWT secret (or just use a long string):
JWT_SECRET=super-secret-jwt-key-that-is-at-least-32-characters-long
```

### Run the Server

```bash
# Load env vars and start
export $(cat .env | xargs) && go run ./cmd/server
```

You should see:

```
Connected to database
Migrations complete
Checking if seed data is needed...
Seeding database...
  Created reviewer: reviewer@kyc.dev
  Created merchant: merchant.draft@kyc.dev (draft submission)
  Created merchant: merchant.review@kyc.dev (under_review submission + 3 docs)
Seed complete.
Storage backend: local
Server starting on :8080
```

### Quick API Test

```bash
# Login with seeded reviewer account
curl -s http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"email":"reviewer@kyc.dev","password":"password123"}' | jq
```

Expected: `{ "token": "eyJ..." }`

---

## 3. Frontend Setup

```bash
cd frontend

# Install dependencies
npm install

# Start dev server
npm run dev
```

Opens at **http://localhost:3000**. The Vite dev server proxies `/api/*` to `localhost:8080` automatically.

---

## 4. Demo Accounts (Auto-Seeded)

| Email                     | Password      | Role     | State         |
|---------------------------|---------------|----------|---------------|
| `reviewer@kyc.dev`        | `password123` | Reviewer | —             |
| `merchant.draft@kyc.dev`  | `password123` | Merchant | `draft`       |
| `merchant.review@kyc.dev` | `password123` | Merchant | `under_review`|

---

## 5. Usage Walkthrough

### As a Merchant

1. Go to http://localhost:3000/login
2. Login with `merchant.draft@kyc.dev` / `password123`
3. Fill in the 3-step KYC form (Personal → Business → Documents)
4. Click **Save as Draft** at any point to persist progress
5. Upload all 3 documents (PAN, Aadhaar, Bank Statement) — any PDF/JPG/PNG under 5MB
6. Click **Submit for Review**

### As a Reviewer

1. Login with `reviewer@kyc.dev` / `password123`
2. View the **Review Queue** with metrics (queue size, avg wait, approval rate)
3. Click a submission to see full details
4. Click **Start Review** to move it to `under_review`
5. Then **Approve**, **Reject** (note required), or **Request Info** (note required)

---

## Environment Variables Reference

| Variable           | Required | Default     | Description                           |
|--------------------|----------|-------------|---------------------------------------|
| `DATABASE_URL`     | ✅ Yes   | —           | PostgreSQL connection string          |
| `JWT_SECRET`       | ✅ Yes   | —           | JWT signing key (min 32 chars)        |
| `PORT`             | No       | `8080`      | HTTP server port                      |
| `STORAGE_BACKEND`  | No       | `local`     | `local` or `s3`                       |
| `STORAGE_ROOT`     | No       | `./uploads` | Directory for local file storage      |
| `AWS_BUCKET`       | No       | —           | S3 bucket (required if backend = s3)  |
| `AWS_REGION`       | No       | —           | AWS region (required if backend = s3) |

---

## Running Tests

```bash
cd backend
go test ./services/... -v
```

---

## Project Structure

```
kyc/
├── backend/
│   ├── cmd/server/main.go      ← entry point
│   ├── config/                  ← env config
│   ├── handlers/                ← thin HTTP layer
│   ├── services/                ← business logic + state machine
│   ├── store/                   ← database queries
│   ├── storage/                 ← file storage interface
│   ├── middleware/               ← auth, logging, recovery
│   ├── models/                  ← data structs
│   ├── migrations/              ← SQL schema
│   ├── seed/                    ← demo data
│   └── .env.example
├── frontend/
│   ├── src/
│   │   ├── api/client.js        ← axios + JWT
│   │   ├── pages/               ← Login, Signup, Dashboards
│   │   └── components/KYCForm/  ← multi-step form
│   ├── index.html
│   └── vite.config.js
├── AGENTS.md
└── README.md
```
