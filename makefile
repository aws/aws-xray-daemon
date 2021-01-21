PREFIX :=.

# Initialize workspace if it's empty
ifeq ($(WORKSPACE),)
	WORKSPACE := $(shell pwd)/../../../../
endif

export GO_LDFLAGS=-ldflags "-s -w -X github.com/aws/aws-xray-daemon/pkg/cfg.Version=${VERSION}"

# Initialize BGO_SPACE
export BGO_SPACE=$(shell pwd)
path := $(BGO_SPACE):$(WORKSPACE)

build: create-folder copy-file build-mac build-linux-amd64 build-linux-arm64 build-windows

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

.PHONY: build-linux-amd64
build-linux-amd64:
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
	$(BGO_SPACE)/Tool/src/packaging/debian/build_deb_linux.sh amd64
	$(BGO_SPACE)/Tool/src/packaging/debian/build_deb_linux.sh arm64

.PHONY: package-rpm
package-rpm:
	$(BGO_SPACE)/Tool/src/packaging/linux/build_rpm_linux.sh amd64
	$(BGO_SPACE)/Tool/src/packaging/linux/build_rpm_linux.sh arm64

# This will be removed in the next major version release
build-package-legacy:
	@echo "===Building legacy artifacts with older names==="
	cd $(BGO_SPACE)

	@echo "Building and packaging legacy artifacts for Linux"
	cp $(BGO_SPACE)/build/dist/aws-xray-daemon-linux-amd64-${VERSION}.zip $(BGO_SPACE)/build/dist/aws-xray-daemon-linux-${VERSION}.zip
	cp $(BGO_SPACE)/build/dist/aws-xray-daemon-linux-amd64-${VERSION}.rpm $(BGO_SPACE)/build/dist/aws-xray-daemon-${VERSION}.rpm
	cp $(BGO_SPACE)/build/dist/aws-xray-daemon-linux-amd64-${VERSION}.deb $(BGO_SPACE)/build/dist/aws-xray-daemon-${VERSION}.deb
	cp $(BGO_SPACE)/build/dist/aws-xray-daemon-linux-arm64-${VERSION}.rpm $(BGO_SPACE)/build/dist/aws-xray-daemon-arm64-${VERSION}.rpm
	cp $(BGO_SPACE)/build/dist/aws-xray-daemon-linux-arm64-${VERSION}.deb $(BGO_SPACE)/build/dist/aws-xray-daemon-arm64-${VERSION}.deb

	@echo "Building and packaging legacy artifacts for MacOS"
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o $(BGO_SPACE)/build/xray-mac-legacy/xray_mac ${PREFIX}/cmd/tracing/daemon.go ${PREFIX}/cmd/tracing/tracing.go
	zip $(BGO_SPACE)/build/dist/aws-xray-daemon-macos-${VERSION}.zip $(BGO_SPACE)/build/xray-mac-legacy/xray_mac $(BGO_SPACE)/build/xray/cfg.yaml $(BGO_SPACE)/build/xray/LICENSE $(BGO_SPACE)/build/xray/THIRD-PARTY-LICENSES.txt

	@echo "Building and packaging legacy artifacts for Windows"
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o $(BGO_SPACE)/build/xray-win-legacy/xray.exe ${PREFIX}/cmd/tracing/daemon.go ${PREFIX}/cmd/tracing/tracing_windows.go
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o $(BGO_SPACE)/build/xray-win-legacy/xray_windows.exe ${PREFIX}/cmd/tracing/daemon.go ${PREFIX}/cmd/tracing/tracing.go
	zip $(BGO_SPACE)/build/dist/aws-xray-daemon-windows-service-${VERSION}.zip $(BGO_SPACE)/build/xray-win-legacy/xray.exe $(BGO_SPACE)/build/xray/cfg.yaml $(BGO_SPACE)/build/xray/LICENSE $(BGO_SPACE)/build/xray/THIRD-PARTY-LICENSES.txt
	zip $(BGO_SPACE)/build/dist/aws-xray-daemon-windows-process-${VERSION}.zip $(BGO_SPACE)/build/xray-win-legacy/xray_windows.exe $(BGO_SPACE)/build/xray/cfg.yaml $(BGO_SPACE)/build/xray/LICENSE $(BGO_SPACE)/build/xray/THIRD-PARTY-LICENSES.txt

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
