ifeq ($(OS),Windows_NT)
	OS = win
	DLL_FORMAT = dll
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		OS = linux
		DLL_FORMAT = so
	endif
	ifeq ($(UNAME_S),Darwin)
		OS = mac
		DLL_FORMAT = dylib
	endif
endif


comma := ,
empty := 
space := $(empty) $(empty)
GO ?= go
GO_TAGS ?=
GO_FILES = $(shell find . -name \*.go)
GOBUILDFLAGS ?= -v -trimpath$(if $(GO_TAGS), -tags $(GO_TAGS),)
GOTESTFLAGS ?= -tags "test$(if $(GO_TAGS),$(comma)$(GO_TAGS),)" -short -race
export GO111MODULE ?= on
GOLANGCILINTFLAGS ?= --build-tags "test$(if $(GO_TAGS),$(comma)$(GO_TAGS),)" $(if $(CI),,--fix)
BUFFORMATFLAGS ?= $(if $(CI),-d --exit-code,-w)

ifeq ($(GO_TAGS), test)
	GO_TAGS=$(error Do not put test in GO_TAGS)
endif
ifneq ($(strip $(filter test,$(subst $(comma),$(space),$(GO_TAGS)))),)
	GO_TAGS=$(error Do not put test in GO_TAGS)
endif

all: makefile.mk generate build 

.PHONY: pregenerate postgenerate generate prelint postlint lint test pretest posttest pretest posttest precobertura postcobertura prebuild postbuild build docker 

generate: 
	@printf \\e[1m"Generate"\\e[0m\\n
	@$(GO) generate ./...

.bin/golangci-lint: .custom-gcl.yml $(command -v golangci-lint)
	@printf \\e[1m"golangci-lint custom"\\e[0m\\n
	@golangci-lint custom -v

lint: .bin/golangci-lint go.sum
	@printf \\e[1m".bin/golangci-lint$(if $(CI),, --fix)"\\e[0m\\n
	@.bin/golangci-lint run $(GOLANGCILINTFLAGS)

test: go.sum 
	@printf \\e[1m"Test"\\e[0m\\n
	@$(GO) test $(GOTESTFLAGS) ./...

cover.profile: 
	@gotestsum help > /dev/null 2>&1;\
	if [ "$$?" != 0 ]; then\
		printf "\\e[31mgotestsum is not installed. Run go install gotest.tools/gotestsum@latest\\e[0m\\n";\
		exit 1;\
	fi
	@printf \\e[1m"Test (cover)"\\e[0m\\n
	@MAIN_MODULE=$$(go list -m);\
	GO_PACKAGES=$$(go list $$MAIN_MODULE/... | grep -v -e "/gen" -e "/mock" -e "/wire" -e "/di");\
	GO_PACKAGES_COMMA_SEP=$$(echo $$GO_PACKAGES | sed 's/ /,/g');\
	gotestsum --junitfile=report.xml -- $(GOTESTFLAGS) -coverpkg=$$GO_PACKAGES_COMMA_SEP --coverprofile=$@ ./...

cover.json: cover.profile 
	@gocov > /dev/null 2>&1;\
	if [ "$$?" != 2 ]; then\
		printf "\\e[31mgocov is not installed. Run go install github.com/axw/gocov/gocov@latest\\e[0m\\n";\
		exit 1;\
	fi
	@printf \\e[1m"Generate coverage report"\\e[0m\\n
	@gocov convert cover.profile > $@
	@printf \\e[1m"Coverage report"\\e[0m\\n
	@gocov report $@

cover.xml: cover.json 
	@gocov-xml -help > /dev/null 2>&1;\
	if [ "$$?" != 0 ]; then\
		printf "\\e[31mgocov-xml is not installed. Run go install github.com/AlekSi/gocov-xml@latest\\e[0m\\n";\
		exit 1;\
	fi
	@printf \\e[1m"Generate Cobertura report"\\e[0m\\n
	@gocov-xml < $< > $@

build: .bin/xds 

go.sum: go.mod 
	@printf \\e[1m"go mod tidy"\\e[0m\\n
	@$(GO) mod tidy

.bin/xds: go.mod go.sum $(GO_FILES) 
	@printf \\e[1m"Build .bin/xds"\\e[0m\\n
	@cd . && $(GO) build $(GOBUILDFLAGS) -o .bin/xds .

docker: 
	@printf \\e[1m"docker build -t ghcr.io/wongnai/xds ."\\e[0m\\n
	@docker build -t ghcr.io/wongnai/xds .
