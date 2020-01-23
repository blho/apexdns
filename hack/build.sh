#!/usr/bin/env bash

REPO=${PWD#"$GOPATH/src/"}
TARGET_NAME=$(basename $PWD)
TARGET_OS=${1:-$(uname -s | awk '{ print tolower($1) }')}
TARGET_ARCH=amd64
BUILD_DATE=`date -u +'%Y-%m-%dT%H:%M:%SZ'`
GIT_COMMIT=`git rev-parse --short HEAD`
# Tag on head
VERSION=`git describe --exact-match --tags HEAD 2>>/dev/null`
[[ -z "$VERSION" ]] && VERSION="testflight-${GIT_COMMIT}"

CGO_ENABLED=0 GOOS="$TARGET_OS" GOARCH="${TARGET_ARCH}" go build -i -v -ldflags \
    "-X ${REPO}/pkg/version.version=${VERSION} -X ${REPO}/pkg/version.buildDate=${BUILD_DATE} -X ${REPO}/pkg/version.gitCommit=${GIT_COMMIT}" \
    -o ${TARGET_NAME} ${REPO}/cmd
if [[ $? -ne 0 ]]; then
    echo "Failed to build ${TARGET_NAME}"
    exit 1
fi
echo "Build ${TARGET_NAME}, OS is ${TARGET_OS}, Arch is ${TARGET_ARCH}"

if hash upx 2>/dev/null; then
    echo "Optimizing binary file size..."
    upx --fast ${TARGET_NAME}
else
    echo "Upx not found, skip binary optimization"
fi
