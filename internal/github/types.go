package github

// Repository holds metadata for a starred repository.
type Repository struct {
	NameWithOwner   string
	Description     string
	URL             string
	StargazerCount  int
	PrimaryLanguage string
}

// List represents a GitHub Star List with its repositories.
type List struct {
	ID    string
	Name  string
	Slug  string
	Repos []Repository
}
