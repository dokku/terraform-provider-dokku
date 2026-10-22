data "dokku_scheduler_k3s_labels" "example" {
  app           = "node-js-app"
  process_type  = "web"
  resource_type = "deployment"
}
