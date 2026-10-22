resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Overriding the auto-selected builder
resource "dokku_builder_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "selected"
  value    = "dockerfile"
}

resource "dokku_app" "monorepo" {
  app = "monorepo"
}

# Changing the build build directory
resource "dokku_builder_property" "example_2" {
  app      = dokku_app.monorepo.app
  property = "build-dir"
  value    = "backend"
}

# Overriding the auto-selected builder globally
resource "dokku_builder_property" "example_3" {
  global   = true
  property = "selected"
  value    = "herokuish"
}
