# Implement: 功能开发路线图

## Checklist

1. [x] Write `research/roadmap.md` (P0–P6, milestones, task split, non-goals).
2. [x] Write `research/index.md` (nav + calibration + next action).
3. [x] Tick PRD AC.
4. [x] `git status` — only task files (and optional journal later).

## Validation

```bash
test -f .trellis/tasks/07-28-feature-dev-roadmap/research/index.md
test -f .trellis/tasks/07-28-feature-dev-roadmap/research/roadmap.md
rg -n "P0|P3|M4|下一步" .trellis/tasks/07-28-feature-dev-roadmap/research/
git status --porcelain
```

## Note

No `trellis-implement` for product code. Main session writes research markdown.
