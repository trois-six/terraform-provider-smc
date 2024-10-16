# Copyright (c) HashiCorp, Inc.

terraform {
  required_providers {
    smc = {
      source = "trois-six/smc"
    }
  }
}

provider "smc" {}

data "smc_account" "jdoe" {
  identifier = "jdoe"
}

data "smc_accounts" "all_accounts" {
  identifier = "jdoe"
}