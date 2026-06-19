package content

import (
	"sort"
	"strings"
	"unicode"
)

type Guide struct {
	Title        string
	Slug         string
	Category     string
	CategorySlug string
	Difficulty   string
	ReadingTime  string
	Tags         []string
	Description  string
	HTML         string
	BodyText     string
	SourcePath   string
	UpdatedAt    string
}

type Category struct {
	Title string
	Slug  string
	Count int
}

type SearchResult struct {
	Guide Guide
	Score int
}

type Store struct {
	guides      []Guide
	bySlug      map[string]Guide
	categories  []Category
	byCategory  map[string][]Guide
	categoryMap map[string]Category
}

func newStore(guides []Guide) *Store {
	sort.Slice(guides, func(i, j int) bool {
		if guides[i].Title == guides[j].Title {
			return guides[i].Slug < guides[j].Slug
		}
		return guides[i].Title < guides[j].Title
	})

	store := &Store{
		guides:      guides,
		bySlug:      make(map[string]Guide, len(guides)),
		byCategory:  make(map[string][]Guide),
		categoryMap: make(map[string]Category),
	}

	for _, guide := range guides {
		store.bySlug[guide.Slug] = guide
		store.byCategory[guide.CategorySlug] = append(store.byCategory[guide.CategorySlug], guide)

		category := store.categoryMap[guide.CategorySlug]
		if category.Slug == "" {
			category = Category{Title: guide.Category, Slug: guide.CategorySlug}
		}
		category.Count++
		store.categoryMap[guide.CategorySlug] = category
	}

	for _, category := range store.categoryMap {
		store.categories = append(store.categories, category)
	}
	sort.Slice(store.categories, func(i, j int) bool {
		return store.categories[i].Title < store.categories[j].Title
	})

	return store
}

func (s *Store) Count() int {
	return len(s.guides)
}

func (s *Store) AllGuides() []Guide {
	return append([]Guide(nil), s.guides...)
}

func (s *Store) Latest(limit int) []Guide {
	guides := append([]Guide(nil), s.guides...)
	sort.Slice(guides, func(i, j int) bool {
		return guides[i].UpdatedAt > guides[j].UpdatedAt
	})
	if limit > 0 && len(guides) > limit {
		guides = guides[:limit]
	}
	return guides
}

func (s *Store) GetBySlug(slug string) (Guide, bool) {
	guide, ok := s.bySlug[slug]
	return guide, ok
}

func (s *Store) Categories() []Category {
	return append([]Category(nil), s.categories...)
}

func (s *Store) Category(slug string) (Category, bool) {
	category, ok := s.categoryMap[slug]
	return category, ok
}

func (s *Store) ByCategory(slug string) []Guide {
	return append([]Guide(nil), s.byCategory[slug]...)
}

func (s *Store) Search(query string) []SearchResult {
	terms := strings.Fields(normalize(query))
	if len(terms) == 0 {
		return nil
	}

	var results []SearchResult
	for _, guide := range s.guides {
		score := 0
		title := normalize(guide.Title)
		category := normalize(guide.Category)
		description := normalize(guide.Description)
		tags := normalize(strings.Join(guide.Tags, " "))
		body := normalize(guide.BodyText)

		for _, term := range terms {
			switch {
			case title == term:
				score += 20
			case strings.Contains(title, term):
				score += 12
			}
			if strings.Contains(tags, term) || strings.Contains(category, term) {
				score += 7
			}
			if strings.Contains(description, term) {
				score += 3
			}
			if strings.Contains(body, term) {
				score++
			}
		}

		if score > 0 {
			results = append(results, SearchResult{Guide: guide, Score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Guide.Title < results[j].Guide.Title
		}
		return results[i].Score > results[j].Score
	})
	return results
}

func normalize(input string) string {
	var b strings.Builder
	b.Grow(len(input))
	for _, r := range strings.ToLower(input) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		b.WriteByte(' ')
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
