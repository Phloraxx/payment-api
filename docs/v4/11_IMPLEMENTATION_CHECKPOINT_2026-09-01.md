# 11 — Implementation checkpoint — 2026-09-01

## Purpose

This is the source-integration checkpoint before v4 staging and production cutover. It records what is implemented and validated without implying that v4 is already live.

## Merged source

- `payment-api/main`: standalone direct-SQLite v4 domain/runtime, merchant/admin/relay APIs, webhooks, migration/corroboration and Digital Asset Links through merge `5fb379c367d871faf61812778e41b8f3a3999428`.
- `paygate-relay-android/main`: unified dark Android v0.5 source through `db061d260b5552de408250abb825513f8ee5b1a5`.
- `payment-frontend/main`: provider-blind test/customer integration through `a34ef147adeeb56b5227195b288c1a640ababd1f`.

## Android release evidence

Main-branch artifact `paygate-v0.5.0-alpha1-release` from workflow run `33494718346` was downloaded but **not installed**.

- APK SHA-256: `e15b8846dcfaeb35ac7bd419d203385f325d5ca6d75449e8664b0e5a0c82b6e9`
- package: `io.github.phloraxx.paygaterelay`
- versionCode: `11`
- versionName: `0.5.0-alpha1`
- signer SHA-256: `412b8f66c06acd93958c4dd11caa3214ba950059c00e57159d5673f94700d44a`
- signer SHA-1: `7e2eb9467839db74f8f25846f9e3fb65de6cfe91`

The signer exactly matches the byte-for-byte installed v0.4.2 release lineage. Normal upgrade policy remains in-place only: no uninstall, clear-data or forced re-pair.

## V4 web dashboard

Branch `v4/web-dashboard` implements a separate React/Vite bundle embedded only by `paygate-v4`:

- Overview;
- Payments;
- Activity;
- Settings;
- password-only login;
- collection profiles;
- webhook/API-key management;
- phone state and replacement pairing;
- password change with admin-session revocation.

The browser uses only the `Secure`, `HttpOnly`, `SameSite=Strict`, `Path=/admin` session cookie. No admin bearer token is persisted in browser storage.

## Local acceptance evidence

The dashboard branch passed:

- `npm audit`: zero vulnerabilities;
- v3 + v4 TypeScript;
- existing worker tests;
- v3 + v4 Vite production builds;
- `git diff --check` / Go formatting;
- `go test -count=1 ./...`;
- `go test -race` for all v4 packages;
- `go vet`;
- standalone v4 binary + migrator builds;
- legacy v3 Docker build;
- dedicated v4 Docker build.

A disposable exact-v4-image runtime smoke with a fresh temporary SQLite DB verified:

- `/health` = healthy/database ok/v4;
- dashboard root and security headers;
- unauthenticated `/admin/overview` = 401;
- password login issues the expected protected cookie;
- authenticated Overview = 200;
- `/device/pair/<token>` SPA browser fallback;
- Android `assetlinks.json` package/signer binding;
- built-in `paygate-v4 healthcheck`;
- Docker health status = `healthy`.

## Explicit non-actions

At this checkpoint there has been no:

- v4 production deployment;
- live v3/PocketBase database mutation by v4 tooling;
- production v3 -> v4 cutover;
- historical webhook replay;
- Android v0.5 installation;
- uninstall/clear-data/re-pair workaround;
- PocketBase/libgm production removal.

Production remains v3 until Phase 13 staging acceptance and the Phase 14 cutover runbook are satisfied.
