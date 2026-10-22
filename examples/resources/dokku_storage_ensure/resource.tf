resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Ensure a storage directory owned by the herokuish user
resource "dokku_storage_ensure" "example" {
  app   = dokku_app.node_js_app.app
  chown = "herokuish"
}
