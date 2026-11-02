# GET URL Shortcuts

A URL shortcut (get) application supporting two deployment modes: static (client-side WASM) and server (HTTP API + persistent SQLite).

## Modes

### Static Mode
- SQLite database embedded in WASM binary, seeded from `shortcuts.json` at build time.
- Read-only UI: resolve, search/filter work; no add/edit/delete.
- Suitable for static hosting (GitHub Pages, GitLab Pages, S3).

### Server Mode
- Go HTTP server on port 8080 with REST API and persistent SQLite database at `./data/shortcuts.db`.
- Full CRUD UI plus database download.
- Suitable for team-shared, dynamic environments.

## Build

### Static
```sh
# Build and run in Docker container (port 8080)
make run-static

# Or build and run in background
make run-static-detached

# Or just build the WASM binary
make build-static
```

### Server
```sh
make build-server
make run-server
# open http://localhost:8080
```

### Based on build.env
```sh
make build
```

## Configuration
Edit `build.env`:
```env
BUILD_MODE=static   # or server
DB_PATH=./data/shortcuts.db
API_PREFIX=/api
SERVER_PORT=8080
DEFAULT_RESOLVER=google  # optional: default search engine for unmatched queries
```
Environment variables override file values.

### Default Resolver
When a shortcut is not found, the application can fall back to a default search engine. Configure this by:

**Server mode**: Set `DEFAULT_RESOLVER` in `build.env` or as an environment variable
**Static mode**: Set `default_resolver` in `shortcuts.json`

Example: If `DEFAULT_RESOLVER=google` and you resolve `?get=anything_not_found`, it will redirect to `?get=google anything_not_found`, which then uses the google shortcut's search URL to search for "anything_not_found".

If no default resolver is configured, unresolved shortcuts will show a "not found" error.

## Migration (server -> static)
```sh
make db-to-json
# or
bash scripts/db-to-json.sh ./data/shortcuts.db shortcuts.json
```
Then rebuild static mode.

## Docker

### Static Mode
```sh
docker build --target static -t get:static .
docker run -p 8080:80 get:static
# open http://localhost:8080
```

### Server Mode
```sh
docker build --target server -t get:server .
docker run -p 8080:8080 -v ./data:/data get:server
# open http://localhost:8080
```

### Using Make
```sh
make docker-static
make docker-server
```

## CI/CD & Automated Deployment

GitHub Actions workflow automatically handles versioning, releases, and deployment:

### Automatic Versioning
- Semantic versioning (MAJOR.MINOR.PATCH) tracked in `VERSION` file
- Patch version auto-increments on every push to `main`
- Manual version bump: edit `VERSION` file and commit

### Automated Releases
Every push to `main` triggers:
1. **Version bump** - Increments patch version in `VERSION`
2. **Build** - Compiles static WASM version
3. **Release** - Creates GitHub release with downloadable archives (`.zip` and `.tar.gz`)
4. **Deploy** - Publishes to GitHub Pages

### GitHub Pages
The static version is automatically deployed to GitHub Pages on every push to `main`.

**URL**: `https://<username>.github.io/<repository>/`

**Features on GitHub Pages**:
- Client-side WASM with embedded SQLite
- Read-only shortcut resolution
- Search and filter functionality
- No backend required

## Notes
- WASM binary ~15MB due to SQLite.
- Server mode creates data directory if missing.
- ncruces/go-sqlite3 pure-Go driver used for both modes.
