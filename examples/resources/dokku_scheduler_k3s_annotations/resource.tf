resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Set deployment annotations on an app's web process
resource "dokku_scheduler_k3s_annotations" "example" {
  app           = dokku_app.node_js_app.app
  process_type  = "web"
  resource_type = "deployment"
  annotations = {
    "prometheus.io/port"   = "9090"
    "prometheus.io/scrape" = "true"
  }
}

# Set ingress annotations on an app at the global process scope
resource "dokku_scheduler_k3s_annotations" "example_2" {
  app           = dokku_app.node_js_app.app
  resource_type = "ingress"
  annotations = {
    "nginx.ingress.kubernetes.io/rewrite-target" = "/"
  }
}

# Set a global deployment annotation across all apps
resource "dokku_scheduler_k3s_annotations" "example_3" {
  global        = true
  resource_type = "deployment"
  annotations = {
    managed-by = "docket"
  }
}
