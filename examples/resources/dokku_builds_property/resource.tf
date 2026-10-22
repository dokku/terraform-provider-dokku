resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Setting the retention value for an app
resource "dokku_builds_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "retention"
  value    = "50"
}

# Setting the retention value globally
resource "dokku_builds_property" "example_2" {
  global   = true
  property = "retention"
  value    = "50"
}
