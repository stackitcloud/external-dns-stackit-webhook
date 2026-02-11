GOLANGCI_VERSION = 2.8.0
LICENCES_IGNORE_LIST = $(shell cat licenses/licenses-ignore-list.txt)

VERSION ?= 0.0.1
IMAGE_TAG_BASE ?= stackitcloud/external-dns-stackit-webhook
IMG ?= $(IMAGE_TAG_BASE):$(VERSION)

BUILD_VERSION ?= $(shell git branch --show-current)
BUILD_COMMIT ?= $(shell git rev-parse --short HEAD)
BUILD_TIMESTAMP ?= $(shell date -u '+%Y-%m-%d %H:%M:%S')

PWD = $(shell pwd)
export PATH := $(PWD)/bin:$(PATH)

download:
	go mod download

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags "-s -w" -o ./bin/external-dns-stackit-webhook -v cmd/webhook/main.go

.PHONY: docker-build
docker-build:
	docker build -t $(IMG) -f Dockerfile .

test:
	go test -race ./...

mocks:
	go generate ./...

GOLANGCI_LINT = bin/golangci-lint-$(GOLANGCI_VERSION)
$(GOLANGCI_LINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | bash -s -- -b bin v$(GOLANGCI_VERSION)
	@mv bin/golangci-lint "$(@)"

lint: $(GOLANGCI_LINT) download
	$(GOLANGCI_LINT) run -v

out:
	@mkdir -pv "$(@)"

reports:
	@mkdir -pv "$(@)/licenses"

coverage: out
	go test -race ./... -coverprofile=out/cover.out

html-coverage: out/report.json
	go tool cover -html=out/cover.out

.PHONY: out/report.json
out/report.json:
	go test -race ./... -coverprofile=out/cover.out --json | tee "$(@)"

run:
	go run cmd/webhook/main.go

.PHONY: clean
clean:
	rm -rf ./bin
	rm -rf ./out

GO_RELEASER = bin/goreleaser
$(GO_RELEASER):
	GOBIN=$(PWD)/bin go install github.com/goreleaser/goreleaser@latest

.PHONY: release-check
release-check: $(GO_RELEASER) ## Check if the release will work
	GITHUB_SERVER_URL=github.com GITHUB_REPOSITORY=stackitcloud/external-dns-stackit-webhook REGISTRY=$(REGISTRY) IMAGE_NAME=$(IMAGE_NAME) $(GO_RELEASER) release --snapshot --clean --skip-publish

GO_LICENSES = bin/go-licenses
$(GO_LICENSES):
	GOBIN=$(PWD)/bin go install github.com/google/go-licenses

.PHONY: license-check
license-check: $(GO_LICENSES) reports ## Check licenses against code.
	$(GO_LICENSES) check --include_tests --ignore $(LICENCES_IGNORE_LIST) ./...

.PHONY: license-report
license-report: $(GO_LICENSES) reports ## Create licenses report against code.
	$(GO_LICENSES) report --include_tests --ignore $(LICENCES_IGNORE_LIST) ./... > ./reports/licenses/licenses-list.csv
