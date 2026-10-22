resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Lock an app
resource "dokku_app_lock" "example" {
  app = dokku_app.node_js_app.app
}
