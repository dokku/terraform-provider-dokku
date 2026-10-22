resource "dokku_app" "example_app" {
  app = "example-app"
}

# Add domains to an app
resource "dokku_domains" "example" {
  app     = dokku_app.example_app.app
  domains = ["example.com", "www.example.com"]
}

# Set global domains
resource "dokku_domains" "example_2" {
  global  = true
  domains = ["global.example.com"]
}
