package provider

import (
	"context"
	"os"

	"github.com/dscof/terraform-provider-quartermaster/internal/client"
	"github.com/dscof/terraform-provider-quartermaster/internal/resources"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type quartermasterProvider struct{}

type quartermasterProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Token    types.String `tfsdk:"token"`
	Insecure types.Bool   `tfsdk:"insecure"`
}

func New() provider.Provider {
	return &quartermasterProvider{}
}

func (p *quartermasterProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "quartermaster"
}

func (p *quartermasterProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for Quartermaster",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Required:    true,
				Description: "Quartermaster server URL",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Bearer JWT for admin auth. Can also be set via QUARTERMASTER_TOKEN env var.",
			},
			"insecure": schema.BoolAttribute{
				Optional:    true,
				Description: "Skip TLS certificate verification",
			},
		},
	}
}

func (p *quartermasterProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config quartermasterProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := config.Endpoint.ValueString()
	token := config.Token.ValueString()
	if token == "" {
		token = os.Getenv("QUARTERMASTER_TOKEN")
	}
	insecure := config.Insecure.ValueBool()

	c := client.New(endpoint, token, insecure)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *quartermasterProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewBilletResource,
		resources.NewPolicyResource,
		resources.NewBilletAssignmentResource,
	}
}

func (p *quartermasterProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
