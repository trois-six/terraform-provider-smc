// Copyright (c) HashiCorp, Inc.

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/trois-six/smc"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &AccountResource{}
var _ resource.ResourceWithConfigure = &AccountResource{}
var _ resource.ResourceWithImportState = &AccountResource{}

func NewAccountResource() resource.Resource {
	return &AccountResource{}
}

// AccountResource defines the resource implementation.
type AccountResource struct {
	client *smc.ClientWithResponses
}

// AccountResourceModel describes the resource data model.
type AccountResourceModel struct {
	Description types.String `tfsdk:"description" json:"description"`
	DN          types.String `tfsdk:"dn" json:"dn"`
	Email       types.String `tfsdk:"email" json:"email"`
	Folders     types.List   `tfsdk:"folders" json:"folders"`
	Identifier  types.String `tfsdk:"identifier" json:"identifier"`
	Kind        types.String `tfsdk:"kind" json:"kind"`
	LastUpdated types.String `tfsdk:"last_updated" json:"last_updated"`
	LocalAuth   types.Bool   `tfsdk:"local_auth" json:"local_auth"`
	Name        types.String `tfsdk:"name" json:"name"`
	Password    types.String `tfsdk:"password" json:"password"`
	Permissions types.List   `tfsdk:"permissions" json:"permissions"`
	UUID        types.String `tfsdk:"uuid" json:"uuid"`
}

