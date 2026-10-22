resource "dokku_plugin" "letsencrypt" {
  name = "letsencrypt"
  url  = "https://github.com/dokku/dokku-letsencrypt.git"
}

resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

resource "dokku_git_from_image" "node_js_app_git_from_image" {
  app   = dokku_app.node_js_app.app
  image = "dokku/smoke-test-app:dockerfile"
}

# Enable letsencrypt for an app
resource "dokku_letsencrypt" "example" {
  app        = dokku_app.node_js_app.app
  depends_on = [dokku_plugin.letsencrypt, dokku_git_from_image.node_js_app_git_from_image]
}
