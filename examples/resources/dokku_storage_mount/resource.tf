resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

resource "dokku_storage_entry" "node_js_app_data" {
  name  = "node-js-app-data"
  chown = "herokuish"
}

# Declare the complete set of mounts for an app, removing any other
resource "dokku_storage_mount" "example" {
  app = dokku_app.node_js_app.app
  mounts = [{
    container_dir = "/app/storage"
    entry_name    = "node-js-app-data"
    }, {
    container_dir  = "/app/uploads"
    host_dir       = "/var/lib/dokku/data/storage/node-js-app"
    volume_options = "Z"
  }]
  depends_on = [dokku_storage_entry.node_js_app_data]
}

# Declare the web process's mounts, each with its own subpath
resource "dokku_storage_mount" "example_2" {
  app          = dokku_app.node_js_app.app
  process_type = "web"
  mounts = [{
    container_dir = "/app/assets"
    entry_name    = "node-js-app-data"
    phases        = ["deploy"]
    readonly      = true
    subpath       = "assets"
    }, {
    container_dir = "/app/cache"
    entry_name    = "node-js-app-data"
    subpath       = "cache"
  }]
  depends_on = [dokku_storage_entry.node_js_app_data]
}
