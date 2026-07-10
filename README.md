# awesome-stars

Turn your [GitHub Star Lists](https://github.com/stars) into an auto-updating Awesome-style README.

Each Star List becomes a category in `README.md`. A scheduled GitHub Actions workflow keeps it fresh by re-fetching your lists and committing changes.

## How it works

1. Authenticate with the GitHub GraphQL API as the Star List owner.
2. Fetch all `viewer.lists` (paginated) and each list's repositories (paginated).
3. Render an Awesome-style `README.md` with a table of contents and per-category repo bullets.
4. Optionally customize the README title and choose which Star Lists to include via `config.yaml`.

## Prerequisites

- Go 1.22+
- A GitHub account with Star Lists
- A Personal Access Token (PAT) belonging to the list owner

## Create a Personal Access Token

GitHub Star Lists are only available through the GraphQL API and require the list owner's identity. The default `GITHUB_TOKEN` in Actions cannot access `viewer.lists`.

1. Open [GitHub Settings → Developer settings → Personal access tokens](https://github.com/settings/tokens).
2. Create a **classic** token (fine-grained tokens may work, but classic is well-tested here).
3. Grant these scopes:
   - `read:user` — required to read your Star Lists
   - `repo` — required for public repository metadata (description, stars, language)
4. Copy the token and store it securely.

## GitHub Actions setup

1. Fork or create a repository for this project.
2. Add a repository secret named `STARS_READ_TOKEN` with your PAT value.
   - Repository → **Settings** → **Secrets and variables** → **Actions** → **New repository secret**
3. Enable GitHub Actions if needed.
4. The workflow in `.github/workflows/update.yml` runs:
   - on a schedule: `0 2 */2 * *` (02:00 UTC every 2 days)
   - on manual `workflow_dispatch`
5. When `README.md` changes, the workflow commits and pushes using the `github-actions[bot]` identity.

## Run locally

Create a `.env` file from the example and put your PAT in it:

```bash
cp .env.example .env
# edit .env and set GH_TOKEN=ghp_...
go run ./cmd/generate
```

You can still set `GH_TOKEN` in the environment instead (or to override `.env`). Existing environment variables take precedence over values in `.env`.

Optional flags:

```bash
go run ./cmd/generate -output README.md -config config.yaml -env .env
```

## Configuration

Edit `config.yaml` to set the README title and choose which Star Lists to include.

```yaml
title: "My Awesome Stars"

# Only lists listed here are included in the generated README.
# Leave this map empty to include all Star Lists.
categories:
  AI-ML-DL:
    title: "AI, ML & Deep Learning"
    emoji: "🤖"
    order: 1
  DevOps:
    title: "DevOps & Infrastructure"
    emoji: "⚙️"
    order: 2
```

- **`title`** — sets the README's main heading (defaults to `Awesome Stars` if omitted).
- **`categories`** — whitelist of GitHub Star Lists to include. Keys must match the exact GitHub list name. When one or more categories are defined, only those lists appear in the README. When empty or missing, all Star Lists are included.
- **`title` / `emoji` / `order`** (per category) — customize section display and sort order.

## Project layout

```
cmd/generate/main.go       Entrypoint
internal/github/           GraphQL client and pagination
internal/render/           README rendering and config.yaml parsing
config.yaml                Project title and Star List whitelist/overrides
.github/workflows/update.yml Scheduled updater
```

## Development

```bash
go test ./...
go build -o bin/generate ./cmd/generate
```

## License

MIT
