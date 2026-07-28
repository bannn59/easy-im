# easy-im frontend

Vite + React + TypeScript client shell.

## Setup

```bash
cd frontend
cp .env.example .env   # optional
npm install
npm run dev
```

## Scripts

| Command | Purpose |
|---------|---------|
| `npm run dev` | Dev server (default http://localhost:5173) |
| `npm run build` | Typecheck + production build |
| `npm run typecheck` | `tsc -b` only |
| `npm run preview` | Preview production build |

## Layout

See `.trellis/spec/frontend/directory-structure.md`.

- `src/app` — router + shell pages
- `src/api` — HTTP helpers (`VITE_API_BASE`)
- `src/realtime` — WS placeholder (no socket yet)
- `src/features` — future product features

No IM product features in this scaffold.
