# Kerberos Pages — Agent Instructions

This directory is the Angular app published to GitHub Pages. It follows the conventions of
[angular-bootstrap](https://github.com/maansaake/angular-bootstrap); the rules below are the ones
that matter here. The layout and routes are described in [`README.md`](README.md).

## Content

- The site is mainly the API reference. Do not explain Kerberos features here: link to the page in
  `../docs/` that does (`docsUrl()` in `src/app/utils/links.ts`).
- The request-flow overview mirrors "Request lifecycle" in `../docs/routing.md`. If one changes,
  change the other.
- Use British English, as `../docs/` does.

## Angular rules

- Standalone components; do not set `standalone: true` or `changeDetection: OnPush` (both default).
- Use signals, `input()`, `computed()` and the `@if`/`@for` control flow. No `@HostBinding` or
  `@HostListener`; use the `host` object.
- Lazy-load every page from `app.routes.ts`, the only file that imports a page.
- Prefer the `@Service` decorator for singleton services.
- Use only `--mat-sys-*` tokens for colours, except in the always-dark `.code-panel` and the
  always-light Redoc panel.

## Accessibility

- It MUST pass all axe checks and meet WCAG 2.2 AA, in both light and dark mode.
- Every page and component spec calls `expectNoAxeViolations` from `src/testing/axe.ts`. jsdom
  cannot check colour contrast, so check it in a browser.

## Verification

Run in order, in `pages/`: `make format`, `make build/prod`, `make lint`, `make test`
(`make check` runs the CI sequence). All must pass before a change is complete.
