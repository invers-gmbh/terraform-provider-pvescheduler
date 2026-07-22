default:
    just --choose

build:
    go build ./...

test:
    go test ./... -v

lint:
    go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run ./...

check: lint test

docs:
    go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest generate --provider-name pvescheduler
