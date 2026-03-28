SHELL      := /bin/bash
VERSION    := $(shell cat VERSION 2>/dev/null || echo dev)
GIT_COMMIT := $(shell git rev-parse --short HEAD)
LDFLAGS    := -ldflags "-X main.version=$(VERSION) -X main.gitCommit=$(GIT_COMMIT)"

.PHONY: build docker-build bump-patch bump-minor bump-major

build:
	go build $(LDFLAGS) -o xan-argus ./cmd/server/

docker-build:
	VERSION=$(VERSION) GIT_COMMIT=$(GIT_COMMIT) docker compose up --build

bump-patch:
	@old=$$(cat VERSION); \
	new=$$(echo $$old | awk -F. '{print $$1"."$$2"."$$3+1}'); \
	echo $$new > VERSION; \
	git add VERSION; \
	git commit -m "chore: bump version to $$new"; \
	git tag v$$new; \
	echo "Bumped to $$new — run: git push && git push --tags"

bump-minor:
	@old=$$(cat VERSION); \
	new=$$(echo $$old | awk -F. '{print $$1"."$$2+1".0"}'); \
	echo $$new > VERSION; \
	git add VERSION; \
	git commit -m "chore: bump version to $$new"; \
	git tag v$$new; \
	echo "Bumped to $$new — run: git push && git push --tags"

bump-major:
	@old=$$(cat VERSION); \
	new=$$(echo $$old | awk -F. '{print $$1+1".0.0"}'); \
	echo $$new > VERSION; \
	git add VERSION; \
	git commit -m "chore: bump version to $$new"; \
	git tag v$$new; \
	echo "Bumped to $$new — run: git push && git push --tags"
