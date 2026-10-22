resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Setting the dockerfile path for an app
resource "dokku_builder_dockerfile_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "dockerfile-path"
  value    = "Dockerfile.production"
}

# Setting the dockerfile path globally
resource "dokku_builder_dockerfile_property" "example_2" {
  global   = true
  property = "dockerfile-path"
  value    = "Dockerfile"
}
