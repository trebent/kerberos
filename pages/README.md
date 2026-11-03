# Kerberos GitHub Pages site

The site published at <https://trebent.github.io/kerberos/>. It is a small Angular single-page app
whose main job is the **API reference**: it renders the OpenAPI contracts in
[`../openapi/`](../openapi/) with Redoc, or shows their YAML source. The landing page has a quick
start, an overview of how the gateway handles a backend call, and links into
[`../docs/`](../docs/README.md). Explanations belong in `docs/`, not here: link to them.

It follows the conventions of [angular-bootstrap](https://github.com/maansaake/angular-bootstrap):
Angular 22 standalone components and signals, Angular Material 3, ESLint, Prettier, and Vitest
specs with axe-core accessibility checks.

## Commands

Run these in `pages/`. Node.js 22 or later is required.

| Command            | What it does                                             |
| ------------------ | -------------------------------------------------------- |
| `make install`     | Install dependencies (`npm ci`)                          |
| `make serve`       | Start the dev server on `http://localhost:4200`          |
| `make check`       | Run everything CI runs: format, lint, build and test     |
| `make format`      | Format the code with Prettier                            |
| `make lint`        | Lint and auto-fix with ESLint                            |
| `make build/prod`  | Production build into `dist/kerberos-pages/browser/`     |
| `make build/pages` | The build GitHub Pages serves; `BASE_HREF` sets its path |
| `make test`        | Run the unit tests once                                  |

## Routes

| Path                     | Page                                                             |
| ------------------------ | ---------------------------------------------------------------- |
| `/`                      | Landing page: quick start, request flow, API and docs links      |
| `/api/:api`              | API reference for `gateway`, `admin` or `auth`; `/api` → gateway |
| `/api/:api?view=source`  | The same contract as highlighted YAML                            |
| `/api.html?api=…&view=…` | Redirect from the old static site's URLs                         |

## How it is built

- `public/openapi` is a symlink to `../openapi`, so the build always ships the current contracts
  at `openapi/*.yaml`. There is no copy to keep in sync.
- `RedocService` imports Redoc's standalone bundle (`redoc/bundles/redoc.standalone.js`) with a
  dynamic `import()`, so it is a lazy chunk downloaded the first time a reference is shown.
- Redoc has its own light and dark themes. `RedocView` switches them with the site's theme, and
  `src/styles.scss` sets Redoc's accent and fonts and the colour fixes that keep it at WCAG AA.
- `go.mod` in this directory only keeps `pages/` (and Go files inside `node_modules/`) out of the
  root Go module.

## Publishing

GitHub Pages serves the `gh-pages` branch (**Settings → Pages → Build and deployment → Deploy from
a branch**, `gh-pages`, `/ (root)`). Both workflows build with `make build/pages`, which adds a
`404.html` copy of `index.html` so deep links work.

- `.github/workflows/publish-pages.yaml` publishes the live site to the root of `gh-pages` on
  every push to `main` that touches `pages/` or `openapi/`, after the same checks that pull
  requests run. It needs approval if the `api-pages` environment requires it.
- `.github/workflows/pull-request-pages.yaml` runs those checks on pull requests and publishes a
  preview to `https://trebent.github.io/kerberos/pr-preview/pr-<number>/`, linked in a comment on
  the pull request. Each push updates the preview, and closing the pull request removes it.
  Pull requests from forks get the checks but no preview.

GitHub Pages answers every unknown path with the live site's `404.html`. For a deep link into a
preview, a short script in `src/index.html` passes the path on to the preview's own app.

## Project layout

```text
src/app/
  app.ts, app.html, app.scss   root component: skip link, header, <router-outlet>, footer
  app.config.ts                providers: router, HTTP, theme
  app.routes.ts                lazy routes, the /api redirect and the old api.html redirect
  core/                        app shell: header, footer
  pages/                       one directory per route, with page-only components under it
    landing/components/        quick-start, request-flow
    api-reference/components/  redoc-view, spec-source
  components/                  shared components used by several pages (copy-button)
  services/                    ThemeService, RedocService
  guards/                      knownApiGuard
  utils/                       the API catalogue, link helpers, the YAML highlighter
src/testing/                   test helpers: axe-core check
```

The placement rules are the same as angular-bootstrap's
[`docs/structure.md`](https://github.com/maansaake/angular-bootstrap/blob/main/docs/structure.md):
place a component by who uses it, and never import from another page's directory.

## Adding an API

1. Add the contract to `../openapi/`.
2. Add an entry to `APIS` in `src/app/utils/apis.ts`, with the docs pages that explain it.
3. Link the rendered reference from the matching page in `../docs/`.
