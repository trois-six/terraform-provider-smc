# Terraform Provider SMC

This is the Stormshield SMC Terraform Provider. It allows to manage your Stormshield SMC configuration using Terraform.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22

## Building The Provider

1. Clone the repository
2. Enter the repository directory
3. Build the provider using the Go `install` command:

```shell
go install
```

## Using the provider

- Set the `hostname` and `api_key` in the provider block:

```hcl
provider "smc" {
  hostname = "https://smc.example.com"
  api_key  = "your_api_key"
}
```

- Use a datasource or a resource

```hcl
data "smc_account" "test" {
  identifier = "jdoe"
}
```

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests do not create real resources, they are using mocks.

```shell
make testacc
```
