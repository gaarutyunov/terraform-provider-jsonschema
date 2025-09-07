// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Ensure JsonschemaProvider satisfies various provider interfaces.
var _ provider.Provider = &JsonschemaProvider{}

// JsonschemaProvider defines the provider implementation.
type JsonschemaProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// NewsProviderModel describes the provider data model.
type NewsProviderModel struct {
}

func (p *JsonschemaProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "jsonschema"
	resp.Version = p.version
}

func (p *JsonschemaProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provider for working with jsonschema.",
	}
}

func (p *JsonschemaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data NewsProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	compiler := jsonschema.NewCompiler()

	resp.DataSourceData = compiler
	resp.ResourceData = compiler
}

func (p *JsonschemaProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *JsonschemaProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewValidatedYAMLDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &JsonschemaProvider{
			version: version,
		}
	}
}
