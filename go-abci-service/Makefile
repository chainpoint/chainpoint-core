DEP := $(shell command -v dep 2> /dev/null)
BUILD_TAGS?='tendermint'
BUILD_FLAGS = -ldflags "-X github.com/chainpoint/tendermint/version.GitCommit=`git rev-parse --short=8 HEAD`"

get_tools:
ifndef DEP
	@echo "Installing dep"
	go get -u -v github.com/golang/dep/cmd/dep
else
	@echo "Dep is already installed..."
endif

get_vendor_deps:
	@echo "--> Generating vendor directory via dep ensure"
	@rm -rf .vendor-new
	@dep ensure -v -vendor-only

update_vendor_deps:
	@echo "--> Running dep ensure"
	@rm -rf .vendor-new
	@dep ensure -v -update

install:
	CGO_ENABLED=1 go install -tags "$(BUILD_TAGS) cleveldb gcc" abci-service.go
