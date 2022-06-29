.DEFAULT_GOAL := build

VERSION:=$(shell date --utc +%Y%m%d%H%M%S)
export VERSION
BUILD_DIR=./build
BINARY=simplek8s-init
PUBLISH_DEST=vbox1:/mnt/kubernetes/simplek8s-website-pvc-569e9642-8054-4213-a046-63af64297575/simplek8s-init
LDFLAGS=-X main.Version=${VERSION} -extldflags=-static -w -s

build_mkdir_dir:
	mkdir -p "${BUILD_DIR}/dist/archive" "${BUILD_DIR}/dist/rolling"

build_x86-64: build_mkdir_dir
	GOOS=linux GOARCH=amd64 \
		go build \
			-ldflags="${LDFLAGS}" \
			-o "${BUILD_DIR}/dist/archive/${BINARY}.${VERSION}.x86-64" \
			cmd/init/main.go
	ln -sf "../archive/${BINARY}.${VERSION}.x86-64" "${BUILD_DIR}/dist/rolling/${BINARY}.latest.x86-64"

build_arm64: build_mkdir_dir
	GOOS=linux GOARCH=arm64 \
		go build \
			-ldflags="${LDFLAGS}" \
			-o "${BUILD_DIR}/dist/archive/${BINARY}.${VERSION}.arm64" \
			cmd/init/main.go
	ln -sf "../archive/${BINARY}.${VERSION}.arm64" "${BUILD_DIR}/dist/rolling/${BINARY}.latest.arm64"

build_x86-64_prod: build_x86-64
	upx --no-progress --best --ultra-brute \
		"${BUILD_DIR}/dist/archive/${BINARY}.${VERSION}.x86-64"

build_arm64_prod: build_arm64
	upx --no-progress --best --ultra-brute \
		"${BUILD_DIR}/dist/archive/${BINARY}.${VERSION}.arm64"

build: build_x86-64 build_arm64

all: build

publish: build_x86-64_prod build_arm64_prod
	rsync \
		--links --recursive \
		"${BUILD_DIR}/dist/" \
		"${PUBLISH_DEST}/"

test:
	go test -v ./... -cover

coverage.out:
	go test \
		-covermode=count \
		-coverprofile ${BUILD_DIR}/coverage.txt \
		$(shell go list ./... | grep -v /vendor/ | tr '\n' ' ')

cover: coverage.out
	go tool cover -func=${BUILD_DIR}/coverage.txt

cover-html: coverage.out
	go tool cover -html=${BUILD_DIR}/coverage.txt

clean:
	go clean
	rm -rf "${BUILD_DIR}"
