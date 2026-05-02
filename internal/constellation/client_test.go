package constellation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetBacklinksMapsLinkingRecords(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total": 1,
			"linking_records": []map[string]string{{
				"did":        "did:plc:test",
				"collection": "sh.tangled.repo.issue",
				"rkey":       "abc",
			}},
			"cursor": "next",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	result, err := client.GetBacklinks(context.Background(), "at://did/repo/rkey", "sh.tangled.repo.issue", ".repo", 10, "cur")
	if err != nil {
		t.Fatalf("GetBacklinks error = %v", err)
	}
	if result.Total != 1 || result.Cursor != "next" || result.Records[0].RKey != "abc" {
		t.Fatalf("result = %#v", result)
	}
	for _, want := range []string{"target=at%3A%2F%2Fdid%2Frepo%2Frkey", "collection=sh.tangled.repo.issue", "path=.repo", "limit=10", "cursor=cur"} {
		if !containsQuery(gotQuery, want) {
			t.Fatalf("query %q missing %q", gotQuery, want)
		}
	}
}

func containsQuery(query, part string) bool {
	for _, item := range splitQuery(query) {
		if item == part {
			return true
		}
	}
	return false
}

func splitQuery(query string) []string {
	if query == "" {
		return nil
	}
	var parts []string
	start := 0
	for i, ch := range query {
		if ch == '&' {
			parts = append(parts, query[start:i])
			start = i + 1
		}
	}
	return append(parts, query[start:])
}
