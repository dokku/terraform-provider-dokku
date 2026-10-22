resource "dokku_plugin" "acl" {
  name = "acl"
  url  = "https://github.com/dokku-community/dokku-acl.git"
}

resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Grant users access to an app
resource "dokku_acl_app" "example" {
  app        = dokku_app.node_js_app.app
  users      = ["alice", "bob"]
  depends_on = [dokku_plugin.acl]
}
