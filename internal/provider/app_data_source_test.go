package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAppDataSourceByID(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing.
			{
				Config: providerConfig + `data "appsignal_app" "test" { id = "6aad24d1ba6bc351255e7cb5" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.appsignal_app.test", "id"),
					resource.TestCheckResourceAttrSet("data.appsignal_app.test", "name"),
					resource.TestCheckResourceAttrSet("data.appsignal_app.test", "environment"),
					resource.TestCheckResourceAttrSet("data.appsignal_app.test", "created_at"),
					resource.TestCheckResourceAttrSet("data.appsignal_app.test", "updated_at"),
				),
			},
		},
	})
}

func TestAccAppDataSourceByNameAndEnvironment(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// An app is created first so the lookup has something to find that
			// is not shared with any other test.
			{
				Config: providerConfig + `
resource "appsignal_app" "test" {
  name        = "my-test-data-source-app"
  environment = "testing"
}

data "appsignal_app" "test" {
  name        = appsignal_app.test.name
  environment = appsignal_app.test.environment
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.appsignal_app.test", "name", "my-test-data-source-app"),
					resource.TestCheckResourceAttr("data.appsignal_app.test", "environment", "testing"),
					resource.TestCheckResourceAttrPair("data.appsignal_app.test", "id", "appsignal_app.test", "id"),
					resource.TestCheckResourceAttrSet("data.appsignal_app.test", "created_at"),
					resource.TestCheckResourceAttrSet("data.appsignal_app.test", "updated_at"),
				),
			},
		},
	})
}

func TestAccAppDataSourceInvalidConfig(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Neither an ID nor a name and environment combination.
			{
				Config:      providerConfig + `data "appsignal_app" "test" {}`,
				ExpectError: regexp.MustCompile(`Exactly one of these attributes must be configured: \[id,name\]`),
			},
			// An ID combined with a name and environment combination.
			{
				Config: providerConfig + `
data "appsignal_app" "test" {
  id          = "6aad24d1ba6bc351255e7cb5"
  name        = "my-test-app"
  environment = "testing"
}
`,
				ExpectError: regexp.MustCompile(`These attributes cannot be configured together: \[id,environment\]`),
			},
			// A name without an environment.
			{
				Config:      providerConfig + `data "appsignal_app" "test" { name = "my-test-app" }`,
				ExpectError: regexp.MustCompile(`These attributes must be configured together: \[name,environment\]`),
			},
		},
	})
}
