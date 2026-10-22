resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Setting the stack value for an app
resource "dokku_buildpacks_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "stack"
  value    = "gliderlabs/herokuish:latest"
}

# Setting the stack value globally
resource "dokku_buildpacks_property" "example_2" {
  global   = true
  property = "stack"
  value    = "gliderlabs/herokuish:latest"
}
