resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Enabling the init process for an app
resource "dokku_scheduler_docker_local_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "init-process"
  value    = "true"
}

# Setting the parallel schedule count for an app
resource "dokku_scheduler_docker_local_property" "example_2" {
  app      = dokku_app.node_js_app.app
  property = "parallel-schedule-count"
  value    = "4"
}

# Setting the parallel schedule count globally
resource "dokku_scheduler_docker_local_property" "example_3" {
  global   = true
  property = "parallel-schedule-count"
  value    = "4"
}
