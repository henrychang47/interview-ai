## Development Principles

Follow these rules strictly:

- Implement one verifiable step at a time.
- Keep changes small and reviewable.
- After each API implementation, provide a curl command or test method.
- After each frontend page implementation, provide manual verification steps.
- Update `.env.example` when adding environment variables.
- Never commit real secrets or API keys.
- Use mock mode when external AI API keys are not configured.


## Documentation Update Rules

The agent may update these files during normal development:

- `docs/API.md`
- `.env.example`

The agent must update `docs/API.md` after implementing or changing an API endpoint.

The agent must update `.env.example` whenever a new environment variable is introduced.