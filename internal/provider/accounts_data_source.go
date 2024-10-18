// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/trois-six/smc"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &AccountsDataSource{}

func NewAccountsDataSource() datasource.DataSource {
	return &AccountsDataSource{}
}

// AccountsDataSource defines the data source implementation.
type AccountsDataSource struct {
	client *smc.ClientWithResponses
}

// AccountsDataSourceModel describes the data source data model.
type AccountsDataSourceModel struct {
	Accounts []AccountDataSourceModel `tfsdk:"accounts"`
}

func (d *AccountsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_accounts"
}

func (d *AccountsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches all the accounts.",
		Attributes: map[string]schema.Attribute{
			"accounts": schema.ListNestedAttribute{
				Description: "List of accounts",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: getAccountDataSourceSchemaAttributes(),
				},
			},
		},
	}
}

func (d *AccountsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*smc.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *smc.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *AccountsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccountsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	respAPI, err := d.client.GetApiAccountsWithResponse(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading SMC Accounts",
			"Could not read SMC accounts: "+err.Error(),
		)
		return
	}

	if respAPI.StatusCode() != http.StatusOK || respAPI.JSON200 == nil {
		resp.Diagnostics.AddError(
			"HTTP Error Reading SMC Accounts",
			"HTTP status code "+respAPI.Status()+" returned while reading SMC accounts",
		)
		return
	}

	if respAPI.JSON200.Result == nil || len(*respAPI.JSON200.Result) == 0 || respAPI.JSON200.Success == nil || !*respAPI.JSON200.Success {
		resp.Diagnostics.AddError(
			"No results Reading SMC Accounts",
			"No results returned while reading SMC accounts",
		)
		return
	}

	accounts := make([]AccountDataSourceModel, len(*respAPI.JSON200.Result))
	for idx, item := range *respAPI.JSON200.Result {
		readAccountDataSourceModel(&accounts[idx], &item)
	}

	data.Accounts = accounts

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}
