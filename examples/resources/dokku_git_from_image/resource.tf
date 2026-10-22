resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Deploy an app from a docker image
resource "dokku_git_from_image" "example" {
  app   = dokku_app.node_js_app.app
  image = "dokku/smoke-test-app:dockerfile"
}
