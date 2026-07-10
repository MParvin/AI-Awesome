package github

import (
	"context"
	"errors"
	"fmt"
	"log"
)

const listsQuery = `
query Lists($after: String) {
  viewer {
    lists(first: 100, after: $after) {
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        id
        name
        slug
      }
    }
  }
}`

const listItemsQuery = `
query ListItems($listID: ID!, $after: String) {
  node(id: $listID) {
    ... on UserList {
      items(first: 100, after: $after) {
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          ... on Repository {
            nameWithOwner
            description
            url
            stargazerCount
            primaryLanguage {
              name
            }
          }
        }
      }
    }
  }
}`

type listsResponse struct {
	Viewer struct {
		Lists struct {
			PageInfo pageInfo   `json:"pageInfo"`
			Nodes    []listNode `json:"nodes"`
		} `json:"lists"`
	} `json:"viewer"`
}

type listNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type listItemsResponse struct {
	Node *struct {
		Items struct {
			PageInfo pageInfo       `json:"pageInfo"`
			Nodes    []itemNode     `json:"nodes"`
		} `json:"items"`
	} `json:"node"`
}

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type itemNode struct {
	NameWithOwner   string `json:"nameWithOwner"`
	Description     string `json:"description"`
	URL             string `json:"url"`
	StargazerCount  int    `json:"stargazerCount"`
	PrimaryLanguage *struct {
		Name string `json:"name"`
	} `json:"primaryLanguage"`
}

// FetchLists retrieves all Star Lists for the authenticated user.
func (c *Client) FetchLists(ctx context.Context) ([]List, error) {
	var all []List
	var after *string

	for {
		vars := map[string]any{}
		if after != nil {
			vars["after"] = *after
		}

		var data listsResponse
		if err := c.queryWithPartialWarning(ctx, listsQuery, vars, &data); err != nil {
			return nil, err
		}

		for _, node := range data.Viewer.Lists.Nodes {
			all = append(all, List{
				ID:   node.ID,
				Name: node.Name,
				Slug: node.Slug,
			})
		}

		if !data.Viewer.Lists.PageInfo.HasNextPage {
			break
		}
		cursor := data.Viewer.Lists.PageInfo.EndCursor
		after = &cursor
	}

	for i := range all {
		repos, err := c.fetchListItems(ctx, all[i].ID)
		if err != nil {
			return nil, fmt.Errorf("fetch items for list %q: %w", all[i].Name, err)
		}
		all[i].Repos = repos
	}

	return all, nil
}

func (c *Client) fetchListItems(ctx context.Context, listID string) ([]Repository, error) {
	var all []Repository
	var after *string

	for {
		vars := map[string]any{"listID": listID}
		if after != nil {
			vars["after"] = *after
		}

		var data listItemsResponse
		if err := c.queryWithPartialWarning(ctx, listItemsQuery, vars, &data); err != nil {
			return nil, err
		}

		if data.Node == nil {
			return nil, fmt.Errorf("list %s not found", listID)
		}

		for _, node := range data.Node.Items.Nodes {
			repo := Repository{
				NameWithOwner:  node.NameWithOwner,
				Description:    node.Description,
				URL:            node.URL,
				StargazerCount: node.StargazerCount,
			}
			if node.PrimaryLanguage != nil {
				repo.PrimaryLanguage = node.PrimaryLanguage.Name
			}
			if repo.NameWithOwner != "" {
				all = append(all, repo)
			}
		}

		if !data.Node.Items.PageInfo.HasNextPage {
			break
		}
		cursor := data.Node.Items.PageInfo.EndCursor
		after = &cursor
	}

	return all, nil
}

func (c *Client) queryWithPartialWarning(ctx context.Context, query string, variables map[string]any, dest any) error {
	err := c.Query(ctx, query, variables, dest)
	if err == nil {
		return nil
	}
	var partial *PartialError
	if errors.As(err, &partial) {
		log.Printf("warning: %v", partial)
		return nil
	}
	return err
}
