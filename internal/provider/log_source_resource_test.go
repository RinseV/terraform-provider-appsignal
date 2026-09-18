// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccLogSourceConfig returns a configuration with an app to hang the log
// source off, so the test owns every resource it touches and cleans them all
// up again.
func testAccLogSourceConfig(name, format string) string {
	return providerConfig + fmt.Sprintf(`
resource "appsignal_app" "test" {
  name        = "my-test-log-source-app"
  environment = "testing"
}

resource "appsignal_log_source" "test" {
  app_id = appsignal_app.test.id
  name   = %[1]q
  type   = "http"
  fmt    = %[2]q
}
`, name, format)
}

func TestAccLogSourceResource(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccLogSourceConfig("my-test-log-source", "PLAINTEXT"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appsignal_log_source.test", "name", "my-test-log-source"),
					resource.TestCheckResourceAttr("appsignal_log_source.test", "type", "http"),
					resource.TestCheckResourceAttr("appsignal_log_source.test", "fmt", "PLAINTEXT"),
					resource.TestCheckResourceAttrSet("appsignal_log_source.test", "id"),
					resource.TestCheckResourceAttrSet("appsignal_log_source.test", "key"),
					resource.TestCheckResourceAttrPair(
						"appsignal_log_source.test", "app_id",
						"appsignal_app.test", "id",
					),
				),
			},
			// Update and Read testing
			{
				Config: testAccLogSourceConfig("my-test-log-source-updated", "JSON"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appsignal_log_source.test", "name", "my-test-log-source-updated"),
					resource.TestCheckResourceAttr("appsignal_log_source.test", "fmt", "JSON"),
					// type is not part of the update mutation, so it has to
					// survive an update untouched.
					resource.TestCheckResourceAttr("appsignal_log_source.test", "type", "http"),
					resource.TestCheckResourceAttrSet("appsignal_log_source.test", "key"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "appsignal_log_source.test",
				ImportState:       true,
				ImportStateVerify: true,
				// A log source is only readable through its app, so the import
				// ID carries both IDs.
				ImportStateIdFunc: testAccLogSourceImportStateID("appsignal_log_source.test"),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// testAccLogSourceImportStateID builds the "<app id>,<log source id>" import ID
// from the state the earlier steps left behind.
func testAccLogSourceImportStateID(resourceName string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		rs, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}

		return fmt.Sprintf("%s,%s", rs.Primary.Attributes["app_id"], rs.Primary.ID), nil
	}
}
