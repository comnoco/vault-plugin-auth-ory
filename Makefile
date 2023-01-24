# Metadata about this makefile and position
MKFILE_PATH := $(lastword $(MAKEFILE_LIST))
CURRENT_DIR := $(patsubst %/,%,$(dir $(realpath $(MKFILE_PATH))))

# Ensure GOPATH
GOPATH ?= $(HOME)/go

# List all our actual files, excluding vendor
GOFILES ?= $(shell go list $(TEST) | grep -v /vendor/)

# Tags specific for building
GOTAGS ?=

# Number of procs to use
GOMAXPROCS ?= 4

# Get the project metadata
GOVERSION := 1.19.2
PROJECT := $(CURRENT_DIR:$(GOPATH)/src/%=%)
OWNER := $(notdir $(patsubst %/,%,$(dir $(PROJECT))))
NAME := $(notdir $(PROJECT))
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
VERSION := $(shell awk -F\" '/Version/ { print $$2; exit }' "${CURRENT_DIR}/version/version.go")

# Current system information
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Default os-arch combination to build
XC_OS ?= darwin freebsd linux netbsd openbsd solaris windows
XC_ARCH ?= 386 amd64 arm arm64
XC_EXCLUDE ?= darwin/386 darwin/arm solaris/386 solaris/arm solaris/arm64 windows/arm openbsd/arm64

# GPG Signing key (blank by default, means no GPG signing)
GPG_KEY ?=

# List of ldflags
LD_FLAGS ?= \
	-s \
	-w \
	-X ${PROJECT}/version.Name=${NAME} \
	-X ${PROJECT}/version.GitCommit=${GIT_COMMIT}

# List of tests to run
TEST ?= ./...

# Create a cross-compile target for every os-arch pairing. This will generate
# a make target for each os/arch like "make linux/amd64" as well as generate a
# meta target (build) for compiling everything.
define make-xc-target
  $1/$2:
  ifneq (,$(findstring ${1}/${2},$(XC_EXCLUDE)))
		@printf "%s%20s %s\n" "-->" "${1}/${2}:" "${PROJECT} (excluded)"
  else
		@printf "%s%20s %s\n" "-->" "${1}/${2}:" "${PROJECT}"
		@docker run \
			--interactive \
			--rm \
			--dns="8.8.8.8" \
			--volume="${CURRENT_DIR}:/go/src/${PROJECT}" \
			--workdir="/go/src/${PROJECT}" \
			"golang:${GOVERSION}" \
			env \
				CGO_ENABLED="0" \
				GOOS="${1}" \
				GOARCH="${2}" \
				go build \
				  -a \
					-o="bin/${1}_${2}/${NAME}${3}" \
					-ldflags "${LD_FLAGS}" \
					-tags "${GOTAGS}" \
					cmd/vault-plugin-auth-ory/main.go
  endif
  .PHONY: $1/$2

  $1:: $1/$2
  .PHONY: $1

  build:: $1/$2
  .PHONY: build
endef
$(foreach goarch,$(XC_ARCH),$(foreach goos,$(XC_OS),$(eval $(call make-xc-target,$(goos),$(goarch),$(if $(findstring windows,$(goos)),.exe,)))))

# dev builds and installs the project locally.
dev:
	@echo "==> Installing ${NAME} for ${GOOS}/${GOARCH}"
	@rm -f "${GOPATH}/bin/${GOOS}_${GOARCH}/${PROJECT}/version.a" # ldflags change and go doesn't detect
	@env \
		CGO_ENABLED="0" \
		go install \
			-ldflags "${LD_FLAGS}" \
			-tags "${GOTAGS}" \
			cmd/vault-plugin-auth-ory/main.go
.PHONY: dev

# dist builds the binaries and then signs and packages them for distribution
dist:
ifndef GPG_KEY
	@echo "==> ERROR: No GPG key specified! Without a GPG key, this release cannot"
	@echo "           be signed. Set the environment variable GPG_KEY to the ID of"
	@echo "           the GPG key to continue."
	@exit 127
else
	@$(MAKE) -f "${MKFILE_PATH}" _cleanup
	@$(MAKE) -f "${MKFILE_PATH}" -j4 build
	@$(MAKE) -f "${MKFILE_PATH}" _compress _checksum _sign
endif
.PHONY: dist

# test runs the test suite.
test:
	@echo "==> Testing ${NAME}"
	@go test -timeout=30s -parallel=20 -tags="${GOTAGS}" ${GOFILES} ${TESTARGS}
.PHONY: test

# test-race runs the test suite.
test-race:
	@echo "==> Testing ${NAME} (race)"
	@go test -timeout=60s -race -tags="${GOTAGS}" ${GOFILES} ${TESTARGS}
.PHONY: test-race

# _cleanup removes any previous binaries
_cleanup:
	@rm -rf "${CURRENT_DIR}/bin/"

# _compress compresses all the binaries in bin/* as tarball and zip.
_compress:
	@mkdir -p "${CURRENT_DIR}/bin/dist"
	@for platform in $$(find ./bin -mindepth 1 -maxdepth 1 -type d); do \
		osarch=$$(basename "$$platform"); \
		if [ "$$osarch" = "dist" ]; then \
			continue; \
		fi; \
		\
		ext=""; \
		if test -z "$${osarch##*windows*}"; then \
			ext=".exe"; \
		fi; \
		cd "$$platform"; \
		tar -czf "${CURRENT_DIR}/bin/dist/${NAME}_${VERSION}_$${osarch}.tgz" "${NAME}$${ext}"; \
		zip -q "${CURRENT_DIR}/bin/dist/${NAME}_${VERSION}_$${osarch}.zip" "${NAME}$${ext}"; \
		cd - &>/dev/null; \
	done
.PHONY: _compress

# _checksum produces the checksums for the binaries in bin/dist
_checksum:
	@cd "${CURRENT_DIR}/bin/dist" && \
		shasum --algorithm 256 * > ${CURRENT_DIR}/bin/dist/${NAME}_${VERSION}_SHA256SUMS && \
		cd - &>/dev/null
.PHONY: _checksum

# _sign signs the binaries using the given GPG_KEY. This should not be called
# as a separate function.
_sign:
	@echo "==> Signing ${PROJECT} at v${VERSION}"
	@gpg \
		--default-key "${GPG_KEY}" \
		--detach-sig "${CURRENT_DIR}/bin/dist/${NAME}_${VERSION}_SHA256SUMS"
	@git commit \
		--allow-empty \
		--gpg-sign="${GPG_KEY}" \
		--message "Release v${VERSION}" \
		--quiet \
		--signoff
	@git tag \
		--annotate \
		--create-reflog \
		--local-user "${GPG_KEY}" \
		--message "Version ${VERSION}" \
		--sign \
		"v${VERSION}" main
	@echo "--> Do not forget to run:"
	@echo ""
	@echo "    git push && git push --tags"
	@echo ""
	@echo "And then upload the binaries in dist/!"
.PHONY: _sign

# starts vault in dev mode with the correct plugin dir
start:
	vault server -dev -dev-root-token-id=root -dev-plugin-dir=./bin/${GOOS}_${GOARCH} -log-level=debug

# starts vault in dev mode with the correct plugin dir and tls enabled
start-tls:
	vault server -dev -dev-root-token-id=root -dev-plugin-dir=./bin/${GOOS}_${GOARCH} -log-level=debug -dev-tls=true

# enables the auth plugin under the 'ory' path
enable:
	vault auth enable -path=ory vault-plugin-auth-ory

# writes an example config to vault
configs:
	vault write auth/ory/config @config/config.example.json

# returns the unique accessor of the auth plugin
accessor:
	vault auth list -format=json | jq -r '."ory/".accessor'

# writes an example auth to vault (update the session cookie and relations)
write:
	vault write auth/ory/login namespace=Files object=my/protected/file.txt relation=view kratos_session_cookie=ory_kratos_session=[kratos session cookie string here]
