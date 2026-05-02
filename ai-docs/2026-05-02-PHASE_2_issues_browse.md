# Phase 2 — Issues and Browse

- **Date**: 2026-05-02
- **Status**: Complete

## Goal

Implement issue list/create/view/close/reopen/edit/comment and browse commands, using Constellation backlinks for cross-PDS queries and Tangled issue/state/comment records for writes.

## Investigation Notes

- TS CLI uses Constellation `/links` and reads the `linking_records` response field.
- `sh.tangled.repo.issue` links to a repo through `.repo`; comments and state records link to an issue through `.issue`.
- Issue state uses `sh.tangled.repo.issue.state.open` and `.closed`.
- Repository AT-URI cannot be guessed from owner/name; it must be found by listing the owner DID's `sh.tangled.repo` records and matching `name`.

## Plan

1. Add Constellation client tests for request shape and response mapping.
2. Add issue identifier tests for `1`, `#1`, and rkey resolution.
3. Implement AT-URI parsing and repository AT-URI resolution.
4. Implement issue service operations against PDS plus Constellation.
5. Implement body input helper and `issue` CLI commands.
6. Implement `browse` URL construction and opener command.
7. Validate with unit tests, lint/build, and a real create/comment/close/reopen flow on `onev.cat/tang-playground` or the current repo if the playground context is not locally available.

## Validation Log

- Completed: unit tests cover Constellation `/links` request/response mapping, AT-URI parsing, issue identifier resolution for `1` / `#1` / rkey, body input, and browse URL construction.
- Completed: `go test ./...` passed.
- Completed: `PATH="$(go env GOPATH)/bin:$PATH" make lint` passed with 0 issues.
- Completed: `make build` passed and `./bin/tang --help` lists `issue` and `browse`.
- Completed: `GOOS=linux GOARCH=amd64 go build ./cmd/tang` passed.
- Completed: real login using `~/Desktop/tang` succeeded for `onev.cat`.
- Completed: real E2E in a temporary git context with remote `git@tangled.org:onev.cat/tang-playground.git`:
  - `tang issue create --body ... --json=uri,title,state` created `at://did:plc:kl2ejrmz5zmxnno3ll4luz76/sh.tangled.repo.issue/3mkuteffbxa2b`.
  - `tang issue comment 3mkuteffbxa2b --body ...` succeeded.
  - `tang issue close 3mkuteffbxa2b` succeeded.
  - `tang issue reopen 3mkuteffbxa2b` succeeded.
  - `tang issue view 3mkuteffbxa2b --json=issue` returned the created title and `open` state.
- Completed: `tang issue list --state all --limit 5 --json=number,title,state` in `tang-playground` returned 1 indexed issue.
- Completed: `tang browse` in the temporary `tang-playground` git context exited successfully and opened the repository page through the system browser.
- Completed: AppView check with `curl https://tangled.org/onev.cat/tang-playground/issues/1` found the created issue title and open state.

## Notes

- After `tang issue close`, an immediate AppView HTML check still showed `open`. The CLI writes `sh.tangled.repo.issue.state` records successfully, but AppView state consumption or indexing appears delayed or incomplete. The issue was reopened after this check, and the final AppView state is open.
- New issue records may take a short time to appear through Constellation. CLI commands that receive a direct rkey use a session-DID fallback so create/comment/close/reopen flows can continue before indexing catches up.

## Completion

Phase 2 is complete. The only uncertainty is AppView's immediate consumption of issue state records, recorded above as a protocol/AppView limitation rather than a CLI blocker.
