# Wave-1 tail bundle (Decision 7e) — 2026-10-17

govulncheck installed on primo (golang.org/x/vuln v1.9.0,
`go install .../cmd/govulncheck@latest`) and run for the first time,
then a deliberate dependency-upgrade pass over discord/go.mod:
discordgo 0.26.1 -> 0.29.0, nats.go 1.31.0 -> 1.54.0, viper 1.18.0 ->
1.21.0, zerolog 1.34.0 -> 1.35.1, uuid 1.4.0 -> 1.6.0,
gorilla/websocket 1.4.2 -> 1.5.3, golang.org/x/crypto 0.16 -> 0.58
(go directive 1.22 -> 1.26.0, raised by the upgraded modules).

govulncheck: BEFORE 1 affecting vulnerability (GO-2026-6278, gorilla
websocket weak PRNG, called via discordgo) + 1 in imported packages +
26 in required modules. AFTER: 0 affecting, 0 in imported packages,
1 remaining in a required-but-uncalled module.
Gates after the pass: go build ok, go vet 0, gofmt clean,
golangci-lint 0 issues (linter itself rebuilt with primo's go1.27 —
the old binary was built with go1.26 and panicked on go1.27 files),
go test -race -count=1 = 23/23 across 6 packages.
