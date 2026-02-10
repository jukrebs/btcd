# Draft: SCION support via PAN (SCION_APPS + pan library)

## Goal (as stated)
- Compare `main`, `mvp`, and `vanilla_testing` branches to understand what they do.
- Update `SCION_APPS` and the PAN library.
- Implement SCION support with PAN in a way that is elegant (not hacky).

## Confirmed choices
- PAN library: Go PAN (SCION)
- Automated tests: No automated tests (agent-executed QA scenarios only)
- SCION support scope: P2P connections (peer dial/listen)
- Config surface: Both config file + env vars first-class
- Target base branch: main/master
- Peer selection: Auto-detect by address format
- Security posture: Secure by default (do not default to InsecureSkipVerify)

## Terminology (needs disambiguation)
- You said: “scion apps is the repository where the pan library is in”.
  - Likely meaning: **scion-apps repo** (e.g., `github.com/netsec-ethz/scion-apps`) which contains the Go `pkg/pan` library.
  - We still need to clarify whether **SCION_APPS** is:
    - an env var used by btcd to find a *local checkout path* of scion-apps (for go.mod replace/dev), OR
    - a legacy name you want to keep, but implemented as proper btcd.conf keys, OR
    - something else.
- SCION support scope: P2P connections (peer dial/listen)
- Config surface: Both config file + env vars first-class (SCION_APPS supported)
- Target base branch: main/master

## What I’m doing now (research in progress)
- Launched repo exploration to map branch differences and current integration points.
- Launched test-infra discovery across branches.
- Launched external research to disambiguate which “PAN” library this repo uses and recommended idiomatic integration patterns.

## Research Findings (repo)
- Current repo (btcd) has **no occurrences** of `SCION_APPS`, `scion`, `github.com/scionproto`, or a PAN dependency in `go.mod` (based on repo-wide search).
- Natural integration points for adding SCION/PAN transport (based on structure):
  - `v2transport/transport.go` (transport abstraction / lowest-level transport layer)
  - `netsync/interface.go`, `netsync/manager.go` (higher-level network/sync orchestration)
  - `sample-btcd.conf`, `docs/configuration.md` (user-facing configuration/documentation)

## Research Findings (tests)
- Test framework: standard `go test` driven via `Makefile`.
- Integration/E2E harness exists: `integration/rpctest/rpc_harness.go` (spawns btcd nodes and drives them via RPC).
- CI (GitHub Actions): `.github/workflows/main.yml` runs `make build`, `make unit-cover`, `make unit-race`.
- Branch differences spotted so far:
  - `mvp` and `vanilla_testing` branches appear to differ mainly in `Makefile` test/coverage tooling (e.g., go-acc usage), not SCION/PAN (pending full branch diff results).

## Branch comparison findings (SCION/PAN)
- `main/master`: baseline btcd; no SCION/PAN integration.
- `mvp`: adds SCION support via `github.com/netsec-ethz/scion-apps` (PAN + quicutil) + local adapter `scion/scion.go`; wires through peer/server/config/address parsing.
- `vanilla_testing`: does not include SCION/PAN integration (appears focused elsewhere).

## mvp branch: concrete SCION/PAN touch points (reference list)
- `scion/scion.go`: PAN adapter/wrapper (Dial/Listen/ParseAddr/SplitHostPort/JoinHostPort; uses PAN+QUIC).
- `go.mod`, `go.sum`: adds scion-apps + replace directives.
- P2P & server wiring:
  - `peer/peer.go`, `server.go`, `config.go`, `addrmgr/addrmanager.go`
- Secondary address parsing sites:
  - `netsync/manager.go`, `rpcserver.go`, `btcutil/certgen.go`, `cmd/btcctl/config.go`

## Potentially “hacky” areas to address (from mvp)
- Peer detection via direct type assertion to `pan.UDPAddr` in `peer/peer.go` (brittle shortcut; leaks PAN types into core).
- Hardcoded SCION default port (8666) in `scion/scion.go`.
- Adapter hardcodes behavior (ping intervals/timeouts) and uses `InsecureSkipVerify` for QUIC/TLS in `scion/scion.go`.
- go.mod replace directives for SCION deps (module hygiene / long-term stability).

## External research findings (PAN library behavior / gotchas)
- Go PAN library: `github.com/netsec-ethz/scion-apps/pkg/pan` (Policy + Selector, DialUDP/ListenUDP/DialQUIC/ListenQUIC).
- Gotchas to account for in a clean btcd integration:
  - Wildcard bind behavior differs (PAN may rewrite wildcard to a default local IP; affects reachability).
  - Path lookup for unverified peers can be abused (avoid doing expensive lookups on arbitrary inbound input).
  - Reply-path caching can be vulnerable to spoofing/hijack; default server-side behavior may not failover on PathDown.

## Branch comparison findings (SCION/PAN)
- `main/master`: baseline btcd; no SCION/PAN integration.
- `mvp`: SCION support via `github.com/netsec-ethz/scion-apps` (PAN + quicutil) + local adapter `scion/scion.go`; changes in peer/server/config/address parsing.
- `vanilla_testing`: no SCION/PAN integration; appears focused on other features/testing.

## Potentially “hacky” areas to address (from mvp)
- Peer detection via direct type assertion to `pan.UDPAddr` (brittle shortcut): `peer/peer.go`
- Hardcoded SCION default port 8666 in adapter: `scion/scion.go`
- Adapter hardcodes PAN/QUIC behavior and uses `InsecureSkipVerify` (needs hardening/config): `scion/scion.go`
- go.mod uses replace directives for SCION deps (module hygiene): `go.mod` (mvp)
- Coverage pipeline uses sed to massage paths (unrelated, but indicates branch drift): `Makefile` (mvp)

## Assumptions (UNCONFIRMED)
- “SCION_APPS” is either an env var/config listing SCION-enabled applications or a build/runtime selector.
- “PAN library” is a SCION-related dependency used as the integration layer for SCION networking.

## Open Questions
1) What is the desired user-facing behavior of “SCION support with PAN” (what should work that doesn’t today)?
2) What’s currently “hacky” (specific files/approach you want removed)?
3) What environments must be supported (Linux only? CI? containerized?)
4) Are we allowed to change public APIs, or must integration be backwards-compatible?
5) Any constraints on dependency upgrades (Go version, module versions, pinned commits)?

## Scope boundaries (to confirm)
- INCLUDE: branch comparison, refactor/integration plan, SCION_APPS + pan updates, verification strategy.
- EXCLUDE (unless you ask): broad refactors unrelated to SCION/PAN, feature work outside networking/integration.
