// Copyright (c) HashiCorp, Inc.

package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var step = 0

func TestAccAccountResource(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var err error
		switch step {
		case 0:
			w.WriteHeader(http.StatusCreated)
			_, err = w.Write([]byte(`{
  "result": {
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
  },
  "success": true
}`))
		case 1:
			w.WriteHeader(http.StatusOK)
			_, err = w.Write([]byte(`{
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
		case 2:
			w.WriteHeader(http.StatusOK)
			_, err = w.Write([]byte(`{
  "result": {
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
  },
  "success": true
}`))
		}
		if err != nil {
			t.Errorf("error writing body: %s", err)
		}
		step++
	}))
	defer testServer.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: fmt.Sprintf(providerConfig, testServer.URL) + testAccAccountResourceConfig("some user description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("smc_account.jdoe", "description", "some user description"),
					resource.TestCheckResourceAttr("smc_account.jdoe", "dn", "CN=bob,DC=company,DC=world"),
					resource.TestCheckResourceAttr("smc_account.jdoe", "email", "user@email.com"),
					resource.TestCheckResourceAttr("smc_account.jdoe", "folders.#", "1"),
					resource.TestCheckResourceAttr("smc_account.jdoe", "folders.0", "folder-uuid"),
					resource.TestCheckResourceAttr("smc_account.jdoe", "identifier", "jdoe"),
					resource.TestCheckResourceAttr("smc_account.jdoe", "kind", "user"),
					resource.TestCheckResourceAttr("smc_account.jdoe", "local_auth", "true"),
					resource.TestCheckResourceAttr("smc_account.jdoe", "name", "Some Account name"),
					resource.TestCheckResourceAttr("smc_account.jdoe", "permissions.#", "1"),
					resource.TestCheckResourceAttr("smc_account.jdoe", "permissions.0", "smc"),
					resource.TestCheckResourceAttr("smc_account.jdoe", "uuid", "75532250-c878-42f1-8871-bafa68e944d4"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "smc_account.jdoe",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: fmt.Sprintf(providerConfig, testServer.URL) + testAccAccountResourceConfig("some another description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("smc_account.jdoe", "description", "some another description"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccAccountResourceConfig(description string) string {
	return fmt.Sprintf(`
resource "smc_account" "jdoe" {
  description = %[1]q
  dn          = "CN=bob,DC=company,DC=world"
  email       = "user@email.com"
  folders     = ["folder-uuid"]
  identifier  = "jdoe"
  kind        = "user"
  local_auth  = true
  name        = "Some Account name"
  password    = "$2a$10$HM7zy3pUuoyKwnaFk4A4W.9gLQZ3BGWeJqwdlPiOJN6TayLbSQ1Na"
  permissions = ["smc"]
}
`, description)
}
