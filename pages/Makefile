# The Kerberos GitHub Pages site: an Angular app that links to docs/ and renders the API
# reference from ../openapi/. Run these targets from this directory, or use the pages/*
# targets in the repository root Makefile.

default: check

install:
	npm ci

format:
	npx prettier --write .

format/check:
	npx prettier --check .

lint:
	npx ng lint --fix

lint/ci:
	npx ng lint

build/dev:
	npx ng build --configuration development

build/prod:
	npx ng build

# The site as GitHub Pages serves it, in dist/kerberos-pages/browser/. BASE_HREF is the path the
# site is served from: /kerberos/ for the live site, /kerberos/pr-preview/pr-<n>/ for a preview.
# GitHub Pages has no SPA fallback, so 404.html (a copy of index.html) serves deep links, and
# .nojekyll stops Jekyll from processing the files.
BASE_HREF ?= /kerberos/
PAGES_DIR := dist/kerberos-pages/browser

build/pages:
	npx ng build --base-href $(BASE_HREF)
	cp $(PAGES_DIR)/index.html $(PAGES_DIR)/404.html
	touch $(PAGES_DIR)/.nojekyll

test:
	npx ng test --watch false

serve:
	npx ng serve

# Everything CI runs, in order.
check: format/check lint/ci build/prod test

.PHONY: default install format format/check lint lint/ci build/dev build/prod build/pages test serve check
