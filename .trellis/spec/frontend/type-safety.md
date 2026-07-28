# Type Safety

> TypeScript conventions for easy-im frontend.

---

## Bootstrap status

Strict TypeScript is assumed (`strict: true`). Protocol types should eventually come from monorepo contracts shared with the Go backend.

---

## Type organization

| Kind | Location |
|------|----------|
| Wire DTOs (HTTP/WS) | Generated or `packages/contracts` → imported by `api/` and `realtime/` |
| Feature view models | `features/<feature>/types.ts` if they differ from wire types |
| Shared UI prop types | Colocated with components |
| Brand / ID types | Optional: `type UserId = string & { readonly __brand: 'UserId' }` |

Transform wire → view model at the boundary (`api` or feature hook), not inside every presentational component.

---

## Runtime validation

TypeScript types are erased. At trust boundaries:

| Boundary | Validate |
|----------|----------|
| HTTP response | Parse with schema (Zod or equivalent) **or** trust generated clients with narrow types + smoke tests |
| WS frames | Decode `unknown` → typed frame via discriminators on `type` / `v` |
| `localStorage` | Parse defensively |

```ts
// bootstrap sketch
function isServerMessageEvent(x: unknown): x is ServerMessageEvent {
  return (
    typeof x === 'object' &&
    x !== null &&
    (x as { type?: string }).type === 'message.created'
  );
}
```

Centralize guards next to frame types in `realtime/`. Features import guards; they do not cast `as Message` ad hoc.

---

## Forbidden patterns

| Pattern | Prefer |
|---------|--------|
| `any` | `unknown` + narrow |
| `as Message` on raw JSON | type guard / schema parse |
| `// @ts-expect-error` without reason | fix types or narrow scope with comment |
| Optional fields used as required without checks | normalize at boundary |
| Divergent hand-written DTO copies | shared contracts |

---

## Error types

Map API error payloads to a shared client type:

```ts
type ApiError = {
  code: string;
  message: string;
  requestId?: string;
};
```

UI switches on `code` for i18n; do not parse English `message` strings.

---

## Generics & utilities

- Use TanStack Query generics for `TData` / `TError`.
- Prefer `Readonly` / `as const` for key factories and frame type maps.
- Exhaustive `switch` on frame `type` with a `never` default to catch missing handlers.

---

## Anti-patterns

- `JSON.parse(...) as T` everywhere.
- Loosening `tsconfig` to silence protocol drift.
- Exporting wide `interface Everything` from a god types file.

---

## Verification

```bash
cd frontend && npm run typecheck
```

Fail CI on `any` growth if eslint `@typescript-eslint/no-explicit-any` is enabled.
