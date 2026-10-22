data "dokku_scheduler_k3s_property" "example" {
  app      = "node-js-app"
  property = "deploy-timeout"
}
