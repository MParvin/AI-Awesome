package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const graphqlURL = "https://api.github.com/graphql"

// Client queries the GitHub GraphQL API.
type Client struct {
	token      string
	httpClient *http.Client
}

// NewClient creates a GraphQL client authenticated with a PAT.
func NewClient(token string) *Client {
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors"`
}

type graphQLError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Query executes a GraphQL query with retry and rate-limit handling.
func (c *Client) Query(ctx context.Context, query string, variables map[string]any, dest any) error {
	const maxAttempts = 5
	backoff := 2 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := c.doRequest(ctx, query, variables)
		if err != nil {
			if attempt == maxAttempts {
				return err
			}
			if wait := retryAfter(err); wait > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(wait):
				}
				continue
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
			continue
		}

		if len(resp.Errors) > 0 && len(resp.Data) == 0 {
			if isRateLimited(resp.Errors) && attempt < maxAttempts {
				wait := backoff
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(wait):
				}
				backoff *= 2
				continue
			}
			return fmt.Errorf("graphql errors: %s", formatErrors(resp.Errors))
		}

		if len(resp.Errors) > 0 {
			// Partial data: log-style warning via returned error only when decode fails.
			_ = formatErrors(resp.Errors)
		}

		if dest != nil && len(resp.Data) > 0 {
			if err := json.Unmarshal(resp.Data, dest); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}
		}

		if len(resp.Errors) > 0 {
			// Return partial success with a wrapped warning for callers that care.
			return &PartialError{Errors: resp.Errors}
		}
		return nil
	}

	return fmt.Errorf("graphql query failed after %d attempts", maxAttempts)
}

// PartialError indicates GraphQL returned data alongside errors.
type PartialError struct {
	Errors []graphQLError
}

func (e *PartialError) Error() string {
	return fmt.Sprintf("partial graphql response: %s", formatErrors(e.Errors))
}

func (c *Client) doRequest(ctx context.Context, query string, variables map[string]any) (*graphQLResponse, error) {
	body, err := json.Marshal(graphQLRequest{Query: query, Variables: variables})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode == http.StatusForbidden || httpResp.StatusCode == http.StatusTooManyRequests {
		reset := httpResp.Header.Get("X-RateLimit-Reset")
		if reset != "" {
			if unix, parseErr := strconv.ParseInt(reset, 10, 64); parseErr == nil {
				wait := time.Until(time.Unix(unix, 0))
				if wait > 0 {
					return nil, &retryableError{wait: wait}
				}
			}
		}
		return nil, &retryableError{wait: 60 * time.Second}
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api status %d: %s", httpResp.StatusCode, string(respBody))
	}

	var resp graphQLResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &resp, nil
}

type retryableError struct {
	wait time.Duration
}

func (e *retryableError) Error() string {
	return fmt.Sprintf("rate limited, retry after %s", e.wait)
}

func retryAfter(err error) time.Duration {
	var re *retryableError
	if errors.As(err, &re) {
		return re.wait
	}
	return 0
}

func isRateLimited(errors []graphQLError) bool {
	for _, e := range errors {
		if e.Type == "RATE_LIMITED" {
			return true
		}
	}
	return false
}

func formatErrors(errors []graphQLError) string {
	if len(errors) == 0 {
		return ""
	}
	msgs := make([]string, len(errors))
	for i, e := range errors {
		msgs[i] = e.Message
	}
	return fmt.Sprintf("%v", msgs)
}
