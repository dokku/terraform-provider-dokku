resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Add buildpacks to an app
resource "dokku_buildpacks" "example" {
  app        = dokku_app.node_js_app.app
  buildpacks = ["https://github.com/heroku/heroku-buildpack-nodejs.git", "https://github.com/heroku/heroku-buildpack-nginx.git"]
}
