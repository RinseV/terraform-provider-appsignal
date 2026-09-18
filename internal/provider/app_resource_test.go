// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAppResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
resource "appsignal_app" "test" {
  name = "my-test-app"
	environment = "testing"
	organization_slug = "terraform-test"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appsignal_app.test", "name", "my-test-app"),
					resource.TestCheckResourceAttr("appsignal_app.test", "environment", "testing"),
					resource.TestCheckResourceAttrSet("appsignal_app.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "appsignal_app.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated", "organization_slug"},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
