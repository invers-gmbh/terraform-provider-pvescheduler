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
    go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest generate --provider-name pvescheduler
