data "dokku_scheduler_docker_local_property" "example" {
  app      = "node-js-app"
  property = "init-process"
}
