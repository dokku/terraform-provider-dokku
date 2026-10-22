data "dokku_scheduler_k3s_annotations" "example" {
  app           = "node-js-app"
  process_type  = "web"
  resource_type = "deployment"
}
