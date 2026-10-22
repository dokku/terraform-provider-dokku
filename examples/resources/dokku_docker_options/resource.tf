resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Mount the docker socket at deploy
resource "dokku_docker_options" "example" {
  app    = dokku_app.node_js_app.app
  phase  = "deploy"
  option = "-v /var/run/docker.sock:/var/run/docker.sock"
}

# Scope a deploy option to the web process type
resource "dokku_docker_options" "example_2" {
  app          = dokku_app.node_js_app.app
  phase        = "deploy"
  process_type = "web"
  option       = "--memory=512m"
}
