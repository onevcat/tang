# Phase 1 — Auth and Basics

- **Date**: 2026-05-02
- **Status**: In Progress

## Goal

Implement the first usable foundation: config store, AT Protocol identity resolution, session/keyring, refresh logic, PDS and Knot clients, auth/config/status/ssh-key commands, and Tangled repository context resolution.

## Investigation Notes

- Tangled docs describe Tangled as decentralized and self-hostable; knots are lightweight Git hosting servers and the appview aggregates repositories across knots. Source: https://docs.tangled.org/
- `tangled.org/core/idresolver` exposes `DefaultResolver(plcURL)` and identity objects expose `PDSEndpoint()`.
- Indigo generated APIs provide `com.atproto.server.createSession`, `refreshSession`, and `getServiceAuth`.
- The current TS CLI only checks fetch URLs for Tangled remotes; this phase fixes that by checking fetch first, then push URL.

## Plan

1. Add logic tests first for config keys, git remote parsing, repository selection, session refresh thresholds, and service-auth request construction.
2. Implement config storage with defaults and user/project overrides.
3. Implement identity resolution without any hardcoded PDS fallback.
4. Implement auth session persistence and refresh with injectable stores/clients for testing.
5. Implement PDS/Knot clients with separate user access JWT and service-auth token paths.
6. Implement git remote and repository context resolution using configured knot hosts.
7. Add Cobra commands for `auth`, `config`, `status`, and `ssh-key`.
8. Run unit tests, lint, build, and limited live checks that do not mutate the user's Tangled account unless explicitly needed.

## Validation Log

- Completed: unit tests cover config defaults/set/reload/unsupported `pds.url`, session PDS requirement/JWT expiry/refresh boundary, service-auth `aud/lxm/exp`, git remote parsing including fetch-first and push fallback, and repository remote selection.
- Completed: `go test ./...` passed.
- Completed: `PATH="$(go env GOPATH)/bin:$PATH" make lint` passed with 0 issues.
- Completed: `make build` passed and `./bin/tang --help` lists `auth`, `ssh-key`, `status`, and `config`.
- Completed: `GOOS=linux GOARCH=amd64 go build ./cmd/tang` passed.
- Completed: `./bin/tang config list` shows implicit defaults: `knot.hosts=tangled.org`, `constellation.url=https://constellation.microcosm.blue`, `appview.url=https://tangled.org`.
- Completed: isolated config write with a temporary `HOME`: `tang config set knot.hosts tangled.org,knot.example.com`, then `tang config list` showed both hosts.
- Completed: real login using the account data from `~/Desktop/tang`: `tang auth login` succeeded for `onev.cat`.
- Completed: `tang auth status` and `tang status` showed authenticated handle/DID, detected current repository `onev.cat/tang`, and reachable AppView/Constellation services.
- Completed: `tang auth token --json token` returned structured JSON containing a token field.
- Completed: `tang auth refresh` succeeded and updated the keychain session.
- Completed: SSH key E2E used a generated temporary ed25519 public key: `tang ssh-key add`, `tang ssh-key list`, and `tang ssh-key delete` all succeeded; the temporary key was removed afterward.
- Completed: push URL fallback E2E used a temporary git repo with `fetch=github.com` and `push=tangled.org`; `tang status --json` detected `onev.cat/tang` with `urlKind=push`.
- Completed: `tang auth logout` cleared the keychain session and `tang auth status` reported `(not authenticated)`.

## Notes

- The original plan expected `onev.cat` to resolve to `https://tngl.sh`; live DID resolution on 2026-05-02 returned `https://discina.us-west.host.bsky.network`. The implementation does not hardcode either value and uses the DID document result, which is the required behavior for migrated or custom-PDS accounts.
- The SSH-key E2E initially exposed that this PDS rejects unknown lexicons when `validate=true`. `PDSClient.CreateRecord` now sets `validate=false` for Tangled records, which keeps the CLI usable across PDS implementations that do not know Tangled schemas locally.

## Completion

Phase 1 is complete. No human-only validation remains for this phase.
