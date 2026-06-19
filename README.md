# codeatlas.dev

A production-ready MVP Go monolith for a public study website. It serves HTML templates, static assets, recursive markdown content, topic pages, guide pages, search, roadmaps, and a sitemap.

## Stack

- Go 1.22+
- chi router
- goldmark markdown rendering
- Chroma-powered code highlighting through `goldmark-highlighting`
- bluemonday HTML sanitization
- YAML frontmatter with `gopkg.in/yaml.v3`
- Go `html/template` views with Tailwind CDN for the MVP UI
- Mermaid.js CDN for markdown diagram rendering hooks

## Project Layout

```text
cmd/server/main.go      HTTP server entrypoint and graceful shutdown
internal/content        Markdown loading, frontmatter parsing, safe HTML, search index
internal/web            chi routes, template rendering, sitemap, static file serving
templates               Go HTML templates: layout, pages, and shared template definitions
static                  CSS, JavaScript, images, article TOC, Mermaid, and copy-code helpers
content                 Markdown guide library
```

## Content Format

Add guides anywhere under `content/`; files are loaded recursively at startup. Use `.md` or `.markdown`.

```markdown
---
title: Binary Search
slug: binary-search
category: Algorithms
difficulty: beginner
reading_time: 7 min read
tags: [algorithms, arrays, search]
description: Build the invariant behind binary search before memorizing the loop.
---

## Guide body

Markdown content goes here.
```

Required field: `title`.

If `slug` is omitted, it is generated from the title. If `category` is omitted, the guide is placed under `Uncategorized`. If `reading_time` is omitted, it is estimated from the body text.

## Local Development

```sh
go mod tidy
go test ./...
PORT=8080 go run ./cmd/server
```

Open `http://localhost:8080`.

## Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | HTTP port to bind. |
| `BASE_URL` | `https://codeatlas.dev` | Absolute base URL for canonical links, social metadata, `robots.txt`, and `sitemap.xml`. |
| `SITE_URL` | unset | Backward-compatible fallback when `BASE_URL` is not set. |
| `CONTENT_DIR` | `content` | Markdown content directory. |
| `TEMPLATES_DIR` | `templates` | HTML template directory. |
| `STATIC_DIR` | `static` | Static asset directory. |

## Routes

- `GET /`
- `GET /topics`
- `GET /topic/{categorySlug}`
- `GET /guides/{slug}`
- `GET /roadmaps`
- `GET /search?q=`
- `GET /sitemap.xml`
- `GET /robots.txt`

## Deployment

The server reads `PORT` from the environment and serves `content/`, `templates/`, and `static/` relative to the working directory. In production, run from the project root or use the included Docker image, which copies those directories into `/app`.

### Binary

```sh
go build -tags netgo -ldflags '-s -w' -o app ./cmd/server
BASE_URL=https://codeatlas.dev PORT=8080 ./app
```

### Docker

Build and run locally:

```sh
docker build -t codeatlas.dev .
docker run --rm -p 8080:8080 \
  -e PORT=8080 \
  -e BASE_URL=http://localhost:8080 \
  codeatlas.dev
```

The Dockerfile uses a multi-stage Go build and compiles with:

```sh
go build -tags netgo -ldflags '-s -w' -o app ./cmd/server
```

The runtime image starts with:

```sh
./app
```

### Render Web Service

This repository includes `render.yaml` for Render Blueprint deploys.

1. Push `codeatlas.dev/` to a Git repository.
2. In Render, create a new Blueprint from the repository, or create a Web Service using Docker.
3. Use the included `Dockerfile`.
4. Set `BASE_URL` to the public site URL, for example `https://codeatlas.dev`.
5. Let Render provide `PORT`; the app binds to that environment variable automatically.
6. Health checks can use `/`.

For manual Render setup:

| Setting | Value |
| --- | --- |
| Runtime | Docker |
| Dockerfile path | `./Dockerfile` |
| Docker context | `.` |
| Start command | from Docker image, `./app` |
| Health check path | `/` |
| Required env var | `BASE_URL=https://codeatlas.dev` |
