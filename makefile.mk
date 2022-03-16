GO ?= go
GO_FILES = $(shell find . -name \*.go)
GOFLAGS ?= -v -trimpath
GOTESTFLAGS ?= -tags test -short -race
export GO111MODULE ?= on
GOLANGCILINTFLAGS ?= $(if $(CI),,--fix)

all: makefile.mk generate build 

.PHONY: pregenerate postgenerate generate prelint postlint lint pretest posttest test pretest posttest precobertura postcobertura prebuild postbuild build docker 

generate: 
	@/bin/echo -e \\e[1m"Generate"\\e[0m
	@$(GO) generate ./...

lint: go.sum
	@golangci-lint --version > /dev/null 2>&1;\
	if [ "$$?" != 0 ]; then\
		/bin/echo -e "\\e[31mgolangci-lint is not installed. Run go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest\\e[0m";\
		exit 1;\
	fi
	@/bin/echo -e \\e[1m"golangci-lint$(if $(CI),, --fix)"\\e[0m
	@golangci-lint run $(GOLANGCILINTFLAGS)

test: go.sum 
	@/bin/echo -e \\e[1m"Test"\\e[0m
	@$(GO) test $(GOTESTFLAGS) ./...

cover.json: 
	@gocov > /dev/null 2>&1;\
	if [ "$$?" != 2 ]; then\
		/bin/echo -e "\\e[31mgocov is not installed. Run go install github.com/axw/gocov/gocov@latest\\e[0m";\
		exit 1;\
	fi
	@/bin/echo -e \\e[1m"Test (cover)"\\e[0m
	@gocov test $(GOTESTFLAGS) ./... > $@
	@/bin/echo -e \\e[1m"Coverage report"\\e[0m
	@gocov report $@

cover.xml: cover.json 
	@gocov-xml -help > /dev/null 2>&1;\
	if [ "$$?" != 0 ]; then\
		/bin/echo -e "\\e[31mgocov-xml is not installed. Run go install github.com/AlekSi/gocov-xml@latest\\e[0m";\
		exit 1;\
	fi
	@/bin/echo -e \\e[1m"Generate Cobertura report"\\e[0m
	@gocov-xml < $< > $@

build: .bin/k8sxds 

go.sum: go.mod 
	@/bin/echo -e \\e[1m"go mod tidy"\\e[0m
	@$(GO) mod tidy

.bin/k8sxds: go.mod go.sum $(GO_FILES) 
	@/bin/echo -e \\e[1m"Build .bin/k8sxds"\\e[0m
	@cd . && $(GO) build $(GOFLAGS) -o .bin/k8sxds .

docker: 
	@/bin/echo -e \\e[1m"docker build -t k8sxds ."\\e[0m
	@docker build -t k8sxds .

