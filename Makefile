VERSION:=$(shell date --utc +%Y%m%d%H%M%S)
export VERSION
BUILD_DIR=./build
BINARY=simplek8s-init
PUBLISH_DEST=vbox1:/mnt/kubernetes/simplek8s-website-pvc-569e9642-8054-4213-a046-63af64297575/simplek8s-init
LDFLAGS=-X main.Version=${VERSION} -extldflags=-static -w -s

build: build_x86_64 build_arm64

build_x86_64:
	GOOS=linux GOARCH=amd64 \
		go build \
			-ldflags="${LDFLAGS}" \
			-o "${BUILD_DIR}/${BINARY}.${VERSION}.x86_64" \
			cmd/init/main.go
	ln -sf "${BINARY}.${VERSION}.x86_64" "${BUILD_DIR}/${BINARY}.latest.x86_64"
	ln -sf "${BINARY}.${VERSION}.x86_64" "${BUILD_DIR}/${BINARY}.x86_64"

build_arm64:
	GOOS=linux GOARCH=arm64 \
		go build \
			-ldflags="${LDFLAGS}" \
			-o "${BUILD_DIR}/${BINARY}.${VERSION}.arm64" \
			cmd/init/main.go
	ln -sf "${BINARY}.${VERSION}.arm64" "${BUILD_DIR}/${BINARY}.latest.arm64"
	ln -sf "${BINARY}.${VERSION}.arm64" "${BUILD_DIR}/${BINARY}.arm64"

build_x86_64_prod: build_x86_64
	upx --no-progress --best --ultra-brute \
		"${BUILD_DIR}/${BINARY}.${VERSION}.x86_64"

build_arm64_prod: build_arm64
	upx --no-progress --best --ultra-brute \
		"${BUILD_DIR}/${BINARY}.${VERSION}.arm64"

publish_x86_64: build_x86_64_prod
	rsync --links \
		"${BUILD_DIR}/${BINARY}.x86_64" \
		"${BUILD_DIR}/${BINARY}.${VERSION}.x86_64" \
		"${PUBLISH_DEST}/"

publish_arm64: build_arm64_prod
	rsync --links \
		"${BUILD_DIR}/${BINARY}.arm64" \
		"${BUILD_DIR}/${BINARY}.${VERSION}.arm64" \
		"${PUBLISH_DEST}/"

publish: publish_x86_64 publish_arm64

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
