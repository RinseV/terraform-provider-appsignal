package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAppResource(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
resource "appsignal_app" "test" {
  name        = "my-test-app"
  environment = "testing"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appsignal_app.test", "name", "my-test-app"),
					resource.TestCheckResourceAttr("appsignal_app.test", "environment", "testing"),
					resource.TestCheckResourceAttrSet("appsignal_app.test", "id"),
					resource.TestCheckResourceAttrSet("appsignal_app.test", "created_at"),
					resource.TestCheckResourceAttrSet("appsignal_app.test", "updated_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "appsignal_app.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
