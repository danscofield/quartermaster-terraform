package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/dscof/terraform-provider-quartermaster/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type billetAssignmentResource struct {
	client *client.Client
}

type billetAssignmentModel struct {
	ID          types.String `tfsdk:"id"`
	Billet      types.String `tfsdk:"billet"`
	Description types.String `tfsdk:"description"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`

	// Simplified matching fields (generate Cedar under the hood)
	AwsAccountID    types.String `tfsdk:"aws_account_id"`
	AwsRoleName     types.String `tfsdk:"aws_role_name"`
	GcpProjectID    types.String `tfsdk:"gcp_project_id"`
	OidcGroup       types.String `tfsdk:"oidc_group"`
	OidcIdpPrefix   types.String `tfsdk:"oidc_idp_prefix"`
	SpiffeID        types.String `tfsdk:"spiffe_id"`
	Selector        types.String `tfsdk:"selector"`
	Environment     types.String `tfsdk:"environment"`

	// Computed: the generated Cedar statement
	Statement types.String `tfsdk:"statement"`
}

func NewBilletAssignmentResource() resource.Resource {
	return &billetAssignmentResource{}
}

func (r *billetAssignmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_billet_assignment"
}

func (r *billetAssignmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Simplified billet assignment — generates Cedar policy from declarative match conditions",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"billet": schema.StringAttribute{
				Required:    true,
				Description: "Target billet name to assign",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"created_at": schema.StringAttribute{
				Computed: true,
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
			"statement": schema.StringAttribute{
				Computed:    true,
				Description: "The generated Cedar policy statement",
			},

			// Match conditions — at least one must be set
			"aws_account_id": schema.StringAttribute{
				Optional:    true,
				Description: "Match any AWS identity from this account",
			},
			"aws_role_name": schema.StringAttribute{
				Optional:    true,
				Description: "Match a specific AWS role name (requires aws_account_id)",
			},
			"gcp_project_id": schema.StringAttribute{
				Optional:    true,
				Description: "Match any GCP identity from this project",
			},
			"oidc_group": schema.StringAttribute{
				Optional:    true,
				Description: "Match OIDC identities with this group",
			},
			"oidc_idp_prefix": schema.StringAttribute{
				Optional:    true,
				Description: "Match OIDC identities from this IdP prefix (use with oidc_group)",
			},
			"spiffe_id": schema.StringAttribute{
				Optional:    true,
				Description: "Match an exact SPIFFE ID",
			},
			"selector": schema.StringAttribute{
				Optional:    true,
				Description: "Match a SPIRE workload selector (e.g., k8s:ns:billing)",
			},
			"environment": schema.StringAttribute{
				Optional:    true,
				Description: "Match workloads in this environment",
			},
		},
	}
}

func (r *billetAssignmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client.Client)
}

func (r *billetAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan billetAssignmentModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	statement := generateCedar(&plan)
	description := plan.Description.ValueString()

	result, err := r.client.CreatePolicy(plan.Billet.ValueString(), &client.Policy{
		Statement:   statement,
		Description: description,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create billet assignment", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)
	plan.Statement = types.StringValue(statement)
	plan.CreatedAt = types.StringValue(result.CreatedAt)
	plan.UpdatedAt = types.StringValue(result.UpdatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *billetAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state billetAssignmentModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.GetPolicy(state.Billet.ValueString(), state.ID.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read billet assignment", err.Error())
		return
	}

	state.Statement = types.StringValue(policy.Statement)
	state.UpdatedAt = types.StringValue(policy.UpdatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *billetAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan billetAssignmentModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state billetAssignmentModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	statement := generateCedar(&plan)
	description := plan.Description.ValueString()

	result, err := r.client.UpdatePolicy(plan.Billet.ValueString(), state.ID.ValueString(), &client.Policy{
		Statement:   statement,
		Description: description,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update billet assignment", err.Error())
		return
	}

	plan.ID = state.ID
	plan.Statement = types.StringValue(statement)
	plan.CreatedAt = state.CreatedAt
	plan.UpdatedAt = types.StringValue(result.UpdatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *billetAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state billetAssignmentModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePolicy(state.Billet.ValueString(), state.ID.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Failed to delete billet assignment", err.Error())
	}
}

// generateCedar builds a Cedar policy statement from the simplified match fields.
func generateCedar(m *billetAssignmentModel) string {
	billet := m.Billet.ValueString()
	var conditions []string
	principalType := "principal"

	// AWS conditions
	if !m.AwsAccountID.IsNull() && !m.AwsAccountID.IsUnknown() {
		principalType = "principal is Quartermaster::AwsRoleIdentity"
		conditions = append(conditions, fmt.Sprintf(`principal.account_id == "%s"`, m.AwsAccountID.ValueString()))
		if !m.AwsRoleName.IsNull() && !m.AwsRoleName.IsUnknown() {
			conditions = append(conditions, fmt.Sprintf(`principal.role_name == "%s"`, m.AwsRoleName.ValueString()))
		}
	}

	// GCP conditions
	if !m.GcpProjectID.IsNull() && !m.GcpProjectID.IsUnknown() {
		principalType = "principal is Quartermaster::GcpIdentity"
		conditions = append(conditions, fmt.Sprintf(`principal.project_id == "%s"`, m.GcpProjectID.ValueString()))
	}

	// OIDC conditions
	if !m.OidcGroup.IsNull() && !m.OidcGroup.IsUnknown() {
		principalType = "principal is Quartermaster::OidcIdentity"
		conditions = append(conditions, fmt.Sprintf(`principal.groups.contains("%s")`, m.OidcGroup.ValueString()))
		if !m.OidcIdpPrefix.IsNull() && !m.OidcIdpPrefix.IsUnknown() {
			conditions = append(conditions, fmt.Sprintf(`principal.idp_prefix == "%s"`, m.OidcIdpPrefix.ValueString()))
		}
	}

	// SPIRE exact match
	if !m.SpiffeID.IsNull() && !m.SpiffeID.IsUnknown() {
		principalType = "principal is Quartermaster::Workload"
		conditions = append(conditions, fmt.Sprintf(`principal.spiffe_id == "%s"`, m.SpiffeID.ValueString()))
	}

	// SPIRE selector match
	if !m.Selector.IsNull() && !m.Selector.IsUnknown() {
		principalType = "principal is Quartermaster::Workload"
		conditions = append(conditions, fmt.Sprintf(`context.selectors.contains("%s")`, m.Selector.ValueString()))
	}

	// Environment match
	if !m.Environment.IsNull() && !m.Environment.IsUnknown() {
		conditions = append(conditions, fmt.Sprintf(`principal.environment == "%s"`, m.Environment.ValueString()))
	}

	// Build the Cedar statement
	stmt := fmt.Sprintf(`permit(%s, action == Quartermaster::Action::"assumeBillet", resource == Quartermaster::Billet::"%s")`,
		principalType, billet)

	if len(conditions) > 0 {
		stmt += " when { " + strings.Join(conditions, " && ") + " }"
	}

	stmt += ";"
	return stmt
}
