# AGENTS.md

## Project Overview

This repository contains an MVP for an interview practice application.

The user can:
1. Enter job information.
2. Generate interview questions.
3. Start a mock interview session.
4. Listen to each question using browser TTS.
5. Record answers using browser MediaRecorder.
6. Upload answer audio files.
7. Review interview results.

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