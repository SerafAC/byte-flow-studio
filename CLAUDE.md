
## Active Technologies
- Go 1.23+ (core backend); TypeScript 5.x (frontend) (001-serial-data-studio)
- SQLite via `modernc.org/sqlite` (CGO-free, cross-compilable) — workflow definition + session data stored in a single `.byteflow` file (SQLite database) (001-serial-data-studio)
- Go 1.23+ (backend), TypeScript 5.x / Vue 3.5 (frontend) + Wails v3 alpha, Vue Flow 1.x, uPlot, PrimeVue 4.x, Pinia 2.x, go.bug.st/serial, gonum (FFT), gorilla/websocket (001-serial-data-studio)
- SCSS (via `sass` dev dependency) — use `<style lang="scss">` in `.vue` SFCs; no vite config required (001-serial-data-studio)
- SQLite via `modernc.org/sqlite` (CGO-free). Single `.byteflow` file per workflow (SQLite DB). (001-serial-data-studio)
- Go 1.23+ (backend), TypeScript 5.x (frontend) + Vue 3.2, Wails v3 alpha, Vue Flow 1.48, `@vue-flow/node-resizer` (new), uPlot 1.6, PrimeVue 4.5, Pinia 3, SCSS (sass) (003-nocturnal-redesign-canvas-analysis)
- SQLite via `modernc.org/sqlite` — workflow as JSON blob; `BlockDef` extended with `width`/`height float64` (backward-compatible, `omitempty`) (003-nocturnal-redesign-canvas-analysis)

## Recent Changes
- 001-serial-data-studio: Added Go 1.23+ (core backend); TypeScript 5.x (frontend)
