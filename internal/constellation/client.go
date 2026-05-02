package constellation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"tangled.org/onev.cat/tang/internal/config"
)

type Record struct {
	DID        string `json:"did"`
	Collection string `json:"collection"`
	RKey       string `json:"rkey"`
}

type Result struct {
	Total   int      `json:"total"`
	Records []Record `json:"records"`
	Cursor  string   `json:"cursor,omitempty"`
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	if env := os.Getenv("TANG_CONSTELLATION_URL"); env != "" {
		baseURL = env
	}
	if baseURL == "" {
		baseURL = config.DefaultConstellationURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), HTTPClient: httpClient}
}

func (c *Client) GetBacklinks(ctx context.Context, target, collection, path string, limit int, cursor string) (*Result, error) {
	u, err := url.Parse(c.BaseURL + "/links")
	if err != nil {
		return nil, err
	}
	query := u.Query()
	query.Set("target", target)
	query.Set("collection", collection)
	query.Set("path", path)
	query.Set("limit", strconv.Itoa(limit))
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("constellation API error: %s", resp.Status)
	}

	var wire struct {
		Total          int      `json:"total"`
		LinkingRecords []Record `json:"linking_records"`
		Cursor         *string  `json:"cursor"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wire); err != nil {
		return nil, err
	}
	result := &Result{Total: wire.Total, Records: wire.LinkingRecords}
	if wire.Cursor != nil {
		result.Cursor = *wire.Cursor
	}
	return result, nil
}
