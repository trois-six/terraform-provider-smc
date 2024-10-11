// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/trois-six/smc"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &AccountDataSource{}

func NewAccountDataSource() datasource.DataSource {
	return &AccountDataSource{}
}

// AccountDataSource defines the data source implementation.
type AccountDataSource struct {
	client *smc.ClientWithResponses
}

// AccountDataSourceModel describes the data source data model.
type AccountDataSourceModel struct {
	UUID        types.String   `tfsdk:"uuid"`
	Description types.String   `tfsdk:"description"`
	DN          types.String   `tfsdk:"dn"`
	Email       types.String   `tfsdk:"email"`
	Folders     []types.String `tfsdk:"folders"`
	Identifier  types.String   `tfsdk:"identifier"`
	Kind        types.String   `tfsdk:"kind"`
	LocalAuth   types.Bool     `tfsdk:"local_auth"`
	Name        types.String   `tfsdk:"name"`
	Permissions []types.String `tfsdk:"permissions"`
}

func (d *AccountDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (d *AccountDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches an account based on an identifier.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Account uuid",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Account description",
				Computed:            true,
			},
			"dn": schema.StringAttribute{
				MarkdownDescription: "Account DN",
				Computed:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Account email",
				Computed:            true,
			},
			"folders": schema.ListAttribute{
				MarkdownDescription: "Account folders",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"identifier": schema.StringAttribute{
				MarkdownDescription: "Account identifier",
				Required:            true,
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "Account kind",
				Computed:            true,
			},
			"local_auth": schema.BoolAttribute{
				MarkdownDescription: "Account local authentication",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Account name",
				Computed:            true,
			},
			"permissions": schema.ListAttribute{
				MarkdownDescription: "Account permissions",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *AccountDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccountDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	account, err := d.client.GetApiAccountsUuidWithResponse(ctx, data.Identifier.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading SMC Account",
			"Could not read SMC account identifier "+data.Identifier.ValueString()+": "+err.Error(),
		)
		return
	}

	if account.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"HTTP Error Reading SMC Account",
			"HTTP status code "+account.Status()+" returned for SMC account identifier "+data.Identifier.ValueString(),
		)
		return
	}

	if account.JSON200 == nil {
		if !data.Identifier.IsNull() {
			resp.Diagnostics.AddError(
				"No results Reading SMC Account",
				"No results returned for given identifier: "+data.Identifier.ValueString(),
			)
		}
		return
	}

	item := account.JSON200
	data.UUID = types.StringValue(item.Uuid)
	data.Description = types.StringPointerValue(item.Description)
	data.DN = types.StringPointerValue(item.Dn)
	data.Email = types.StringPointerValue(item.Email)

	if item.Folders != nil {
		for _, folder := range *item.Folders {
			data.Folders = append(data.Folders, types.StringValue(folder))
		}
	}

	data.Identifier = types.StringPointerValue(item.Identifier)
	data.Kind = types.StringPointerValue(item.Kind)
	data.LocalAuth = types.BoolPointerValue(item.LocalAuth)
	data.Name = types.StringPointerValue(item.Name)

	if item.Permissions != nil {
		for _, permission := range *item.Permissions {
			data.Permissions = append(data.Permissions, types.StringValue(string(permission)))
		}
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}
