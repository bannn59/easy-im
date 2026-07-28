# Design: 功能开发路线图（文档）

## Purpose

Single source for “what to build next” after feature-map + scaffold. Consumed by humans and future Trellis tasks.

## Inputs

| Source | Role |
|--------|------|
| Archived feature map research | User-perceptible groups, empty-state evidence |
| Current `backend/` / `frontend/` | Calibrate partial health/shell |
| `.trellis/spec/backend/realtime-messaging.md` etc. | Ordering constraints (HTTP before multi-node WS) |
| Prior session plan (chat) | P0–P6 narrative already reviewed by user |

## Output layout

```text
research/
├── index.md       # nav, calibration summary, next action
└── roadmap.md     # full phased plan
```

## Roadmap structure (roadmap.md)

1. Principles  
2. Map calibration (after scaffold)  
3. Phases P0–P6 with tables (slice / user value / backend / frontend / exit)  
4. Suggested Trellis task sequence  
5. Milestones M0–M5  
6. Explicit non-goals  
7. Default next slice  

## Non-goals

- Code, migrations, compose files in this task.
- Binding the team to calendar dates (order only, not week estimates unless asked).
