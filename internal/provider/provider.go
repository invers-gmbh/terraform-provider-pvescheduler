package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PveSchedulerProvider struct{}

type PveSchedulerProviderModel struct {
	Endpoint           types.String `tfsdk:"endpoint"`
	ApiToken           types.String `tfsdk:"api_token"`
	Username           types.String `tfsdk:"username"`
	Password           types.String `tfsdk:"password"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
	Nodes              types.List   `tfsdk:"nodes"`
}

func New() provider.Provider { return &PveSchedulerProvider{} }

func (p *PveSchedulerProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "pvescheduler"
}

func (p *PveSchedulerProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schedules Proxmox VMs onto the least-loaded node. Falls back to PROXMOX_VE_ENDPOINT and PROXMOX_VE_API_TOKEN env vars if attributes are omitted.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "PVE API endpoint, e.g. https://pve.example.com:8006. Falls back to PROXMOX_VE_ENDPOINT.",
			},
			"api_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "PVE API token in the form user@realm!tokenid=secret. Falls back to PROXMOX_VE_API_TOKEN.",
			},
			"username": schema.StringAttribute{
				Optional:    true,
				Description: "PVE API username in the form user@realm. Falls back to PROXMOX_VE_USERNAME.",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "PVE API password. Used with `username`. Falls back to PROXMOX_VE_PASSWORD.",
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Optional:    true,
				Description: "Skip TLS certificate verification. Falls back to PROXMOX_VE_INSECURE=true.",
			},
			"nodes": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Global allowlist of PVE node names eligible for scheduling. If omitted, all online nodes are considered.",
			},
		},
	}
}

func (p *PveSchedulerProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config PveSchedulerProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := config.Endpoint.ValueString()
	if endpoint == "" {
		endpoint = os.Getenv("PROXMOX_VE_ENDPOINT")
	}
	apiToken := config.ApiToken.ValueString()
	if apiToken == "" {
		apiToken = os.Getenv("PROXMOX_VE_API_TOKEN")
	}

	username := config.Username.ValueString()
	if username == "" {
		username = os.Getenv("PROXMOX_VE_USERNAME")
	}

	password := config.Password.ValueString()
	if password == "" {
		password = os.Getenv("PROXMOX_VE_PASSWORD")
	}

	insecure := config.InsecureSkipVerify.ValueBool()
	if !insecure {
		insecure = os.Getenv("PROXMOX_VE_INSECURE") == "true"
	}

	if endpoint == "" {
		resp.Diagnostics.AddError("missing endpoint", "set endpoint in provider config or PROXMOX_VE_ENDPOINT env var")
		return
	}

	var nodes []string
	if !config.Nodes.IsNull() && !config.Nodes.IsUnknown() {
		resp.Diagnostics.Append(config.Nodes.ElementsAs(ctx, &nodes, false)...)
	}

	var client *PveClient
	switch {
	case apiToken != "":
		client = NewClientWithToken(endpoint, apiToken, insecure, nodes)
	case username != "" && password != "":
		var err error
		client, err = NewClientWithPassword(endpoint, username, password, insecure, nodes)
		if err != nil {
			resp.Diagnostics.AddError("authentication failed", err.Error())
			return
		}
	default:
		resp.Diagnostics.AddError("no credentials",
			"set api_token or username+password in provider config or via env vars")
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *PveSchedulerProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewNodeSchedulerDataSource,
	}
}

func (p *PveSchedulerProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewPlacementResource,
	}
}
