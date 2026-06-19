package content

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBuildsGuidesCategoriesAndSearch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGuide(t, dir, "go/chi.md", `---
title: Chi Routing Basics
slug: chi-routing
category: Go
difficulty: beginner
reading_time: 4 min read
tags: [go, chi, routing]
description: Learn how to compose HTTP routes with chi.
---

# Chi Routing Basics

Use `+"`chi.Router`"+` to compose small handlers.
`)
	writeGuide(t, dir, "systems/cache.md", `---
title: Cache Primer
slug: cache-primer
category: Systems
difficulty: beginner
tags: [cache, reliability]
description: Keep data close to users.
---

# Cache Primer

Cache layers reduce read latency.
`)
	writeGuide(t, dir, "algorithms/binary.md", `---
title: Binary Search
slug: binary-search
category: Algorithms
tags: [arrays]
description: Find values in sorted arrays.
---

# Binary Search

Search by shrinking the candidate range.
`)

	store, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if store.Count() != 3 {
		t.Fatalf("Count() = %d, want 3", store.Count())
	}

	guides := store.AllGuides()
	if got := []string{guides[0].Title, guides[1].Title, guides[2].Title}; got[0] != "Binary Search" || got[1] != "Cache Primer" || got[2] != "Chi Routing Basics" {
		t.Fatalf("AllGuides() titles = %v, want title sorted", got)
	}

	guide, ok := store.GetBySlug("chi-routing")
	if !ok {
		t.Fatal("GetBySlug(chi-routing) not found")
	}
	if guide.CategorySlug != "go" {
		t.Fatalf("CategorySlug = %q, want go", guide.CategorySlug)
	}
	if guide.HTML == "" {
		t.Fatal("HTML was empty")
	}

	goGuides := store.ByCategory("go")
	if len(goGuides) != 1 || goGuides[0].Slug != "chi-routing" {
		t.Fatalf("ByCategory(go) = %+v, want chi-routing", goGuides)
	}

	results := store.Search("router")
	if len(results) != 1 {
		t.Fatalf("Search(router) returned %d results, want 1", len(results))
	}

	categories := store.Categories()
	if len(categories) != 3 {
		t.Fatalf("Categories() returned %d categories, want 3", len(categories))
	}
}

func TestSearchRanksTitleAboveCategoryTagAndBody(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGuide(t, dir, "title.md", `---
title: Cache Fundamentals
slug: cache-fundamentals
category: Systems
tags: [latency]
description: A guide about read paths.
---

Body text.
`)
	writeGuide(t, dir, "tag.md", `---
title: Read Latency
slug: read-latency
category: Systems
tags: [cache]
description: A guide about read paths.
---

Body text.
`)
	writeGuide(t, dir, "body.md", `---
title: Storage Notes
slug: storage-notes
category: Databases
tags: [storage]
description: A guide about read paths.
---

This body mentions cache once.
`)

	store, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	results := store.Search("cache")
	if len(results) != 3 {
		t.Fatalf("Search(cache) returned %d results, want 3", len(results))
	}
	if results[0].Guide.Slug != "cache-fundamentals" {
		t.Fatalf("top result = %q, want title match first", results[0].Guide.Slug)
	}
	if results[1].Guide.Slug != "read-latency" {
		t.Fatalf("second result = %q, want tag match before body match", results[1].Guide.Slug)
	}
	if results[2].Guide.Slug != "storage-notes" {
		t.Fatalf("third result = %q, want body match last", results[2].Guide.Slug)
	}
}

func TestLoadRejectsDuplicateSlugs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	body := `---
title: One
slug: duplicate
category: Go
---

Body.
`
	writeGuide(t, dir, "one.md", body)
	writeGuide(t, dir, "two.md", body)

	if _, err := Load(dir); err == nil {
		t.Fatal("Load() error = nil, want duplicate slug error")
	}
}

func writeGuide(t *testing.T, root, name, body string) {
	t.Helper()

	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
