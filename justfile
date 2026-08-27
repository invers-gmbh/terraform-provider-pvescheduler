default:
    just --choose

build:
    go build ./...

test:
    go test ./... -v

testacc:
    TF_ACC=1 go test ./... -v -timeout 20m

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

# Verify every commit in RANGE carries a Signed-off-by trailer for its author (DCO).
dco range="trunk..HEAD":
    #!/usr/bin/env bash
    set -euo pipefail
    range="{{ range }}"
    fail=0
    for sha in $(git rev-list --no-merges "$range"); do
      name="$(git show -s --format='%an' "$sha")"
      email="$(git show -s --format='%ae' "$sha")"
      signoffs="$(git show -s --format='%(trailers:key=Signed-off-by,valueonly)' "$sha")"
      if grep -qiF "<$email>" <<<"$signoffs"; then
        continue
      fi
      printf '%s %s\n' "$(git rev-parse --short "$sha")" "$(git show -s --format='%s' "$sha")" >&2
      printf '    missing: Signed-off-by: %s <%s>\n' "$name" "$email" >&2
      fail=1
    done
    if [ "$fail" -ne 0 ]; then
      echo >&2
      echo "Every commit needs a Developer Certificate of Origin sign-off." >&2
      echo "Sign new commits with 'git commit -s'." >&2
      exit 1
    fi
    echo "DCO: every commit in $range carries a matching Signed-off-by trailer"
