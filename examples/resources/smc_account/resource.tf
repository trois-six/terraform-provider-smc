# Copyright (c) HashiCorp, Inc.

terraform {
  required_providers {
    smc = {
      source = "trois-six/smc"
    }
  }
}

provider "smc" {}

resource "smc_account" "jdoe" {
  description = "some user description"
  dn          = "CN=bob,DC=company,DC=world"
  email       = "user@email.com"
  folders     = ["folder-uuid"]
  identifier  = "jdoe"
  kind        = "user"
  local_auth  = true
  name        = "Some Account name"
  permissions = ["smc"]
}

output "jdoe" {
  value = smc_account.uuid
}
