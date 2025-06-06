all: zannotate

zannotate:
	cd cmd/zannotate && \
	go build -o zannotate && \
	cd - && \
	mv cmd/zannotate/zannotate .

clean:
	rm -f zannotate

install:
	cd cmd/zannotate && \
	go install

uninstall:
# Remove the binary from the $PATH
	@echo "Are you sure you want to uninstall zannotate? (y/n)"
	@read -r answer && \
	if [ "$$answer" = "y" ]; then \
		rm -f $$(which zannotate 2>/dev/null || true); \
		echo "zannotate has been uninstalled."; \
	else \
		echo "Uninstallation cancelled."; \
	fi
test:
	go test -v ./...

lint:
	goimports -w -local "github.com/zmap/zannotate" ./
	gofmt -s -w ./
	golangci-lint run
	@if ! command -v black >/dev/null 2>&1; then pip3 install black; fi
	black --check ./

license-check:
	./.github/workflows/check_license.sh

ci: zannotate lint test license-check

.PHONY: zannotate clean install test lint ci license-check
