package render

import (
	"os"
	"sort"

	"github.com/mparvin/awesome-stars/internal/github"
	"gopkg.in/yaml.v3"
)

const defaultTitle = "Awesome Stars"

// CategoryOverride customizes how a GitHub list appears in the README.
type CategoryOverride struct {
	Title string `yaml:"title"`
	Emoji string `yaml:"emoji"`
	Order int    `yaml:"order"`
}

// Config holds project settings and Star List inclusion/display overrides.
type Config struct {
	Title      string                      `yaml:"title"`
	Categories map[string]CategoryOverride `yaml:"categories"`
}

// LoadConfig reads project config from a YAML file.
// Returns an empty config if the file does not exist.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{Categories: map[string]CategoryOverride{}}, nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Categories == nil {
		cfg.Categories = map[string]CategoryOverride{}
	}
	return cfg, nil
}

// ProjectTitle returns the configured README title or the default.
func (cfg Config) ProjectTitle() string {
	if cfg.Title != "" {
		return cfg.Title
	}
	return defaultTitle
}

// FilterLists returns only lists whitelisted in config when categories are defined.
// An empty categories map includes all lists.
func FilterLists(lists []github.List, cfg Config) []github.List {
	if len(cfg.Categories) == 0 {
		return lists
	}

	filtered := make([]github.List, 0, len(cfg.Categories))
	for _, list := range lists {
		if _, ok := cfg.Categories[list.Name]; ok {
			filtered = append(filtered, list)
		}
	}
	return filtered
}

// CategoryMeta holds resolved display metadata for a list.
type CategoryMeta struct {
	ListName string
	Title    string
	Emoji    string
	Order    int
	Anchor   string
}

// ResolveCategory returns display metadata for a list, applying overrides when present.
func ResolveCategory(listName string, cfg Config) CategoryMeta {
	override, ok := cfg.Categories[listName]
	title := listName
	emoji := ""
	order := 9999

	if ok {
		if override.Title != "" {
			title = override.Title
		}
		emoji = override.Emoji
		if override.Order > 0 {
			order = override.Order
		}
	}

	displayTitle := title
	if emoji != "" {
		displayTitle = emoji + " " + title
	}

	return CategoryMeta{
		ListName: listName,
		Title:    displayTitle,
		Emoji:    emoji,
		Order:    order,
		Anchor:   anchorID(displayTitle),
	}
}

// SortCategories orders category metadata by override order, then name.
func SortCategories(categories []CategoryMeta) {
	sort.Slice(categories, func(i, j int) bool {
		if categories[i].Order != categories[j].Order {
			return categories[i].Order < categories[j].Order
		}
		return categories[i].ListName < categories[j].ListName
	})
}
