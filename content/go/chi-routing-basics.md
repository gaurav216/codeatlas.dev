---
title: Chi Routing Basics
slug: chi-routing-basics
category: Go
difficulty: beginner
reading_time: 6 min read
tags:
  - go
  - chi
  - http
  - routing
description: Learn how to structure small, composable HTTP routes with the chi router.
---

## Why chi works well for monoliths

Chi gives you a tiny routing layer around the standard `net/http` package. That means handlers, middleware, graceful shutdown, and tests can stay close to Go's standard library while still giving you expressive route groups.

## Minimal router

```go
r := chi.NewRouter()
r.Use(middleware.RequestID)
r.Use(middleware.Recoverer)

r.Get("/", homeHandler)
r.Get("/guides/{slug}", guideHandler)
```

Route parameters are read with `chi.URLParam(r, "slug")`, and route groups let you apply middleware to one branch of the application without changing the rest.

## Production notes

- Keep handlers small and push business logic into internal packages.
- Set explicit server timeouts.
- Terminate long requests during deploys with graceful shutdown.
- Prefer boring middleware until traffic proves a sharper need.
