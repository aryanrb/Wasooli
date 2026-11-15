# Wasooli — Frontend

Mobile-first React SPA for the rental-income ledger. Talks to the Go backend's REST API.

## Setup

```bash
npm install
cp .env.example .env   # optional — defaults to http://localhost:8080
npm run dev            # http://localhost:5173
npm run build          # type-check + production build
```

## Environment

- `VITE_API_URL` — backend base URL (default `http://localhost:8080`). All API
  paths are under `${VITE_API_URL}/api/v1`.

## Backend requirements

- The Go backend must be running (default `:8080`).
- Auth is an httpOnly session cookie set by Google OAuth, so the backend's
  `FRONTEND_URL` **must exactly match the Vite dev origin** (`http://localhost:5173`)
  for CORS + credentialed cookies to work. Change both together if you use a
  different host/port.

## Stack

Vite + React + TypeScript, react-router-dom, Tailwind CSS v4 (`@tailwindcss/vite`).
No UI component library, no state-management library — plain hooks and a small
fetch wrapper (`src/api/client.ts`).
