# Task 8 Report

## Completed
- `cmd/audit.go`: added `approved_first` and `needs_fix` glyphs.
- `cmd/audit.go`: added `listingGlyph(state string) (icon, color string)`.
- `cmd/audit.go`: prepended the listing column in `auditOneLiner`.
- `CLAUDE.md`: documented the separate `listing` dimension in `apkgo audit`.

## Verification
- `GOCACHE=/private/tmp/apkgo-gocache GOMODCACHE=/private/tmp/apkgo-gomodcache go build ./... && GOCACHE=/private/tmp/apkgo-gocache GOMODCACHE=/private/tmp/apkgo-gomodcache go vet ./cmd/ ./pkg/store/...`
- `GOCACHE=/private/tmp/apkgo-gocache GOMODCACHE=/private/tmp/apkgo-gomodcache go run . audit -p com.example.nonexistent -s huawei -o json 2>/dev/null || true`
- `GOCACHE=/private/tmp/apkgo-gocache GOMODCACHE=/private/tmp/apkgo-gomodcache go test ./...`

## Result
- Build and vet passed.
- Manual audit command returned structured JSON with an `error` field and did not panic.
- `go test ./...` failed only on the pre-existing baseline issue: `pkg/apkgo TestDiagnose_RealProbe`.

## Notes
- I did not modify tests because the task scope restricted file changes to `cmd/audit.go` and `CLAUDE.md`.
