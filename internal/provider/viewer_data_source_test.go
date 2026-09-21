package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccViewerDataSource reads the viewer from the AppSignal API. The account
// behind the token decides the values, so it only asserts every attribute came
// back set, which is enough to catch a schema or query that no longer matches
// the API.
func TestAccViewerDataSource(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing.
			{
				Config: providerConfig + `data "appsignal_viewer" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.appsignal_viewer.test", "id"),
					resource.TestCheckResourceAttrSet("data.appsignal_viewer.test", "name"),
					resource.TestCheckResourceAttrSet("data.appsignal_viewer.test", "email"),
				),
			},
		},
	})
}
