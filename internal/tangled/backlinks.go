package tangled

import (
	"context"
	"net/http"

	"tangled.org/onev.cat/tang/internal/constellation"
)

func repoBacklinks(ctx context.Context, constellationClient *constellation.Client, httpClient *http.Client, repoURI, collection, path string, limit int, cursor string) ([]constellation.Record, error) {
	targets := []string{}
	if repoDID, err := ResolveRepoDID(ctx, repoURI, httpClient); err == nil && repoDID != "" {
		targets = append(targets, repoDID)
	}
	if !contains(targets, repoURI) {
		targets = append(targets, repoURI)
	}

	seen := map[string]bool{}
	records := []constellation.Record{}
	for _, target := range targets {
		backlinks, err := constellationClient.GetBacklinks(ctx, target, collection, path, limit, cursor)
		if err != nil {
			return nil, err
		}
		for _, record := range backlinks.Records {
			key := record.DID + "/" + record.Collection + "/" + record.RKey
			if seen[key] {
				continue
			}
			seen[key] = true
			records = append(records, record)
		}
	}
	return records, nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
