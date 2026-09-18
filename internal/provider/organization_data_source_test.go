// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOrganizationDataSource(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing. The organization comes from the provider
			// configuration, so the data source takes no arguments.
			{
				Config: providerConfig + `data "appsignal_organization" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.appsignal_organization.test", "slug", "terraform-test"),
					resource.TestCheckResourceAttrSet("data.appsignal_organization.test", "id"),
					resource.TestCheckResourceAttrSet("data.appsignal_organization.test", "name"),
				),
			},
		},
	})
}
