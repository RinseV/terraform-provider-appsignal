# Configuration-based authentication
provider "appsignal" {
  host  = "https://appsignal.com/graphql"
  token = "abcdef..."

  organization_slug = "my-org"
}
