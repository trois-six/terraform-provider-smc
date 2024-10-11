# Copyright (c) HashiCorp, Inc.

terraform {
  required_providers {
    smc = {
      source = "trois-six/smc"
    }
  }
}

provider "smc" {}

data "smc_account" "test" {
  identifier = "jdoe"
}
