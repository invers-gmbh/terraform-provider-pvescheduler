default:
    just --choose

build:
    go build ./...

test:
    go test ./... -v

lint:
    go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run ./...

vuln:
    go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...

check: lint vuln test

docs:
    go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.25.0 generate --provider-name pvescheduler

docs-check: docs
    #!/usr/bin/env bash
    set -euo pipefail
    if [ -n "$(git status --porcelain docs/)" ]; then
      echo "docs/ is out of date, regenerate with: just docs"
      git status --porcelain docs/
      git diff -- docs/
      exit 1
    fi

validate-examples:
    #!/usr/bin/env bash
    set -euo pipefail
    bin="$(mktemp -d)"
    tfrc="$(mktemp)"
    trap 'rm -rf "$bin" "$tfrc"' EXIT
    go build -o "$bin/terraform-provider-pvescheduler" .
    cat > "$tfrc" <<EOF
    provider_installation {
      dev_overrides {
        "invers-gmbh/pvescheduler" = "$bin"
      }
      direct {}
    }
    EOF
    export TF_CLI_CONFIG_FILE="$tfrc"
    terraform fmt -check -recursive examples/
    for d in examples/provider examples/resources/* examples/data-sources/*; do
      terraform -chdir="$d" validate
    done
