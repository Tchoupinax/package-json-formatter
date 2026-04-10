# package-json-formatter

CLI tool that formats `package.json` files: stable key order, tidy `scripts` and `exports`, optional script and key rules from a YAML config, and monorepo-friendly discovery.

## Installation

```
brew tap tchoupinax/brew
brew install tchoupinax/brew/pjf
```

## Build

```bash
go build -o pjf ./cmd/pjf
```

Run from source (use the **`./cmd/pjf` package**, not `main.go` alone, or helpers like `newUI` in other files are missing):

```bash
go run ./cmd/pjf -- -w .
```

Needs Go 1.26+ (see `go.mod`). Tool versions for local dev are optional in `mise.toml`.

## CI

GitHub Actions (`.github/workflows/`):

- **`pull-request.yaml`**: lint (`golangci-lint`) and `go test` / `go build` on every PR.
- **`master.yml`**: same jobs on push to `main` or `master`.

`GOFLAGS=-buildvcs=false` is set so builds work without full git metadata.

## Usage

```bash
# Format one package (stdout). Implies a single path.
./pjf path/to/package.json
./pjf path/to/dir          # that directory's package.json

# Monorepo: format every package.json under the given roots (stdout refused; use -w)
./pjf -w .
./pjf -w ./packages ./apps

# Write mode is required when more than one file would be printed
./pjf -w .
```

### Flags

| Flag | Meaning |
|------|---------|
| `-config <path>` | YAML config file. If omitted, `./pjf.yml` in the current working directory is used when it exists. |
| `-w` | Write results back to each file. |
| `-r` | Recursive discovery (default `true`). With `-r=false`, only the exact `package.json` for each path is used. |

Run `./pjf -h` for the built-in help text.

Progress is printed on **stderr**: status (` ok` / `FAIL`), path (relative to the current directory when possible), human-readable time, and any error text. With more than one file you get a small table, header line (`pjf` and file count), and a footer with totals and wall time. Colors are used when stderr is a terminal. Formatted JSON still goes only to **stdout** for a single file without `-w`, so you can redirect it safely.

## Config (`pjf.yml`)

Optional. Paths in the config (`roots`, ignore globs, per-file rules) are relative to the **directory that contains the config file**.

See `package-json-formatter.example.yaml` for a commented template. Main ideas:

- **`keyOrder`**: Preferred order for top-level keys; any other keys follow in alphabetical order.
- **`scripts`**: Script entries merged into every matched file (overwrites the same script name).
- **`scriptsIgnore`**, **`scriptsFiles`**: Skip script merging for some globs, or add scripts only for matching paths.
- **`ensureKeys`**, **`ensureKeysIgnore`**, **`ensureKeysFiles`**: Add top-level keys only when they are missing.
- **`pinDependencyVersions`**: When true (default), strip a leading `^` or `~` in dependency maps. Set to `false` to keep semver ranges as written.
- **`roots`**: If set, only discover `package.json` under these directories (and you must load a config file so paths resolve).

## Behavior notes

- **`scripts`**: `pre*`, main name, and `post*` groups are kept together for the same stem (`prebuild`, `build`, `postbuild`).
- **Dependencies**: `dependencies`, `devDependencies`, `peerDependencies`, and `optionalDependencies` get sorted keys; pinning affects version strings as above when enabled.
- **HTML escaping**: `<`, `>`, and `&` in strings are not escaped as `\u00xx` in output.
