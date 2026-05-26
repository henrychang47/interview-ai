# AGENTS.md

## Project Overview

This repository contains an MVP for an interview practice application.

The main MVP specification is located at:

- `docs/mvp-spec.md`

Always read `docs/mvp-spec.md` before implementing features.

---

## Development Principles

Follow these rules strictly:

1. Implement one verifiable step at a time.
2. Do not implement future steps early.
3. Prioritize a working MVP flow over architecture optimization.
4. Keep changes small and reviewable.
5. After each API implementation, provide a curl command or test method.
6. After each frontend page implementation, provide manual verification steps.
7. Update `.env.example` when adding environment variables.
8. Never commit real secrets or API keys.
9. Use mock mode when external AI API keys are not configured.
10. Do not add non-MVP features unless explicitly requested.

## Documentation Update Rules

The agent may update these files during normal development:

- `README.md`
- `docs/DEVELOPMENT_PLAN.md`
- `docs/API.md`
- `.env.example`

The agent must update `docs/DEVELOPMENT_PLAN.md` after completing each step.

The agent must update `docs/API.md` after implementing or changing an API endpoint.

The agent must update `.env.example` whenever a new environment variable is introduced.

The agent must update `README.md` whenever setup, startup, migration, or verification commands change.

---

## Spec Change Rules

`docs/mvp-spec.md` is the source of truth for MVP scope and system design.

Do not modify `docs/mvp-spec.md` directly unless explicitly instructed.

If implementation reveals that the spec should change, first provide a spec change proposal containing:

1. Current spec behavior.
2. Problem discovered during implementation.
3. Proposed change.
4. Reason for the change.
5. Impact on database, API, frontend, or development plan.
6. Whether this changes MVP scope.

Only modify `docs/mvp-spec.md` after the change is approved.