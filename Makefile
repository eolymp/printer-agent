BUILD_PREFIX?=
BUILD_SUFFIX?=
BUILD_COMMIT?=$(shell git rev-list HEAD --max-count=1 --abbrev-commit)
BUILD_TAG?=${BUILD_COMMIT}

# Dep target installs application dependencies
.PHONY: dep
dep:
	go get ./...

# Gen target generates code
.PHONY: gen
gen:
	go generate ./...

# Test target runs tests
.PHONY: test
test: TEST_PACKAGES?=./...
test: TEST_FLAGS?=-count=1
test:
	go test ${TEST_FLAGS} ${TEST_PACKAGES}

# Build target builds binary
.PHONY: build
build:
	go build -ldflags "-X main.version=${BUILD_TAG} -X main.commit=${BUILD_COMMIT} ${BUILD_LD_FLAGS}" ${BUILD_ARGS} \
		-o ${BUILD_PREFIX}eolymp-printer${BUILD_SUFFIX} ./

# Lint target runs linter
.PHONY: lint
lint:
	go vet ./...

.PHONY: ship
ship: build
	docker build --platform amd64 -t ghcr.io/eolymp/printer/printer:${BUILD_TAG} .
	docker push ghcr.io/eolymp/printer/printer:${BUILD_TAG}

	rm ${BUILD_PREFIX}eolymp-printer${BUILD_SUFFIX}
