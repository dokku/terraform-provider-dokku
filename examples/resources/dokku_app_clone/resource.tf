resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

resource "dokku_git_from_image" "node_js_app_git_from_image" {
  app   = dokku_app.node_js_app.app
  image = "dokku/smoke-test-app:dockerfile"
}

# Clone an app
resource "dokku_app_clone" "example" {
  app        = "node-js-app-staging"
  source_app = "node-js-app"
  depends_on = [dokku_app.node_js_app, dokku_git_from_image.node_js_app_git_from_image]
}
