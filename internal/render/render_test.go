package render

import (
	"strings"
	"testing"
	"time"

	"github.com/mparvin/awesome-stars/internal/github"
)

func TestRenderREADME_structure(t *testing.T) {
	lists := []github.List{
		{
			Name: "AI-ML-DL",
			Repos: []github.Repository{
				{
					NameWithOwner:  "org/low-stars",
					URL:            "https://github.com/org/low-stars",
					Description:    "Less popular project",
					StargazerCount: 100,
				},
				{
					NameWithOwner:  "org/high-stars",
					URL:            "https://github.com/org/high-stars",
					Description:    "Popular project",
					StargazerCount: 5000,
				},
			},
		},
		{
			Name: "DevTools",
			Repos: []github.Repository{
				{
					NameWithOwner:  "owner/tool",
					URL:            "https://github.com/owner/tool",
					Description:    "A handy tool",
					StargazerCount: 250,
				},
			},
		},
	}

	cfg := Config{
		Categories: map[string]CategoryOverride{
			"AI-ML-DL": {
				Title: "AI, ML & Deep Learning",
				Emoji: "🤖",
				Order: 1,
			},
			"DevTools": {
				Title: "Developer Tools",
				Emoji: "🛠️",
				Order: 2,
			},
		},
	}

	updatedAt := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	out := RenderREADME(lists, cfg, updatedAt)

	if !strings.HasPrefix(out, "# Awesome Stars\n") {
		t.Error("expected default title heading")
	}
	if !strings.Contains(out, "Do not edit manually") {
		t.Error("expected auto-generated notice")
	}
	if !strings.Contains(out, "Last updated: 2026-07-10T12:00:00Z") {
		t.Error("expected last-updated timestamp")
	}

	if !strings.Contains(out, "## Table of Contents") {
		t.Error("expected table of contents section")
	}
	if !strings.Contains(out, "- [🤖 AI, ML & Deep Learning](#") {
		t.Error("expected TOC entry for AI category")
	}
	if !strings.Contains(out, "- [🛠️ Developer Tools](#") {
		t.Error("expected TOC entry for DevTools category")
	}

	aiSection := sectionContent(out, "## 🤖 AI, ML & Deep Learning")
	if aiSection == "" {
		t.Fatal("expected AI section")
	}

	highIdx := strings.Index(aiSection, "org/high-stars")
	lowIdx := strings.Index(aiSection, "org/low-stars")
	if highIdx == -1 || lowIdx == -1 {
		t.Fatal("expected both repos in AI section")
	}
	if highIdx > lowIdx {
		t.Error("expected repos sorted by stargazer count descending")
	}

	if !strings.Contains(aiSection, "[org/high-stars](https://github.com/org/high-stars) ⭐ 5000 — Popular project") {
		t.Error("expected formatted high-stars bullet")
	}

	devSection := sectionContent(out, "## 🛠️ Developer Tools")
	if !strings.Contains(devSection, "[owner/tool](https://github.com/owner/tool) ⭐ 250 — A handy tool") {
		t.Error("expected formatted devtools bullet")
	}
}

func TestRenderREADME_whitelistFilter(t *testing.T) {
	lists := []github.List{
		{
			Name: "AI-ML-DL",
			Repos: []github.Repository{
				{NameWithOwner: "org/ai", URL: "https://github.com/org/ai", StargazerCount: 100},
			},
		},
		{
			Name: "DevTools",
			Repos: []github.Repository{
				{NameWithOwner: "owner/tool", URL: "https://github.com/owner/tool", StargazerCount: 250},
			},
		},
		{
			Name: "Golang",
			Repos: []github.Repository{
				{NameWithOwner: "owner/go", URL: "https://github.com/owner/go", StargazerCount: 50},
			},
		},
	}

	cfg := Config{
		Categories: map[string]CategoryOverride{
			"AI-ML-DL": {
				Title: "AI, ML & Deep Learning",
				Emoji: "🤖",
				Order: 1,
			},
		},
	}

	out := RenderREADME(lists, cfg, time.Now())

	if !strings.Contains(out, "## 🤖 AI, ML & Deep Learning") {
		t.Error("expected whitelisted list in output")
	}
	if strings.Contains(out, "DevTools") {
		t.Error("unlisted DevTools list should not appear in output")
	}
	if strings.Contains(out, "Golang") {
		t.Error("unlisted Golang list should not appear in output")
	}
	if strings.Contains(out, "owner/tool") {
		t.Error("repos from unlisted lists should not appear in output")
	}
}

func TestRenderREADME_customTitle(t *testing.T) {
	lists := []github.List{
		{
			Name: "AI-ML-DL",
			Repos: []github.Repository{
				{NameWithOwner: "org/ai", URL: "https://github.com/org/ai", StargazerCount: 1},
			},
		},
	}

	cfg := Config{
		Title: "AI Awesome",
		Categories: map[string]CategoryOverride{
			"AI-ML-DL": {Title: "AI", Order: 1},
		},
	}

	out := RenderREADME(lists, cfg, time.Now())
	if !strings.HasPrefix(out, "# AI Awesome\n") {
		t.Error("expected custom project title in heading")
	}
}

func TestRenderREADME_fallbackCategoryName(t *testing.T) {
	lists := []github.List{
		{
			Name: "Uncategorized",
			Repos: []github.Repository{
				{
					NameWithOwner:  "a/b",
					URL:            "https://github.com/a/b",
					StargazerCount: 1,
				},
			},
		},
	}

	out := RenderREADME(lists, Config{Categories: map[string]CategoryOverride{}}, time.Now())
	if !strings.Contains(out, "## Uncategorized") {
		t.Error("expected raw list name as section title")
	}
	if !strings.Contains(out, "[a/b](https://github.com/a/b) ⭐ 1\n") {
		t.Error("expected repo bullet without description")
	}
}

func TestRenderREADME_emptyList(t *testing.T) {
	lists := []github.List{{Name: "Empty", Repos: nil}}
	out := RenderREADME(lists, Config{Categories: map[string]CategoryOverride{}}, time.Now())
	if !strings.Contains(out, "_No repositories in this list yet._") {
		t.Error("expected placeholder for empty list")
	}
}

func TestFilterLists_emptyConfigIncludesAll(t *testing.T) {
	lists := []github.List{
		{Name: "A"},
		{Name: "B"},
	}

	filtered := FilterLists(lists, Config{Categories: map[string]CategoryOverride{}})
	if len(filtered) != 2 {
		t.Errorf("expected all lists, got %d", len(filtered))
	}
}

func TestResolveCategory_overrideAndFallback(t *testing.T) {
	cfg := Config{
		Categories: map[string]CategoryOverride{
			"My-List": {Title: "Custom", Emoji: "✨", Order: 3},
		},
	}

	overridden := ResolveCategory("My-List", cfg)
	if overridden.Title != "✨ Custom" {
		t.Errorf("got title %q", overridden.Title)
	}
	if overridden.Order != 3 {
		t.Errorf("got order %d", overridden.Order)
	}

	fallback := ResolveCategory("Other", cfg)
	if fallback.Title != "Other" {
		t.Errorf("got title %q", fallback.Title)
	}
	if fallback.Order != 9999 {
		t.Errorf("got order %d", fallback.Order)
	}
}

func sectionContent(markdown, heading string) string {
	start := strings.Index(markdown, heading)
	if start == -1 {
		return ""
	}
	start += len(heading)
	rest := markdown[start:]
	next := strings.Index(rest, "\n## ")
	if next == -1 {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:next])
}
