resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Setting the project.toml path for an app
resource "dokku_builder_pack_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "projecttoml-path"
  value    = "config/project.toml"
}

# Setting the project.toml path globally
resource "dokku_builder_pack_property" "example_2" {
  global   = true
  property = "projecttoml-path"
  value    = "project.toml"
}
