MODULE = github.com/sulmone/xds-control-server

GO_BUILD_VARS = \
	github.com/projectcontour/contour/internal/build.Version=${BUILD_VERSION} \
	github.com/projectcontour/contour/internal/build.Sha=${BUILD_SHA} \
	github.com/projectcontour/contour/internal/build.Branch=${BUILD_BRANCH}

GO_LDFLAGS := -s -w $(patsubst %,-X %, $(GO_BUILD_VARS))

ifndef $(GOPATH)
	GOPATH=$(shell go env GOPATH)
	export GOPATH
endif

install: ## Build and install the binary
	go build -o $(GOPATH)/bin/xds-control-server -mod=readonly -v -ldflags="$(GO_LDFLAGS)" $(MODULE)/cmd/server
