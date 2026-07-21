# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Security
- **Admin MySQL protocol now authenticates.** The handshake response is verified
  against the configured credential using `mysql_native_password`; unauthenticated
  or wrong-password connections are rejected (error 1045). When no password is
  configured, only loopback connections are accepted.
- **Removed dev-mode auth bypass.** The admin dashboard no longer accepts "any
  password" when no hash is configured. If `admin_password_hash` is unset, a random
  bootstrap password is generated and logged once at startup (fail-closed).
- **Fixed stored XSS in the monitoring and admin dashboards.** All user/log-derived
  values (model, username, server name/endpoint/comment, health names) are now
  HTML-escaped at render, and dashboards send a `Content-Security-Policy` plus
  `X-Frame-Options`, `X-Content-Type-Options`, and `Referrer-Policy` headers.
- **Admin plane binds to loopback (`127.0.0.1`) by default**, matching ProxySQL.
- Session cookie `Secure` flag is now driven by `tls_enabled`.
- Added `mindbalancer -hash-password` to generate a bcrypt `admin_password_hash`.

### Fixed
- **Encrypted API keys were sent to upstream providers.** The balancer, health
  checker, referee, and model-listing paths now decrypt keys before use, while the
  admin/display paths keep them encrypted/masked.
- **Graceful shutdown drain counter was dead.** In-flight proxy requests are now
  tracked, so shutdown actually waits for them to drain.
- **Google (Gemini) embeddings** now report `SupportsEmbeddings() == false`, so the
  proxy returns a clean 400 instead of a 500.
- **Default port collision.** Proxy and admin HTTP no longer both bind `6033`;
  ports are now proxy `6034`, admin HTTP `6033` (`admin_http_port`), admin MySQL
  `6032`.

### Added
- `admin_http_port` and `admin_password` configuration keys.
- GitHub Actions CI (build, vet, gofmt, `go test -race`) and a `.golangci.yml`.
- Unit tests for `crypto`, `storage` (key-encryption round-trip), and the MySQL
  `mysql_native_password` verifier.
- A working multi-stage `Dockerfile`.

### Changed
- `go.mod` tidied (dropped unused `zerolog`); repository formatted with `gofmt`.
- Documentation fixes: license badge (Apache-2.0), clone directory names, Go
  version, and contact details aligned across README/CONTRIBUTING.
