package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"codeatlas.dev/internal/content"
)

func TestRoutesRender(t *testing.T) {
	t.Parallel()

	store, err := content.Load("../../content")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	app, err := New(store, Config{
		TemplatesDir: "../../templates",
		StaticDir:    "../../static",
		SiteURL:      "https://codeatlas.dev",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	handler := app.Routes()
	tests := []struct {
		name     string
		path     string
		contains string
	}{
		{name: "home", path: "/", contains: "Build production instincts"},
		{name: "topics", path: "/topics", contains: "Map the stack by topic"},
		{name: "topic", path: "/topic/go", contains: "Chi Routing Basics"},
		{name: "guide", path: "/guides/binary-search", contains: "Binary Search"},
		{name: "roadmaps", path: "/roadmaps", contains: "Study paths"},
		{name: "search", path: "/search?q=cache", contains: "Cache Invalidation"},
		{name: "sitemap", path: "/sitemap.xml", contains: "/guides/cache-invalidation"},
		{name: "robots", path: "/robots.txt", contains: "Sitemap: https://codeatlas.dev/sitemap.xml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("%s returned status %d, want 200", tt.path, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tt.contains) {
				t.Fatalf("%s did not contain %q", tt.path, tt.contains)
			}
		})
	}
}

func TestGuideSEO(t *testing.T) {
	t.Parallel()

	handler := testServer(t).Routes()
	req := httptest.NewRequest(http.MethodGet, "/guides/design-a-url-shortener", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("guide returned status %d, want 200", rec.Code)
	}

	body := rec.Body.String()
	checks := []string{
		"<title>Design a URL Shortener | codeatlas.dev</title>",
		`<meta name="description" content="Design a tiny URL service with short code generation, redirects, analytics, and hot-key protection.">`,
		`<link rel="canonical" href="https://codeatlas.dev/guides/design-a-url-shortener">`,
		`<meta property="og:type" content="article">`,
		`<meta property="og:url" content="https://codeatlas.dev/guides/design-a-url-shortener">`,
		`<meta name="twitter:card" content="summary">`,
		`<script type="application/ld+json">`,
		`"@type":"Article"`,
	}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("guide SEO output missing %q", check)
		}
	}
}

func TestSitemapUsesCanonicalURLs(t *testing.T) {
	t.Parallel()

	handler := testServer(t).Routes()
	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("sitemap returned status %d, want 200", rec.Code)
	}

	body := rec.Body.String()
	checks := []string{
		"<loc>https://codeatlas.dev/</loc>",
		"<loc>https://codeatlas.dev/topics</loc>",
		"<loc>https://codeatlas.dev/roadmaps</loc>",
		"<loc>https://codeatlas.dev/guides/design-a-rate-limiter</loc>",
	}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("sitemap missing %q", check)
		}
	}
	if strings.Contains(body, "<loc>https://codeatlas.dev/search</loc>") {
		t.Fatal("sitemap should not include search results page")
	}
}

func testServer(t *testing.T) *Server {
	t.Helper()

	store, err := content.Load("../../content")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	app, err := New(store, Config{
		TemplatesDir: "../../templates",
		StaticDir:    "../../static",
		SiteURL:      "https://codeatlas.dev",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return app
}
