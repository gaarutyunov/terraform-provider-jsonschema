// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var schemaRegex = regexp.MustCompile(`# yaml-language-server: \$schema=(.+)`)

func NewValidatedYAMLDataSource() datasource.DataSource {
	return &ValidatedYAMLDataSource{}
}

// ValidatedYAMLDataSource defines the data source implementation.
type ValidatedYAMLDataSource struct {
	compiler *jsonschema.Compiler
}

// ValidatedYAMLDataSourceModel describes the data source data model.
type ValidatedYAMLDataSourceModel struct {
	InputPattern types.String `tfsdk:"input_pattern"`
	Values       types.Map    `tfsdk:"values"`
}

func (d *ValidatedYAMLDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_validated_yaml"
}

func (d *ValidatedYAMLDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "YAML files validated against a json schema",

		Attributes: map[string]schema.Attribute{
			"input_pattern": schema.StringAttribute{
				Description: "Directory containing YAML files to validate",
				Required:    true,
			},
			"values": schema.MapAttribute{
				Description: "Map of file paths to validated YAML content",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *ValidatedYAMLDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	compiler, ok := req.ProviderData.(*jsonschema.Compiler)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *jsonschema.Compiler, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.compiler = compiler
}

func (d *ValidatedYAMLDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ValidatedYAMLDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.InputPattern = types.StringValue(data.InputPattern.ValueString())

	files, err := filepath.Glob(data.InputPattern.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading input files",
			"Could not read input files: "+err.Error(),
		)
		return
	}

	if len(files) == 0 {
		resp.Diagnostics.AddError(
			"No input files found",
			"No files matched the provided input pattern: "+data.InputPattern.ValueString(),
		)
		return
	}

	valuesMap := make(map[string]string)
	for _, file := range files {
		func() {
			fi, err := os.Open(file)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error opening file",
					"Could not open file "+file+": "+err.Error(),
				)
				return
			}
			defer func(fi *os.File) {
				err := fi.Close()
				if err != nil {
					resp.Diagnostics.AddError(
						"Error closing file",
						"Could not close file "+file+": "+err.Error(),
					)
				}
			}(fi)

			contentRaw, err := io.ReadAll(fi)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error reading file",
					"Could not read file "+file+": "+err.Error(),
				)
				return
			}

			content := string(contentRaw)

			// check that first line contains schema reference
			// e.g. # yaml-language-server: $schema=path
			matches := schemaRegex.FindStringSubmatchIndex(content)
			// matches should contain 4 elements: full match start, full match end, first group start, first group end
			if len(matches) != 4 {
				resp.Diagnostics.AddError(
					"Error validating file",
					"File "+file+" does not contain a valid schema reference in the first line, e.g. '# yaml-language-server: $schema=path'",
				)
				return
			}

			schemaPath := filepath.Join(filepath.Dir(file), content[matches[2]:matches[3]])

			compiledSchema, err := d.compiler.Compile(schemaPath)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error compiling schema",
					"Could not compile schema "+schemaPath+" for file "+file+": "+err.Error(),
				)
				return
			}

			var value interface{}

			err = yaml.Unmarshal(contentRaw, &value)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error decoding YAML",
					"Could not decode YAML file "+file+": "+err.Error(),
				)
				return
			}

			err = compiledSchema.Validate(value)

			if err != nil {
				resp.Diagnostics.AddError(
					"Error validating YAML",
					"YAML file "+file+" does not conform to schema "+schemaPath+": "+err.Error(),
				)
				return
			}

			// content without the first line (which contains the schema reference)
			valuesMap[file] = strings.Trim(content[matches[1]:], "\n")
		}()
	}

	values, diag := types.MapValueFrom(ctx, types.StringType, valuesMap)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Values = values

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
