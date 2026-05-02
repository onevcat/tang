# Phase 3 — Repo and PR

- **Date**: 2026-05-02
- **Status**: Complete

## Goal

Implement repository commands and Tangled pull request workflows: repo view/list/create/clone and PR list/view/create/diff/comment/close/reopen/checkout/merge with Tangled's patch-round model.

## Investigation Notes

- `sh.tangled.repo.pull` stores `rounds[]`; each round contains a gzipped format-patch blob.
- Branch-based PR creation uses Knot `sh.tangled.repo.compare` to produce patch data, uploads a gzip blob to the PDS, then writes `sh.tangled.repo.pull`.
- `pr checkout` can only work when `source.branch` exists; patch-only PRs must return a clear unsupported error.
- `pr merge` uses Knot `sh.tangled.repo.merge` with a patch and target branch; gh merge strategies are not represented in the current lexicon.

## Plan

1. Add repo service for list/view/create and clone URL construction.
2. Add PR service for list/view/status/comment/diff and patch blob download.
3. Add PR create helper tests covering compare → gzip upload blob → pull record shape where feasible.
4. Add CLI commands for repo and PR.
5. Validate with unit tests, build/lint, and safe E2E on `tang-playground` where possible.

## Validation Log

- Completed: unit tests cover repo clone URL construction and pull identifier resolution.
- Completed: `go test ./...` passed.
- Completed: `PATH="$(go env GOPATH)/bin:$PATH" make lint` passed with 0 issues.
- Completed: `make build` passed and `./bin/tang --help` lists `repo` and `pr`.
- Completed: `GOOS=linux GOARCH=amd64 go build ./cmd/tang` passed.
- Completed: `tang repo view --json=name,knot,cloneSsh` in a temporary `tang-playground` git context resolved `onev.cat/tang-playground` with knot `knot1.tangled.sh`.
- Completed: `tang repo list onev.cat --json=name,knot` returned 4 repos.
- Completed: `tang repo clone onev.cat/tang-playground <tmp>` succeeded after switching clone behavior to HTTPS; SSH clone to `knot1.tangled.sh` timed out in this environment.
- Completed: `tang repo create tang-phase3-e2e-20260502143344 --knot knot1.tangled.sh ...` created `at://did:plc:kl2ejrmz5zmxnno3ll4luz76/sh.tangled.repo/3mkuueutmtb22`, and `tang repo view onev.cat/tang-phase3-e2e-20260502143344` found it. This empty repo is intentionally left as an E2E artifact because no repo delete command exists in scope.
- Completed: `tang pr list --state all --json=number,title,status,uri` against `tangled.org/core` returned 35 PRs.
- Completed: `tang pr view 1 --json=title,status,branch` against `tangled.org/core` returned a real PR.
- Completed: PR create/diff/comment/close/reopen E2E on `onev.cat/tang`:
  - Created temporary branch `tang-phase3-e2e-20260502143010` from a worktree and pushed it.
  - `tang pr create --base main --head tang-phase3-e2e-20260502143010 --json=uri,title,status,branch` created `at://did:plc:kl2ejrmz5zmxnno3ll4luz76/sh.tangled.repo.pull/3mkuu6q672u22`.
  - `tang pr diff 3mkuu6q672u22` printed the gzipped-patch round as a format-patch.
  - `tang pr comment 3mkuu6q672u22 --body ...` succeeded.
  - `tang pr close 3mkuu6q672u22` and `tang pr reopen 3mkuu6q672u22` succeeded.
- Completed: `tang pr checkout 3mkuu6q672u22` in a temporary git repo fetched and checked out `tang-phase3-e2e-20260502143010`.
- Completed: `tang pr merge` E2E used temporary target/head branches so `main` was not changed:
  - Created target `tang-phase3-target-20260502143109` and head `tang-phase3-merge-20260502143109`.
  - Created PR `at://did:plc:kl2ejrmz5zmxnno3ll4luz76/sh.tangled.repo.pull/3mkuuamwfj322`.
  - `tang pr merge 3mkuuamwfj322 --subject "Merge Phase 3 E2E"` succeeded.
  - Deleted both temporary remote branches afterward.

## Notes

- `repo create` without `--knot` currently tries the configured default `tangled.org`; live validation returned a 404 HTML response for `sh.tangled.repo.create`. The real hosted create-capable knot for this account was `knot1.tangled.sh`, so the successful validation used `--knot knot1.tangled.sh`. This is a real deployment/config distinction, not a PDS hardcoding issue.
- Some old `tangled.org/core` PR records do not contain `rounds[]`; `tang pr diff` correctly reports `pull has no rounds` for those patchless/legacy records.
- `pr merge` does not expose `--squash`, `--rebase`, or `--merge`, matching the current Tangled merge endpoint.

## Completion

Phase 3 is complete. The remaining operational caveat is choosing a create-capable Knot host for `repo create`; this is configurable with `--knot` and `knot.hosts`.
