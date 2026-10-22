resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Set deployment labels on an app's web process
resource "dokku_scheduler_k3s_labels" "example" {
  app           = dokku_app.node_js_app.app
  process_type  = "web"
  resource_type = "deployment"
  labels = {
    "app.kubernetes.io/component" = "api"
    tier                          = "edge"
  }
}

# Set ingress labels on an app at the global process scope
resource "dokku_scheduler_k3s_labels" "example_2" {
  app           = dokku_app.node_js_app.app
  resource_type = "ingress"
  labels = {
    team = "platform"
  }
}

# Set a global deployment label across all apps
resource "dokku_scheduler_k3s_labels" "example_3" {
  global        = true
  resource_type = "deployment"
  labels = {
    managed-by = "docket"
  }
}
