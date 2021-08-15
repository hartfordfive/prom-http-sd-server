GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BASE_NAME=prom-http-sd-server
GO_DEP_FETCH=go mod vendor 
UNAME=$(shell uname)
BUILD_DIR=build
GITHASH=$(shell sh -c 'git rev-parse --verify HEAD')
BUILDDATE=$(shell sh -c 'date +%Y-%m-%d')
VERSION=$(shell sh -c 'cat VERSION.txt')
PACKAGE_BASE=github.com/hartfordfive/${BASE_NAME}
ADD_VERSION_OS_ARCH=1

ifeq ($(UNAME), Linux)
	OS=linux
endif
ifeq ($(UNAME), Darwin)
	OS=darwin
endif
ARCH=amd64

ifeq ($(ADD_VERSION_OS_ARCH), 1)
	BINARY_NAME=$(BASE_NAME)-$(VERSION)-$(OS)-$(ARCH)
endif

all: cleanall build-all

# Cross compilation
build: cleanall
	echo "Output: ${BUILD_DIR}/${BINARY_NAME}"
	CGO_ENABLED=0 GOOS=${OS} GOARCH=${ARCH} $(GOBUILD) -ldflags "-s -w -X $(PACKAGE_BASE)/version.CommitHash=$(GITHASH) -X $(PACKAGE_BASE)/version.BuildDate=$(BUILDDATE) -X $(PACKAGE_BASE)/version.Version=$(VERSION)" -o ${BUILD_DIR}/$(BINARY_NAME) -v

build-all: cleanall
	CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} $(GOBUILD) -ldflags "-s -w -X $(PACKAGE_BASE)/version.CommitHash=$(GITHASH) -X $(PACKAGE_BASE)/version.BuildDate=$(BUILDDATE) -X ${PACKAGE_BASE}/version.Version=${VERSION}" -o ${BUILD_DIR}/$(BASE_NAME)-$(VERSION)-linux-$(ARCH) -v
	CGO_ENABLED=0 GOOS=darwin GOARCH=${ARCH} $(GOBUILD) -ldflags "-s -w -X $(PACKAGE_BASE)/version.CommitHash=$(GITHASH) -X $(PACKAGE_BASE)/version.BuildDate=$(BUILDDATE) -X ${PACKAGE_BASE}/version.Version=${VERSION}" -o ${BUILD_DIR}/$(BASE_NAME)-$(VERSION)-darwin-$(ARCH) -v

build-release: all
	tar -cvzf ${BUILD_DIR}$(BASE_NAME)-$(VERSION)-linux-$(ARCH).tar.gz ${BUILD_DIR}$(BASE_NAME)-$(VERSION)-linux-$(ARCH)
	tar -cvzf ${BUILD_DIR}$(BASE_NAME)-$(VERSION)-darwin-$(ARCH).tar.gz ${BUILD_DIR}$(BASE_NAME)-$(VERSION)-darwin-$(ARCH)

build-debug:
	CGO_ENABLED=0 GOOS=${OS} GOARCH=amd64 $(GOBUILD) -ldflags "-X $(PACKAGE_BASE)/version.CommitHash=$(GITHASH) -X $(PACKAGE_BASE)/version.BuildDate=${BUILDDATE} -X ${PACKAGE_BASE}/version.Version=${VERSION}" -o ${BUILD_DIR}/$(BASE_NAME)-$(VERSION)-${OS}-$(ARCH)-debug -v

test: 
	$(GOTEST) -v ./...

clean: 
	$(GOCLEAN)
	rm -rf ${BUILD_DIR}*

cleanplugins:
	$(GOCLEAN)

cleanall: clean cleanplugins

run:
	mkdir -p ${BUILD_DIR}/tmp/
	$(GOBUILD) -a -o ${BUILD_DIR}/$(BINARY_NAME) -v ./...
	./${BUILD_DIR}/$(BINARY_NAME)

deps:
	$(GO_DEP_FETCH)

