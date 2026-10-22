resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Setting the deploy timeout for an app
resource "dokku_scheduler_k3s_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "deploy-timeout"
  value    = "300s"
}

# Setting the namespace for an app
resource "dokku_scheduler_k3s_property" "example_2" {
  app      = dokku_app.node_js_app.app
  property = "namespace"
  value    = "production"
}

# Setting the letsencrypt prod email globally
resource "dokku_scheduler_k3s_property" "example_3" {
  global   = true
  property = "letsencrypt-email-prod"
  value    = "admin@example.com"
}
