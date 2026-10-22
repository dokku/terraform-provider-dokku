resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Setting the image repo for an app
resource "dokku_registry_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "image-repo"
  value    = "registry.example.com/node-js-app"
}

# Enabling push-on-release for an app
resource "dokku_registry_property" "example_2" {
  app      = dokku_app.node_js_app.app
  property = "push-on-release"
  value    = "true"
}

# Setting the registry server globally
resource "dokku_registry_property" "example_3" {
  global   = true
  property = "server"
  value    = "registry.example.com"
}
