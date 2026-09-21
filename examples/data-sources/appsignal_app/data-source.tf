# Get app by ID
data "appsignal_app" "by_id" {
  id = "abcdef"
}

# Get app by name and environment
data "appsignal_app" "by_name" {
  name        = "my-app"
  environment = "production"
}
