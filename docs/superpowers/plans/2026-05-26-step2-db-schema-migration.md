# Step 2 DB Schema Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create PostgreSQL migrations for the MVP `interviews`, `questions`, and `answers` tables and provide a Docker Compose command to run them.

**Architecture:** Keep Step 2 limited to database schema and migration execution. Use SQL migration files under `backend/migrations` and a one-shot Docker Compose `migrate` service so developers do not need to install a migration CLI locally. Do not connect the Go API to the database in this step.

**Tech Stack:** PostgreSQL 16, Docker Compose, `migrate/migrate:v4.17.1`, SQL migrations.

---

## File Structure

- Create `backend/migrations/000001_create_interview_tables.up.sql`: creates `pgcrypto`, `interviews`, `questions`, and `answers`.
- Create `backend/migrations/000001_create_interview_tables.down.sql`: drops the three MVP tables in dependency order.
- Modify `docker-compose.yml`: adds a one-shot `migrate` service that waits for PostgreSQL and runs migrations.
- Modify `README.md`: documents migration and schema verification commands.
- Modify `docs/development-progress.md`: marks Step 2 as completed after implementation verification.

## Task 1: Add Migration SQL Files

**Files:**
- Create: `backend/migrations/000001_create_interview_tables.up.sql`
- Create: `backend/migrations/000001_create_interview_tables.down.sql`

- [ ] **Step 1: Write the migration files**

Create `backend/migrations/000001_create_interview_tables.up.sql`:

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE interviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_title TEXT NOT NULL,
    job_description TEXT NOT NULL,
    user_profile TEXT NOT NULL,
    question_count INTEGER NOT NULL CHECK (question_count BETWEEN 1 AND 10),
    status TEXT NOT NULL DEFAULT 'created' CHECK (
        status IN ('created', 'questions_ready', 'in_progress', 'completed', 'failed')
    ),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interview_id UUID NOT NULL REFERENCES interviews(id) ON DELETE CASCADE,
    question_order INTEGER NOT NULL,
    question_text TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (interview_id, question_order)
);

CREATE TABLE answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interview_id UUID NOT NULL REFERENCES interviews(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    audio_path TEXT,
    transcript_text TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (interview_id, question_id)
);
```

Create `backend/migrations/000001_create_interview_tables.down.sql`:

```sql
DROP TABLE IF EXISTS answers;
DROP TABLE IF EXISTS questions;
DROP TABLE IF EXISTS interviews;
```

- [ ] **Step 2: Verify files are present**

Run:

```powershell
Get-ChildItem backend\migrations
```

Expected: both `000001_create_interview_tables.up.sql` and `000001_create_interview_tables.down.sql` are listed.

## Task 2: Add Docker Compose Migration Service

**Files:**
- Modify: `docker-compose.yml`

- [ ] **Step 1: Update `docker-compose.yml`**

Add this service between `postgres` and `backend`:

```yaml
  migrate:
    image: migrate/migrate:v4.17.1
    volumes:
      - ./backend/migrations:/migrations:ro
    command:
      [
        "-path",
        "/migrations",
        "-database",
        "${DATABASE_URL:-postgres://interview_ai:interview_ai_dev_password@postgres:5432/interview_ai?sslmode=disable}",
        "up"
      ]
    depends_on:
      postgres:
        condition: service_healthy
```

Do not make `backend` depend on `migrate` in Step 2. Migration execution remains an explicit command so this step is easy to verify and rerun intentionally.

- [ ] **Step 2: Validate Compose configuration**

Run:

```powershell
docker compose config
```

Expected: exit code 0 and the rendered config includes `migrate`.

## Task 3: Document Migration Usage

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Add migration commands**

Add a `Database Migrations` section after setup:

````md
## Database Migrations

Start PostgreSQL:

```powershell
docker compose up -d postgres
```

Run migrations:

```powershell
docker compose run --rm migrate
```

Verify tables:

```powershell
docker compose exec postgres psql -U interview_ai -d interview_ai -c "\dt"
```

Expected tables:

- `interviews`
- `questions`
- `answers`
````

- [ ] **Step 2: Keep Step 1 verification intact**

Ensure the existing `/health` and frontend verification instructions remain in `README.md`.

## Task 4: Verify Migration End-to-End

**Files:**
- No production code changes.

- [ ] **Step 1: Start PostgreSQL**

Run:

```powershell
docker compose up -d postgres
```

Expected: PostgreSQL container is running and healthy.

- [ ] **Step 2: Run migrations**

Run:

```powershell
docker compose run --rm migrate
```

Expected: command exits 0 and logs show migration version `1` applied.

- [ ] **Step 3: Verify tables exist**

Run:

```powershell
docker compose exec postgres psql -U interview_ai -d interview_ai -c "\dt"
```

Expected output includes:

```text
interviews
questions
answers
schema_migrations
```

- [ ] **Step 4: Verify schema details**

Run:

```powershell
docker compose exec postgres psql -U interview_ai -d interview_ai -c "\d interviews"
docker compose exec postgres psql -U interview_ai -d interview_ai -c "\d questions"
docker compose exec postgres psql -U interview_ai -d interview_ai -c "\d answers"
```

Expected:

- `interviews.id`, `questions.id`, and `answers.id` are UUID primary keys with `gen_random_uuid()`.
- `questions.interview_id` references `interviews(id)` with cascade delete.
- `answers.interview_id` references `interviews(id)` with cascade delete.
- `answers.question_id` references `questions(id)` with cascade delete.
- `questions` has unique constraint on `(interview_id, question_order)`.
- `answers` has unique constraint on `(interview_id, question_id)`.

## Task 5: Update Progress and Commit

**Files:**
- Modify: `docs/development-progress.md`

- [ ] **Step 1: Update progress document after verification**

Change Step 2 status from `Not started` to `Completed`, and update current status:

```md
- Current step: Step 2 - DB schema 與 migration
- Status: Completed
- Last updated: 2026-05-26
```

Add a Step 2 completed section:

```md
### Step 2 - DB schema 與 migration

Completed on 2026-05-26.

Implemented:

- Added SQL migrations for `interviews`, `questions`, and `answers`.
- Added Docker Compose migration service.
- Documented migration execution and verification commands.

Verification:

- `docker compose config` passed.
- `docker compose run --rm migrate` applied migration version 1.
- PostgreSQL listed `interviews`, `questions`, `answers`, and `schema_migrations`.
```

- [ ] **Step 2: Run final checks**

Run:

```powershell
go test ./...
npm test
npm run build
docker compose config
```

Expected: all commands exit 0.

- [ ] **Step 3: Commit Step 2**

Run:

```powershell
git add backend/migrations docker-compose.yml README.md docs/development-progress.md
git commit -m "feat: add database migrations"
```

Expected: commit succeeds.

## Assumptions

- Step 2 does not add Go database connection code or repository code.
- Step 2 does not create seed data.
- Step 2 keeps migration execution explicit through `docker compose run --rm migrate`.
- Existing local PostgreSQL data may already contain tables from a previous run. If verification needs a clean database, use `docker compose down -v` only after explicit confirmation because it deletes local database data.
