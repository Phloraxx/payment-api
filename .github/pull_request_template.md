## What changed

<!-- The behaviour being changed and why. Link the issue if there is one. -->

## Affected areas

- [ ] Payment creation / amount allocation
- [ ] Notification parsing
- [ ] Relay pairing, enrollment or signing
- [ ] Matching / lifecycle timing
- [ ] Webhook delivery
- [ ] Merchant API
- [ ] Admin UI
- [ ] Android relay app
- [ ] Deployment / migrations
- [ ] docs only

## Verification

```sh
npm run typecheck:v4
npm run build:v4
go test -race -count=1 ./...
go vet ./...
```

<!-- What the new or changed tests actually protect against. -->

## Evidence and trust impact

- [ ] No change to what counts as origin-bound evidence
- [ ] Evidence rules changed — described below
- [ ] No change to amount allocation or matching
- [ ] Amount allocation or matching changed — described below
- [ ] No merchant-visible response shape change
- [ ] Merchant-visible response shape changed — described below

<!-- If a payment can now reach `paid` through a path it previously could not, say so explicitly. -->

## Migration and rollback

<!-- Schema changes, rollback safety, and what happens to an existing data volume. -->
