# Repository Guidelines

## Build, Test, and Development Commands
- `make` builds server + client using vendored modules and runs `go-assets.sh`.
- `make client` builds a CGO-free client; `make debug` keeps symbols.
- `make pb` regenerates protobuf stubs from `protobuf/`.
- `./go-tests.sh` runs the Go test suite, this is the only way you should run unit tests.
- Docs site: `cd docs/sliver-docs && npm install && npm run dev` for local dev; `npm run build` for a static build.

## Coding Style & Naming Conventions
- Run `gofmt` on all Go code; keep package names lowercase and idiomatic.
- Avoid `CGO` and empty interfaces; prefer explicit types and interfaces.
- Do not import `server` packages from `client`.

## Testing Guidelines
- Prefer `docker build --target test --build-arg GO_TESTS_FLAGS=--skip-generate .` 
- Add or update tests when behavior changes; some server packages run longer tests.

## Release Process (delifer branch)

Tags follow the convention `delifer-v<major>.<minor>.<patch>` (e.g. `delifer-v1.7.10`).
Release titles use `delifer v<major>.<minor>.<patch>`.

### Building release binaries

Every release ships **four** binaries built from the same commit:

```bash
# macOS Apple Silicon client
CGO_ENABLED=0 go build -mod=vendor -trimpath -tags go_sqlite,client $(LDFLAGS) -o sliver-client-darwin-arm64 ./client

# Linux x86_64 client
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -mod=vendor -trimpath -tags go_sqlite,client $(LDFLAGS) -o sliver-client-linux-amd64 ./client

# macOS Apple Silicon server
CGO_ENABLED=0 go build -mod=vendor -trimpath -tags go_sqlite,server $(LDFLAGS) -o sliver-server-darwin-arm64 ./server

# Linux x86_64 server
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -mod=vendor -trimpath -tags go_sqlite,server $(LDFLAGS) -o sliver-server-linux-amd64 ./server
```

Where `$(LDFLAGS)` is `-ldflags "-s -w -X ...PublicKey=... -X ...ArmoryPublicKey=... -X ...ArmoryRepoURL=..."` (see Makefile for full values).

### Creating a release

```bash
# 1. Tag the commit
git tag -a delifer-v1.7.X -m "delifer-v1.7.X: <summary>"
git push solasec delifer-v1.7.X
git push origin delifer-v1.7.X

# 2. Create the GitHub release on SolaSec with all 4 binaries
gh release create delifer-v1.7.X \
  --repo SolaSec/sliver \
  --title "delifer v1.7.X" \
  --notes "<release notes>" \
  sliver-server-linux-amd64 \
  sliver-server-darwin-arm64 \
  sliver-client-darwin-arm64 \
  sliver-client-linux-amd64
```

### After building locally

Copy the macOS client into the PATH so the MCP server uses it:

```bash
cp sliver-client-darwin-arm64 ~/.local/bin/sliver-client
```

