define PRE_COMMIT_HOOK
#!/bin/sh -xe
make lint test
endef

export PRE_COMMIT_HOOK

install_hook := $(shell [ "$$INSTALL_HOOK" == "" ] && { export INSTALL_HOOK=1; \
	make .git/hooks/pre-commit; \
})

source_files := $(shell find . -name '*.go')

.PHONY: all
all: assume-role

.git/hooks/pre-commit: Makefile
	echo "$$PRE_COMMIT_HOOK" >.git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit

.PHONY: clean
clean:
	rm -f ./assume-role

.PHONY: clean-mocks
clean-mocks:
	rm mocks/*.go

assume-role: vendor $(source_files)
	go build \
		-o assume-role \
		-ldflags "-X github.com/uber/assume-role-cli/cli.date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" \
		./cli/assume-role/main.go

.PHONY:
lint: lint-license

.PHONY:
lint-license:
	@f="$$(find . -name '*.go' ! -path './vendor/*' | xargs grep -L 'Licensed under the Apache License')"; \
	if [ ! -z "$$f" ]; then \
		echo "ERROR: Files missing license header:"$$'\n'"$$f" >&2; \
		exit 1; \
	fi

.PHONY: mocks
mocks:
	go get -u github.com/golang/mock/gomock
	go get -u github.com/golang/mock/mockgen
	mockgen -package=mocks github.com/uber/assume-role-cli AWSProvider,AWSConfigProvider | sed -e 's/assume_role/assumerole/g' > mocks/aws_mocks.go

	# Add license to mocks
	set -eu; licence="$$(echo "/*"; while read -r line; do [ -z "$$line" ] && echo " *" || echo " * $$line"; done <LICENSE-short; echo " */";)"; \
	for f in mocks/aws_mocks.go; do \
		tmp=$$(mktemp); \
		echo "$$licence" | cat - "$$f" >"$$tmp"; \
		mv "$$tmp" "$$f"; \
	done

.PHONY: test
test: vendor
	go test -short -coverprofile=.coverage.out ./...

vendor:
	go mod vendor
