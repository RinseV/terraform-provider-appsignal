resource "appsignal_app" "example" {
  name              = "my-example-app"
  environment       = "production"
  organization_slug = "my-org"
}
