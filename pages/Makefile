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

test:
	npx ng test --watch false

serve:
	npx ng serve

# Everything CI runs, in order.
check: format/check lint/ci build/prod test

.PHONY: default install format format/check lint lint/ci build/dev build/prod test serve check
