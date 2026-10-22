resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Deploy a tar.gz archive
resource "dokku_git_from_archive" "example" {
  app          = dokku_app.node_js_app.app
  archive_url  = "https://github.com/dokku/smoke-test-app/archive/refs/heads/master.tar.gz"
  archive_type = "tar.gz"
}
