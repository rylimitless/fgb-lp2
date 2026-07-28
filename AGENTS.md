# AGENTS.md

## Dev environment tips
- The whole stack runs with a single `docker compose up`. The frontend (SvelteKit/Vite) and backend (Go/Air) both watch for file changes and reload automatically.
- Host ports (see `compose.yaml`): frontend `http://localhost:3039`, backend `http://localhost:5556`, database `localhost:5441`, embeddings `http://localhost:7101`. Inside the containers these are the standard ports (5173, 5555, 5432, 7100).
- Source mounts in `compose.yaml` keep `./src`, `./static`, `./vite.config.ts`, and `./backend` synced so hot-reload works out of the box.
- The database schema in `./backend/database/schema.sql` is applied automatically on first start via the Docker entrypoint.

## Adding new dependencies
- After adding any npm package or Go module, you must rebuild the images:
  1. `docker compose down`
  2. `docker compose build --no-cache`
  3. `docker compose up`
- For npm packages, run `npm install <package>` locally first so `package.json` and `package-lock.json` are updated before the build.
- For Go modules, run `go get <module>` inside `./backend` first so `go.mod` and `go.sum` are updated before the build.

## Testing instructions
- Frontend checks: `npm run check` runs `svelte-check` for TypeScript and template validation.
- Build check: `npm run build` runs `vite build` to confirm the frontend compiles cleanly.
- Backend: `cd backend && go build ./...` confirms the Go code compiles. Run `go vet ./...` for static analysis.
- Fix any type or compilation errors until everything passes.

## PR instructions
- Title format: `[component] <Title>` (e.g. `[frontend] Fix chart re-render bug`, `[backend] Add user auth middleware`)
- Confirm the full stack starts cleanly with `docker compose up` before opening a PR.
- Make sure `npm run check` and `go vet ./...` pass with no errors.
