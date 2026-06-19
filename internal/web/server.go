package web

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"codeatlas.dev/internal/content"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Config struct {
	TemplatesDir string
	StaticDir    string
	SiteURL      string
}

type Server struct {
	store     *content.Store
	templates map[string]*template.Template
	config    Config
}

type PageData struct {
	SiteName      string
	SiteURL       string
	Title         string
	Description   string
	CanonicalURL  string
	CanonicalPath string
	CurrentPath   string
	OGType        string
	JSONLD        template.JS
	Categories    []content.Category
	Category      content.Category
	Guides        []content.Guide
	Guide         content.Guide
	Query         string
	Results       []content.SearchResult
}

func New(store *content.Store, config Config) (*Server, error) {
	if config.TemplatesDir == "" {
		config.TemplatesDir = "templates"
	}
	if config.StaticDir == "" {
		config.StaticDir = "static"
	}
	if config.SiteURL == "" {
		config.SiteURL = "https://codeatlas.dev"
	}
	config.SiteURL = strings.TrimRight(config.SiteURL, "/")

	templates, err := parseTemplates(config.TemplatesDir)
	if err != nil {
		return nil, err
	}

	return &Server{
		store:     store,
		templates: templates,
		config:    config,
	}, nil
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(middleware.Timeout(30 * time.Second))

	r.Handle("/static/*", s.staticHandler())

	r.Get("/", s.home)
	r.Get("/topics", s.topics)
	r.Get("/topic/{categorySlug}", s.topic)
	r.Get("/guides/{slug}", s.guide)
	r.Get("/roadmaps", s.roadmaps)
	r.Get("/search", s.search)
	r.Get("/sitemap.xml", s.sitemap)
	r.Get("/robots.txt", s.robots)
	r.NotFound(s.notFound)

	return r
}

func (s *Server) staticHandler() http.Handler {
	fileServer := http.StripPrefix("/static/", http.FileServer(http.Dir(s.config.StaticDir)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		fileServer.ServeHTTP(w, r)
	})
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	data := s.pageData(r, "Practical Computer Science Study Guides", "Focused guides, roadmaps, and examples for learning software engineering fundamentals.")
	data.Guides = s.store.Latest(6)
	s.render(w, r, http.StatusOK, "home", data)
}

func (s *Server) topics(w http.ResponseWriter, r *http.Request) {
	data := s.pageData(r, "Topics", "Browse codeatlas.dev study guides by topic.")
	s.render(w, r, http.StatusOK, "topics", data)
}

func (s *Server) topic(w http.ResponseWriter, r *http.Request) {
	categorySlug := chi.URLParam(r, "categorySlug")
	category, ok := s.store.Category(categorySlug)
	if !ok {
		s.notFound(w, r)
		return
	}

	data := s.pageData(r, category.Title, "Study guides and examples for "+category.Title+".")
	data.Category = category
	data.Guides = s.store.ByCategory(categorySlug)
	s.render(w, r, http.StatusOK, "topic", data)
}

func (s *Server) guide(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	guide, ok := s.store.GetBySlug(slug)
	if !ok {
		s.notFound(w, r)
		return
	}

	description := guide.Description
	if description == "" {
		description = "Read " + guide.Title + " on codeatlas.dev."
	}
	data := s.pageData(r, guide.Title, description)
	data.Guide = guide
	data.OGType = "article"
	data.JSONLD = articleJSONLD(s.absoluteURL(r.URL.Path), guide)
	s.render(w, r, http.StatusOK, "guide", data)
}

func (s *Server) roadmaps(w http.ResponseWriter, r *http.Request) {
	data := s.pageData(r, "Roadmaps", "Structured study paths across the codeatlas.dev guide library.")
	data.Guides = s.store.AllGuides()
	s.render(w, r, http.StatusOK, "roadmaps", data)
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	data := s.pageData(r, "Search", "Search codeatlas.dev guides by title, tags, description, and body text.")
	data.Query = query
	if query != "" {
		data.Results = s.store.Search(query)
	}
	s.render(w, r, http.StatusOK, "search", data)
}

