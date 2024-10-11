// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// providerConfig is a shared configuration to combine with the actual
	// test configuration so the SMC client is properly configured.
	// It is also possible to use the SMC_ environment variables instead,
	// such as updating the Makefile and running the testing through that tool.
	providerConfig = `
terraform {
  required_providers {
    smc = {
      source  = "registry.terraform.io/trois-six/smc"
    }
  }
}

provider "smc" {
  hostname = "%s"
  api_key = "YOUR_API_KEY"
}
`
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"smc": providerserver.NewProtocol6WithError(New("test")()),
}

func providerConfigDynamicValue(config map[string]string) (tfprotov6.DynamicValue, error) {
	providerConfigTypes := map[string]tftypes.Type{
		"hostname": tftypes.String,
		"api_key":  tftypes.String,
	}
	providerConfigObjectType := tftypes.Object{AttributeTypes: providerConfigTypes}

	providerConfigObjectValue := tftypes.NewValue(providerConfigObjectType, map[string]tftypes.Value{
		"hostname": tftypes.NewValue(tftypes.String, config["hostname"]),
		"api_key":  tftypes.NewValue(tftypes.String, config["api_key"]),
	})

	value, err := tfprotov6.NewDynamicValue(providerConfigObjectType, providerConfigObjectValue)
	if err != nil {
		err = fmt.Errorf("failed to create dynamic value: %w", err)
	}

	return value, err
}

func TestAccConfigureProvider(t *testing.T) {
	providerServer, err := testAccProtoV6ProviderFactories["smc"]()
	require.NotNil(t, providerServer)
	require.NoError(t, err)

	providerConfigValue, err := providerConfigDynamicValue(map[string]string{
		"hostname": "http://localhost:8080",
		"api_key":  "YOUR_API_KEY",
	})
	require.NotNil(t, providerConfigValue)
	require.NoError(t, err)

	resp, err := providerServer.ConfigureProvider(context.Background(), &tfprotov6.ConfigureProviderRequest{
		Config: &providerConfigValue,
	})
	require.NotNil(t, resp)
	require.NoError(t, err)

	for _, diag := range resp.Diagnostics {
		t.Logf("Diagnostics: %#v", diag)
	}

	assert.Empty(t, resp.Diagnostics)
}

// TODO: Implement the test to check that the client is well using the API Key
//

// func TestAccProvider(t *testing.T) {
// 	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		t.Logf("Received request %s %s", r.Method, r.URL)
// 		headerAuth := r.Header.Get("Authorization")
// 		if want := "Bearer YOUR_API_KEY"; headerAuth != want {
// 			t.Errorf("Unexpected Authorization header %q, want %q", headerAuth, want)
// 		}
// 	}))
// 	defer testServer.Close()

// 	providerFactoryRes := New("test")()
// 	providerAdapter, ok := providerFactoryRes.(*SMCProvider)
// 	if !ok {
// 		t.Fatalf("Expected *SMCProvider, got: %T", providerFactoryRes)
// 	}

// 	var providerConfigureResponse provider.ConfigureResponse
// 	providerConfigTypes := map[string]tftypes.Type{
// 		"hostname": tftypes.String,
// 		"api_key":  tftypes.String,
// 	}
// 	providerConfigObjectType := tftypes.Object{AttributeTypes: providerConfigTypes}

// 	providerConfigObjectValue := tftypes.NewValue(providerConfigObjectType, map[string]tftypes.Value{
// 		"hostname": tftypes.NewValue(tftypes.String, testServer.URL),
// 		"api_key":  tftypes.NewValue(tftypes.String, "YOUR_API_KEY"),
// 	})

// 	providerAdapter.Configure(context.Background(), provider.ConfigureRequest{
// 		Config: tfsdk.Config{
// 			Raw: providerConfigObjectValue,
// 		},
// 	}, &providerConfigureResponse)

// 	client, ok := providerConfigureResponse.DataSourceData.(*smc.Client)
// 	if !ok {
// 		t.Fatalf("Could not get client from provider configuration, got: %T", client)
// 	}

// 	req, _ := http.NewRequest(http.MethodGet, testServer.URL, nil)
// 	_, err := client.Client.Do(req)

// 	require.NoError(t, err)
// }
