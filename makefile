PREFIX :=.

# Initialize workspace if it's empty
ifeq ($(WORKSPACE),)
	WORKSPACE := $(shell pwd)/../../../../
endif

export GO_LDFLAGS=-ldflags "-s -w -X github.com/aws/aws-xray-daemon/pkg/cfg.Version=${VERSION}"

# Initialize BGO_SPACE
export BGO_SPACE=$(shell pwd)
path := $(BGO_SPACE):$(WORKSPACE)

build: create-folder copy-file build-mac build-linux build-linux-arm64 build-windows

packaging: zip-linux zip-osx zip-win package-rpm package-deb

release: build test packaging clean-folder

.PHONY: create-folder
create-folder:
	mkdir -p build/xray
	mkdir -p build/dist

.PHONY: copy-file
copy-file:
	cp pkg/cfg.yaml build/xray/
	cp $(BGO_SPACE)/LICENSE build/xray
	cp $(BGO_SPACE)/THIRD-PARTY-LICENSES.txt build/xray

.PHONY: build-mac
build-mac:
	@echo "Build for MAC amd64"
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(GO_LDFLAGS) -o $(BGO_SPACE)/build/xray-mac-amd64/xray ${PREFIX}/cmd/tracing/daemon.go ${PREFIX}/cmd/tracing/tracing.go

.PHONY: build-linux
build-linux:
	@echo "Build for Linux amd64"
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(GO_LDFLAGS) -o $(BGO_SPACE)/build/xray-linux-amd64/xray ${PREFIX}/cmd/tracing/daemon.go ${PREFIX}/cmd/tracing/tracing.go

.PHONY: build-linux-arm64
build-linux-arm64:
	@echo "Build for Linux arm64"
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(GO_LDFLAGS) -o $(BGO_SPACE)/build/xray-linux-arm64/xray ${PREFIX}/cmd/tracing/daemon.go ${PREFIX}/cmd/tracing/tracing.go

.PHONY: build-windows
build-windows:
	@echo "Build for Windows amd64"
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(GO_LDFLAGS) -o $(BGO_SPACE)/build/xray-windows-amd64/xray_service.exe ${PREFIX}/cmd/tracing/daemon.go ${PREFIX}/cmd/tracing/tracing_windows.go
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(GO_LDFLAGS) -o $(BGO_SPACE)/build/xray-windows-amd64/xray.exe ${PREFIX}/cmd/tracing/daemon.go ${PREFIX}/cmd/tracing/tracing.go

.PHONY: build-docker
build-docker:
	docker build -t amazon/aws-xray-daemon:$VERSION .

.PHONY: push-docker
push-docker:
	docker push amazon/aws-xray-daemon:$VERSION

.PHONY: zip-linux
zip-linux:
	$(BGO_SPACE)/Tool/src/packaging/linux/build_zip_linux.sh

.PHONY: zip-osx
zip-osx:
	$(BGO_SPACE)/Tool/src/packaging/osx/build_zip_osx.sh

.PHONY: zip-win
zip-win:
	$(BGO_SPACE)/Tool/src/packaging/windows/build_zip_win.sh

.PHONY: package-deb
package-deb:
	$(BGO_SPACE)/Tool/src/packaging/debian/build_deb_linux.sh

.PHONY: package-rpm
package-rpm:
	-$(BGO_SPACE)/Tool/src/packaging/linux/build_rpm_linux.sh

.PHONY: test
test:
	@echo "Testing daemon"
	go test -cover ./...

vet:
	go vet ./...

lint:
	golint ${SDK_BASE_FOLDERS}...

fmt:
	go fmt pkg/...

.PHONY: clean-folder
clean-folder:
	cd build && \
	find . ! -name "xray" ! -name "." -type d -exec rm -rf {} + || true
