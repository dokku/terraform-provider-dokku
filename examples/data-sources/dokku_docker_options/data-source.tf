data "dokku_docker_options" "example" {
  app    = "node-js-app"
  phase  = "deploy"
  option = "-v /var/run/docker.sock:/var/run/docker.sock"
}
