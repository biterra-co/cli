# Biterra Checker CLI

[![Test](https://github.com/biterra-co/cli/actions/workflows/test.yml/badge.svg)](https://github.com/biterra-co/cli/actions/workflows/test.yml)

Configure and validate access to the **Checker API** used for Attack-Defence (A/D) game automation. The CLI stores your API URL and checker token, optionally team and service, and can export environment variables for the checker process (e.g. in Docker).

## Getting the checker token

The checker API uses a **Bearer token** (one per A/D world). There is no separate login endpoint.

1. Open the **world portal** for your event.
2. Go to the **A/D** (Attack-Defence) tab.
3. Click **Rotate token**. The new token is shown **once**; copy it and store it securely.
4. Use this token when running `biterra init` or `biterra config set --token <token>`.

Validation is done by calling `GET /api/ad/checker/rounds/current`: **200** means the token is valid, **401** means invalid or expired (rotate the token again and update config).

## Installation

### 1. Pre-built binaries (recommended)

Download the latest release for your platform from [GitHub Releases](https://github.com/biterra-co/cli/releases):

- `linux/amd64`, `linux/arm64`
- `darwin/amd64`, `darwin/arm64`
- `windows/amd64`

Unpack the tarball or zip and put the `biterra` binary on your `PATH` (e.g. `~/bin` or `/usr/local/bin`).

### 2. Go install

If you have Go 1.21+ installed:

```bash
go install github.com/biterra-co/cli@latest
```

Ensure `$GOPATH/bin` or `$HOME/go/bin` is on your `PATH`.

## Quick start

1. **Interactive setup** (prompts for API URL, token, then team and service):

   ```bash
   biterra init
   ```

2. **Validate** that the token works:

   ```bash
   biterra check
   ```

3. **Export env** for your checker process (e.g. for Docker):

   ```bash
   eval $(biterra env)
   ```

   Or write a `.env`-style block for `docker run --env-file`:

   ```bash
   biterra env --format dotenv > .env.checker
   docker run --env-file .env.checker your-checker-image
   ```

## Commands

| Command | Description |
|--------|-------------|
| `biterra init` | Interactive setup: API URL, token (validated), then optional team and service. Saves config. |
| `biterra config set --api-url URL --token TOKEN [--team-uid UID] [--service-uid UID]` | Non-interactive: set and persist API URL, token, and optional team/service. |
| `biterra config get` | Print current config (token masked). Use `--show-token` for scripting. |
| `biterra check` | Validate: call `GET /rounds/current`; print success and current round or exit 1 with error. |
| `biterra env` | Print env vars: `BITERRA_API_URL`, `BITERRA_CHECKER_TOKEN`, `BITERRA_TEAM_UID`, `BITERRA_SERVICE_UID`. Default: shell `export` lines; `--format dotenv` for a `.env` block. |

## Config file and precedence

- **Project-local:** `./.biterra.yaml` (or `./.biterra.json`), if present.
- **User-global:** `~/.config/biterra/config.yaml` (or `$XDG_CONFIG_HOME/biterra/config.yaml` on Linux).

**Precedence:** Local over global; environment variables override file values.

**Environment variables:** `BITERRA_API_URL`, `BITERRA_CHECKER_TOKEN`, `BITERRA_TEAM_UID`, `BITERRA_SERVICE_UID`.

**Security:** Do not commit the config file (it contains the token). Prefer `chmod 600` on the config file.

## Checker API spec

The canonical OpenAPI 3 spec for the Checker API is in this repo:

- **[api/openapi.yaml](api/openapi.yaml)**

Base path: `/api/ad/checker`. All requests require `Authorization: Bearer <token>`.

## Error messages

- **401 (invalid or expired token):**  
  Rotate the token in the world portal and run:  
  `biterra config set --token NEW_TOKEN`

- **No config:**  
  Run `biterra init` or set `BITERRA_API_URL` and `BITERRA_CHECKER_TOKEN`.

- **Network / 5xx:**  
  Check the API URL and connectivity; the CLI exits non-zero with a clear message.

## Development

- **Tests:** `make test` or `go test ./...`
- **Build:** `make build` (produces `biterra` in the project directory)
- **Lint:** `make vet` runs `go vet ./...`
- **Install locally:** `make install` (builds and runs `go install`)

CI runs the test suite and `go vet` on every push and pull request ([`.github/workflows/test.yml`](.github/workflows/test.yml)).
