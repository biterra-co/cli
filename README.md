# Biterra Checker CLI

[![Test](https://github.com/biterra-co/cli/actions/workflows/test.yml/badge.svg)](https://github.com/biterra-co/cli/actions/workflows/test.yml)

Configure and validate access to the **Checker API** used for Attack-Defence (A/D) game automation. The CLI stores your API URL and checker token, optionally team and service, and can export environment variables for the checker process (e.g. in Docker).

## Getting the checker token

The checker API uses a **Bearer token** (one per A/D world). There is no separate login endpoint.

When you run **`biterra init`**, the CLI opens your browser to the **Biterra customer portal** (Account → Developer). Sign in if prompted, create a token for your A/D world in the Developer section, then paste the token back into the terminal. The CLI looks up which world the token is for (via the customer portal) and saves the API URL and token. The token is shown **once** when you create it—copy it when the CLI asks for it.

To get the token manually: open the [Biterra customer portal](https://ctf.biterra.co), go to **Settings** → **Account** → **Developer**, create a token for your world, and copy it. Paste it when you run `biterra init` (you don't need to enter the world URL—the CLI fetches it from the token).

Validation is done by calling `GET /api/ad/checker/rounds/current`: **200** means the token is valid, **401** means invalid or expired (create a new token in the Developer section and update config).

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

1. **Interactive setup:** run `biterra init`. The CLI asks whether to open the Biterra customer portal (Developer section) in your browser or to paste a token you already have. Paste the token when prompted; the CLI looks up which world it's for and validates it.

   ```bash
   biterra init
   biterra init              # choose "browser" or "paste" when prompted
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
| `biterra init` | Interactive setup: choose to open the portal in your browser or paste a token. CLI looks up which world it's for, validates, prompts for team/service, then performs checker check-in (`PUT /teams/instances` with team+service). Saves config. |
| `biterra config set --api-url URL --token TOKEN [--customer-portal-url URL] [--team-uid UID] [--service-uid UID]` | Non-interactive: set and persist API URL, token, and optional customer portal URL, team, and service. |
| `biterra config get` | Print current config (token masked). Use `--show-token` for scripting. |
| `biterra check` | Validate: call `GET /rounds/current`; print success and current round or exit 1 with error. |
| `biterra env` | Print env vars: `BITERRA_API_URL`, `BITERRA_CHECKER_TOKEN`, `BITERRA_TEAM_UID`, `BITERRA_SERVICE_UID`. Default: shell `export` lines; `--format dotenv` for a `.env` block. |
| `biterra run [--interval-seconds N] [--health-url URL]` | Run checker SLA loop: submit SLA only while the current round matches the selected service's round. Optional `--health-url` sets up/down by HTTP 2xx. |

## Config file and precedence

- **Project-local:** `./.biterra.yaml` (or `./.biterra.json`), if present.
- **User-global:** `~/.config/biterra/config.yaml` (or `$XDG_CONFIG_HOME/biterra/config.yaml` on Linux).

**Precedence:** Local over global; environment variables override file values.

**Environment variables:** `BITERRA_API_URL`, `BITERRA_CHECKER_TOKEN`, `BITERRA_CUSTOMER_PORTAL_URL` (optional; default `https://ctf.biterra.co`), `BITERRA_TEAM_UID`, `BITERRA_SERVICE_UID`.

**Optional config:** `customer_portal_url` — base URL of the Biterra customer portal, used when opening the browser and for token-info lookup during `biterra init`. Override locally (e.g. `BITERRA_CUSTOMER_PORTAL_URL=http://localhost:3000` or `biterra config set --customer-portal-url http://localhost:3000`) when running against a local portal.

**Security:** Do not commit the config file (it contains the token). Prefer `chmod 600` on the config file.

## Checker API spec

The canonical OpenAPI 3 spec for the Checker API is in this repo:

- **[api/openapi.yaml](api/openapi.yaml)**

Base path: `/api/ad/checker`. All requests require `Authorization: Bearer <token>`.

## Error messages

- **401 (invalid or expired token):**  
  Create a new token in your Biterra account (Developer section) and run:  
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