func (r *AccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (r *AccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Manage the account resource.",

		Attributes: map[string]schema.Attribute{
			"description": schema.StringAttribute{
				MarkdownDescription: "The user's description",
				Optional:            true,
			},
			"dn": schema.StringAttribute{
				MarkdownDescription: "user's DN",
				Optional:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Account's email",
				Optional:            true,
			},
			"folders": schema.ListAttribute{
				MarkdownDescription: "Array of folder rights",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"identifier": schema.StringAttribute{
				MarkdownDescription: "the account's id (different from login if the user is member of a group)",
				Optional:            true,
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "Type of account (user or group)",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"user",
						"group",
					),
				},
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"local_auth": schema.BoolAttribute{
				MarkdownDescription: "does the user can use the local authentication",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "the user's name",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "User password",
				Optional:            true,
				Sensitive:           true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^\$2[ayb]\$.{56}$`),
						"Password must be a valid bcrypt hash",
					),
				},
			},
			"permissions": schema.ListAttribute{
				MarkdownDescription: "Array of access rights",
				Optional:            true,
				ElementType:         types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf(
							"smc",
							"sns",
							"console",
							"ssh",
							"api",
						),
					),
				},
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Account uuid",
				Computed:            true,
			},
		},
	}
}

func (r *AccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*smc.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *smc.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func readAccountResourceModel(data *AccountResourceModel, item *smc.DefinitionsAccountsAccountPropertiesWithoutPassword) {
	data.Description = types.StringPointerValue(item.Description)
	data.DN = types.StringPointerValue(item.Dn)
	data.Email = types.StringPointerValue(item.Email)

	if item.Folders != nil {
		folderAttrs := make([]attr.Value, len(*item.Folders))
		for idx, folder := range *item.Folders {
			folderAttrs[idx] = types.StringValue(folder)
		}

		listValue, _ := types.ListValue(types.StringType, folderAttrs)
		data.Folders = listValue
	}

	data.Identifier = types.StringPointerValue(item.Identifier)
	data.Kind = types.StringPointerValue(item.Kind)
	data.LocalAuth = types.BoolPointerValue(item.LocalAuth)
	data.Name = types.StringPointerValue(item.Name)

	if item.Permissions != nil {
		permissionAttrs := make([]attr.Value, len(*item.Permissions))
		for idx, permission := range *item.Permissions {
			permissionAttrs[idx] = types.StringValue(string(permission))
		}

		listValue, _ := types.ListValue(types.StringType, permissionAttrs)
		data.Permissions = listValue
	}

	data.UUID = types.StringValue(item.Uuid)

	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
}

func (r *AccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AccountResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	body, err := json.Marshal(data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting the JSON encoding of the SMC Account data",
			"Could not get the JSON encoding of the SMC Account data: "+err.Error(),
		)
		return
	}

	respAPI, err := r.client.PostApiAccountsWithBodyWithResponse(ctx, "application/json", bytes.NewBuffer(body))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating the SMC Account",
			"Could not read the SMC account identifier "+data.Identifier.ValueString()+": "+err.Error(),
		)
		return
	}

	if respAPI.StatusCode() != http.StatusCreated {
		resp.Diagnostics.AddError(
			"HTTP Error Creating the SMC Account",
			"HTTP status code "+respAPI.Status()+" returned while creating the SMC account",
		)
		return
	}

	if respAPI.JSON201 == nil || respAPI.JSON201.Result == nil {
		resp.Diagnostics.AddError(
			"No results Reading response after creating the SMC Account",
			"No results returned after creating SMC the Account",
		)
		return
	}

	readAccountResourceModel(&data, respAPI.JSON201.Result)

	// Write logs using the tflog package
	tflog.Trace(ctx, "Created an account", map[string]interface{}{"uuid": data.UUID})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AccountResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	respAPI, err := r.client.GetApiAccountsUuidWithResponse(ctx, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading the SMC Account",
			"Could not read the SMC account with UUID "+data.UUID.String()+": "+err.Error(),
		)
		return
	}

	if respAPI.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"HTTP Error Reading the SMC Account",
			"HTTP status code "+respAPI.Status()+" returned while reading the SMC account",
		)
		return
	}

	if respAPI.JSON200 == nil {
		resp.Diagnostics.AddError(
			"No result Reading the SMC Account",
			"No result returned after reading the SMC Account",
		)
		return
	}

	readAccountResourceModel(&data, respAPI.JSON200)

	// Write logs using the tflog package
	tflog.Trace(ctx, "Read an account", map[string]interface{}{"uuid": data.UUID})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AccountResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	body, err := json.Marshal(data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting the JSON encoding of the SMC Account data",
			"Could not get the JSON encoding of the SMC Account data: "+err.Error(),
		)
		return
	}

	respAPI, err := r.client.PutApiAccountsUuidWithBodyWithResponse(ctx, data.UUID.String(), "application/json", bytes.NewBuffer(body))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating the SMC Account",
			"Could not update the SMC account UUID "+data.UUID.String()+": "+err.Error(),
		)
		return
	}

	if respAPI.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"HTTP Error Updating the SMC Account",
			"HTTP status code "+respAPI.Status()+" returned while updating the SMC account",
		)
		return
	}

	if respAPI.JSON200 == nil || respAPI.JSON200.Result == nil {
		resp.Diagnostics.AddError(
			"No results Reading response after updating the SMC Account",
			"No results returned after updating the SMC Account",
		)
		return
	}

	readAccountResourceModel(&data, respAPI.JSON200.Result)

	// Write logs using the tflog package
	tflog.Trace(ctx, "Updated an account", map[string]interface{}{"uuid": data.UUID})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AccountResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	respAPI, err := r.client.DeleteApiAccountsUuidWithResponse(ctx, data.UUID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting the SMC Account",
			"Could not delete the SMC account UUID "+data.UUID.String()+": "+err.Error(),
		)
		return
	}

	if respAPI.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"HTTP Error Deleting the SMC Account",
			"HTTP status code "+respAPI.Status()+" returned while deleting the SMC account",
		)
		return
	}

	if respAPI.JSON200 == nil || respAPI.JSON200.Result == nil {
		resp.Diagnostics.AddError(
			"No results Reading response after deleting the SMC Account",
			"No results returned after deleting SMC the Account",
		)
		return
	}
}

func (r *AccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}
