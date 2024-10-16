// Copyright (c) HashiCorp, Inc.

package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAccountDataSource(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
  "uuid": "75532250-c878-42f1-8871-bafa68e944d4",
  "description": "some user description",
  "dn": "CN=bob,DC=company,DC=world",
  "email": "user@email.com",
  "folders": [
    "folder-uuid"
  ],
  "identifier": "jdoe",
  "kind": "user",
  "localAuth": true,
  "name": "Some Account name",
  "permissions": [
    "smc"
  ]
}`))
		if err != nil {
			t.Errorf("error writing body: %s", err)
		}
	}))
	defer testServer.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: fmt.Sprintf(providerConfig, testServer.URL) + `
data "smc_account" "jdoe" {
  identifier = "jdoe"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.smc_account.jdoe", "description", "some user description"),
					resource.TestCheckResourceAttr("data.smc_account.jdoe", "dn", "CN=bob,DC=company,DC=world"),
					resource.TestCheckResourceAttr("data.smc_account.jdoe", "email", "user@email.com"),
					resource.TestCheckResourceAttr("data.smc_account.jdoe", "folders.#", "1"),
					resource.TestCheckResourceAttr("data.smc_account.jdoe", "folders.0", "folder-uuid"),
					resource.TestCheckResourceAttr("data.smc_account.jdoe", "identifier", "jdoe"),
					resource.TestCheckResourceAttr("data.smc_account.jdoe", "kind", "user"),
					resource.TestCheckResourceAttr("data.smc_account.jdoe", "local_auth", "true"),
					resource.TestCheckResourceAttr("data.smc_account.jdoe", "name", "Some Account name"),
					resource.TestCheckResourceAttr("data.smc_account.jdoe", "permissions.#", "1"),
					resource.TestCheckResourceAttr("data.smc_account.jdoe", "permissions.0", "smc"),
					resource.TestCheckResourceAttr("data.smc_account.jdoe", "uuid", "75532250-c878-42f1-8871-bafa68e944d4"),
				),
			},
		},
	})
}
