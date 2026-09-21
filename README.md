# Terraform Provider for AppSignal

Manage [AppSignal](https://appsignal.com) apps, log sources, log views and log
triggers with Terraform.

## Usage

```terraform
terraform {
  required_providers {
    appsignal = {
      source = "RinseV/appsignal"
    }
  }
}

provider "appsignal" {
  token        = "abcdef..." # or APPSIGNAL_TOKEN
  organization = "my-org"    # or APPSIGNAL_ORGANIZATION
}
```

The `organization` is the slug from the AppSignal URL: for
`https://appsignal.com/my-org` the slug is `my-org`.

See the [docs](docs/) for all resources and data sources, and
[examples](examples/) for usage examples.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.27

## Development

- `make install` builds the provider into `$GOPATH/bin`
- `make generate` regenerates the docs
- `make test` runs the unit tests

### Acceptance tests

`make testacc` runs against the real AppSignal API and needs a token. Copy
`.env.example` to `.env` and fill in a personal AppSignal API token. `.env` is
gitignored.

```shell
cp .env.example .env
$EDITOR .env
make testacc
```
