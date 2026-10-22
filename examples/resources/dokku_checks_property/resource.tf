resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Setting the wait-to-retire value for an app
resource "dokku_checks_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "wait-to-retire"
  value    = "60"
}

# Setting the wait-to-retire value globally
resource "dokku_checks_property" "example_2" {
  global   = true
  property = "wait-to-retire"
  value    = "60"
}
