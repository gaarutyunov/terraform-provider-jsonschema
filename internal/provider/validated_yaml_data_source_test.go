// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestValidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	metadataDir := filepath.Join(tmpDir, "metadata")

	err := os.Mkdir(metadataDir, 0755)
	require.NoError(t, err)

	err = os.Mkdir(filepath.Join(metadataDir, "examples"), 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(metadataDir, "examples/example.yaml"), []byte(`
# yaml-language-server: $schema=../schema.json
id: "example-id"
name: "Example Name"
tags:
  - "tag1"
  - "tag2"
`), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(metadataDir, "schema.json"), []byte(testAccValidatedYAMLDataSourceSchema), 0644)
	require.NoError(t, err)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: fmt.Sprintf(testAccValidatedYAMLDataSourceConfig, filepath.Join(metadataDir, "**/*.yaml")),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.jsonschema_validated_yaml.metadata",
						tfjsonpath.New("values").AtMapKey(filepath.Join(metadataDir, "examples/example.yaml")),
						knownvalue.StringExact(`id: "example-id"
name: "Example Name"
tags:
  - "tag1"
  - "tag2"`),
					),
				},
			},
		},
	})
}

func TestInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	metadataDir := filepath.Join(tmpDir, "metadata")

	err := os.Mkdir(metadataDir, 0755)
	require.NoError(t, err)

	err = os.Mkdir(filepath.Join(metadataDir, "examples"), 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(metadataDir, "examples/example.yaml"), []byte(`
# yaml-language-server: $schema=../schema.json
id: 12345
name: "Example Name"
tags:
  - "tag1"
  - "tag2"
`), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(metadataDir, "schema.json"), []byte(testAccValidatedYAMLDataSourceSchema), 0644)
	require.NoError(t, err)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config:      fmt.Sprintf(testAccValidatedYAMLDataSourceConfig, filepath.Join(metadataDir, "**/*.yaml")),
				ExpectError: regexp.MustCompile(`- at '/id': got number, want string`),
			},
		},
	})
}

func TestNoSchema(t *testing.T) {
	tmpDir := t.TempDir()

	metadataDir := filepath.Join(tmpDir, "metadata")

	err := os.Mkdir(metadataDir, 0755)
	require.NoError(t, err)

	err = os.Mkdir(filepath.Join(metadataDir, "examples"), 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(metadataDir, "examples/example.yaml"), []byte(`
id: "example-id"
name: "Example Name"
tags:
  - "tag1"
  - "tag2"
`), 0644)
	require.NoError(t, err)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config:      fmt.Sprintf(testAccValidatedYAMLDataSourceConfig, filepath.Join(metadataDir, "**/*.yaml")),
				ExpectError: regexp.MustCompile(`does not contain a valid schema reference in the first line`),
			},
		},
	})
}

const (
	testAccValidatedYAMLDataSourceConfig = `
data "jsonschema_validated_yaml" "metadata" {
  input_pattern = "%s"
}
`
	testAccValidatedYAMLDataSourceSchema = `
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://github.com/gaarutyunov/terraform-provider-jsonschemas/test",
  "title": "Test Schema",
  "description": "Schema for Tests",
  "type": "object",
  "properties": {
	"id": {
	  "type": "string"
	},
	"name": {
	  "type": "string"
	},
	"tags": {
	  "type": "array",
	  "items": {
		"type": "string"
	  }
	}
  },
  "required": ["id", "name"]
}
`
)
