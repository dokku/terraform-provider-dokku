resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Setting the deploy branch for an app
resource "dokku_git_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "deploy-branch"
  value    = "main"
}

# Keeping the .git directory during builds
resource "dokku_git_property" "example_2" {
  app      = dokku_app.node_js_app.app
  property = "keep-git-dir"
  value    = "true"
}

# Setting the rev env var for an app
resource "dokku_git_property" "example_3" {
  app      = dokku_app.node_js_app.app
  property = "rev-env-var"
  value    = "GIT_REV"
}
