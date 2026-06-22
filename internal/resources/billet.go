package resources

import (
	"context"

	"github.com/dscof/terraform-provider-quartermaster/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type billetResource struct {
	client *client.Client
}

type billetModel struct {
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	Tags              types.List   `tfsdk:"tags"`
	AssociatedAWSRoles types.List   `tfsdk:"associated_aws_roles"`
	AssociatedGCPSAs  types.List   `tfsdk:"associated_gcp_sas"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func NewBilletResource() resource.Resource {
	return &billetResource{}
}

func (r *billetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_billet"
}

func (r *billetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Quartermaster billet",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Billet name",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"tags": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"associated_aws_roles": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"associated_gcp_sas": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *billetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client.Client)
}

func (r *billetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan billetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	b := &client.Billet{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}
	plan.Tags.ElementsAs(ctx, &b.Tags, false)
	plan.AssociatedAWSRoles.ElementsAs(ctx, &b.AssociatedAWSRoles, false)
	plan.AssociatedGCPSAs.ElementsAs(ctx, &b.AssociatedGCPSAs, false)

	result, err := r.client.CreateBillet(b)
	if err != nil {
		resp.Diagnostics.AddError("Error creating billet", mapAPIError(err))
		return
	}

	plan.UpdatedAt = types.StringValue(result.UpdatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *billetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state billetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetBillet(state.Name.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading billet", mapAPIError(err))
		return
	}

	state.Description = types.StringValue(result.Description)
	state.Tags, _ = types.ListValueFrom(ctx, types.StringType, result.Tags)
	state.AssociatedAWSRoles, _ = types.ListValueFrom(ctx, types.StringType, result.AssociatedAWSRoles)
	state.AssociatedGCPSAs, _ = types.ListValueFrom(ctx, types.StringType, result.AssociatedGCPSAs)
	state.UpdatedAt = types.StringValue(result.UpdatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *billetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan billetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	b := &client.Billet{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}
	plan.Tags.ElementsAs(ctx, &b.Tags, false)
	plan.AssociatedAWSRoles.ElementsAs(ctx, &b.AssociatedAWSRoles, false)
	plan.AssociatedGCPSAs.ElementsAs(ctx, &b.AssociatedGCPSAs, false)

	result, err := r.client.UpdateBillet(plan.Name.ValueString(), b)
	if err != nil {
		resp.Diagnostics.AddError("Error updating billet", mapAPIError(err))
		return
	}

	plan.UpdatedAt = types.StringValue(result.UpdatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *billetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state billetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteBillet(state.Name.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error deleting billet", mapAPIError(err))
	}
}
