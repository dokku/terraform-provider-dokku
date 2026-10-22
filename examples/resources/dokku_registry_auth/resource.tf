resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Log in to a registry for an app
resource "dokku_registry_auth" "example" {
  app      = dokku_app.node_js_app.app
  server   = "ghcr.io"
  username = "deploy-bot"
  password = "ghp_examplepat"
}

# Log in to a registry globally
resource "dokku_registry_auth" "example_2" {
  global   = true
  server   = "docker.io"
  username = "deploy-bot"
  password = "examplepassword"
}
