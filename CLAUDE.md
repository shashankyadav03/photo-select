# photo-select

<One line: what this product does and for whom.>

## Stack
- <language/framework>, <db>, <hosting>
- LLM: <provider/model IDs live in `src/config/models.ts` (or equivalent)>

## Commands
- Dev: `<cmd>`
- Test (fast subset): `<cmd>`   Full: `<cmd>`
- Lint/typecheck: `<cmd>`
- Evals: `<cmd>`

## Structure
- `<dir>`: <what lives here>
- `<dir>`: <what lives here>

## Conventions (only what Claude can't infer)
- <e.g. all DB access via repository layer>
- <e.g. never call the LLM outside `src/llm/client.ts`>

## Gotchas
- <non-obvious thing that has bitten us>

## Out of bounds
- Don't edit: `<generated/ or vendor/ paths>`
- Don't touch prod config or migrations without asking.

Keep this file under 200 lines. Move detail into `.claude/rules/` or `docs/`.
