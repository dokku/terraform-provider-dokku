resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Map http port 80 to container port 5000
resource "dokku_ports" "example" {
  app = dokku_app.node_js_app.app
  port_mappings = [{
    container = 5000
    host      = 80
    scheme    = "http"
  }]
}
