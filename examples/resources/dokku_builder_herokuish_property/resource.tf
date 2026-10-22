resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Allowing the herokuish builder for an app
resource "dokku_builder_herokuish_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "allowed"
  value    = "true"
}

# Allowing the herokuish builder globally
resource "dokku_builder_herokuish_property" "example_2" {
  global   = true
  property = "allowed"
  value    = "true"
}
