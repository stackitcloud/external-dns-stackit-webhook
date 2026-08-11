GOLANGCI_VERSION = 2.12.2
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
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/main/install.sh | bash -s -- -b bin v$(GOLANGCI_VERSION)
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

# ==============================================================================
# E2E Local Testing
# ==============================================================================

E2E_TMP_DIR = tests/e2e-tmp

.PHONY: build-linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(shell go env GOARCH) go build -ldflags "-s -w" -o ./external-dns-stackit-webhook -v cmd/webhook/main.go

.PHONY: docker-build-e2e
docker-build-e2e: build-linux
	docker build -t stackitcloud/external-dns-stackit-webhook:e2e -f Dockerfile .
	rm ./external-dns-stackit-webhook # Clean up the binary after build

# Run this to test the webhook locally
# make test-e2e-local \
    PROJECT_ID="your-project-id" \
    ZONE_NAME="your.test.zone.cloud" \
    AUTH_KEY_PATH="/absolute/path/to/your/sa.json"
.PHONY: test-e2e-local
test-e2e-local: docker-build-e2e
	@if [ -z "$(PROJECT_ID)" ] || [ -z "$(ZONE_NAME)" ] || [ -z "$(AUTH_KEY_PATH)" ]; then \
		echo "Error: Missing PROJECT_ID, ZONE_NAME, or AUTH_KEY_PATH environment variables."; \
		exit 1; \
	fi
	@echo "=> Creating Kind cluster..."
	kind create cluster --name stackit-e2e || true
	@echo "=> Loading image into Kind..."
	kind load docker-image stackitcloud/external-dns-stackit-webhook:e2e --name stackit-e2e
	@echo "=> Setting up STACKIT credentials..."
	kubectl create secret generic external-dns-stackit-webhook \
		--from-file=sa.json=$(AUTH_KEY_PATH) \
		--dry-run=client -o yaml | kubectl apply -f -
	@echo "=> Preparing test manifests..."
	rm -rf $(E2E_TMP_DIR)
	cp -r tests/e2e $(E2E_TMP_DIR)
	find $(E2E_TMP_DIR) -type f -name "*.yaml" -exec sed -i.bak "s/\$${PROJECT_ID}/$(PROJECT_ID)/g" {} +
	find $(E2E_TMP_DIR) -type f -name "*.yaml" -exec sed -i.bak "s/\$${ZONE_NAME}/$(ZONE_NAME)/g" {} +
	find $(E2E_TMP_DIR) -type f -name "*.bak" -delete

	@echo "=> Deploying ExternalDNS and Webhook..."
	kubectl apply -f $(E2E_TMP_DIR)/deploy/external-dns.yaml
	kubectl wait --for=condition=available --timeout=60s deployment/external-dns

	@echo "=> Running Kuttl Tests..."
	cd $(E2E_TMP_DIR) && \
	kubectl kuttl test; \
	RET=$$?; \
	echo "=> Cleaning up local test environment..."; \
	kind delete cluster --name stackit-e2e; \
	cd .. && rm -rf $(E2E_TMP_DIR); \
	exit $$RET

.PHONY: clean-e2e-local
clean-e2e-local:
	kind delete cluster --name stackit-e2e