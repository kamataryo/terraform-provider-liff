# Terraform Provider LIFF

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.8
- [Go](https://golang.org/doc/install) >= 1.22.5

## Using the provider

```terraform
provider "liff" {
  channel_id     = "0000000000"
  channel_secret = "00112233445566778899aabbccddeeff"
}

resource "liff_app" "example" {
  description = "Your LIFF App name"
  view = {
    type = "full"
    url  = "https://example.com"
  }
  bot_prompt = "normal"
  scope      = ["profile"]
  feature = {
    qr_code = true
  }
}
```

For more information, please refer [the documentation](https://registry.terraform.io/providers/kamataryo/liff/latest/docs).

## Development
 
### Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install .
```

### Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.