func (s *Server) sitemap(w http.ResponseWriter, r *http.Request) {
	type sitemapURL struct {
		Loc     string `xml:"loc"`
		LastMod string `xml:"lastmod,omitempty"`
	}
	type urlSet struct {
		XMLName xml.Name     `xml:"urlset"`
		Xmlns   string       `xml:"xmlns,attr"`
		URLs    []sitemapURL `xml:"url"`
	}

	paths := []string{"/", "/topics", "/roadmaps"}
	urls := make([]sitemapURL, 0, len(paths)+len(s.store.Categories())+len(s.store.AllGuides()))
	for _, path := range paths {
		urls = append(urls, sitemapURL{Loc: s.absoluteURL(path)})
	}
	for _, category := range s.store.Categories() {
		urls = append(urls, sitemapURL{Loc: s.absoluteURL("/topic/" + category.Slug)})
	}
	for _, guide := range s.store.AllGuides() {
		urls = append(urls, sitemapURL{
			Loc:     s.absoluteURL("/guides/" + guide.Slug),
			LastMod: guide.UpdatedAt,
		})
	}

	output, err := xml.MarshalIndent(urlSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}, "", "  ")
	if err != nil {
		http.Error(w, "failed to build sitemap", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(output)
}

func (s *Server) robots(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintf(w, "User-agent: *\nAllow: /\n\nSitemap: %s\n", s.absoluteURL("/sitemap.xml"))
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	data := s.pageData(r, "Page not found", "The requested codeatlas.dev page could not be found.")
	s.render(w, r, http.StatusNotFound, "notfound", data)
}

func (s *Server) pageData(r *http.Request, title, description string) PageData {
	return PageData{
		SiteName:      "codeatlas.dev",
		SiteURL:       s.config.SiteURL,
		Title:         title,
		Description:   description,
		CanonicalURL:  s.absoluteURL(r.URL.Path),
		CanonicalPath: r.URL.Path,
		CurrentPath:   r.URL.Path,
		OGType:        "website",
		Categories:    s.store.Categories(),
	}
}

func (s *Server) render(w http.ResponseWriter, _ *http.Request, status int, name string, data PageData) {
	tmpl, ok := s.templates[name]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "base", data); err != nil {
		http.Error(w, "failed to render template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

func (s *Server) absoluteURL(path string) string {
	if path == "" || path[0] != '/' {
		path = "/" + path
	}
	return s.config.SiteURL + path
}

func articleJSONLD(canonicalURL string, guide content.Guide) template.JS {
	data := map[string]any{
		"@context":         "https://schema.org",
		"@type":            "Article",
		"headline":         guide.Title,
		"description":      guide.Description,
		"dateModified":     guide.UpdatedAt,
		"mainEntityOfPage": canonicalURL,
		"author": map[string]string{
			"@type": "Organization",
			"name":  "codeatlas.dev",
		},
		"publisher": map[string]string{
			"@type": "Organization",
			"name":  "codeatlas.dev",
		},
	}

	output, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return template.JS(output)
}

func parseTemplates(dir string) (map[string]*template.Template, error) {
	pages := []string{"home", "guide", "topic", "topics", "search", "roadmaps", "notfound"}
	layout := filepath.Join(dir, "layout.html")

	funcs := template.FuncMap{
		"safeHTML": func(value string) template.HTML {
			return template.HTML(value)
		},
		"join": strings.Join,
		"isActive": func(currentPath, path string) bool {
			if path == "/" {
				return currentPath == "/"
			}
			return currentPath == path || strings.HasPrefix(currentPath, path+"/")
		},
	}

	cache := make(map[string]*template.Template, len(pages))
	for _, name := range pages {
		page := filepath.Join(dir, name+".html")
		files := []string{layout, page}

		tmpl, err := template.New(filepath.Base(layout)).Funcs(funcs).ParseFiles(files...)
		if err != nil {
			return nil, fmt.Errorf("parse templates for %s: %w", filepath.Base(page), err)
		}

		cache[name] = tmpl
	}

	return cache, nil
}
