data "dokku_logs_property" "example" {
  app      = "node-js-app"
  property = "max-size"
}
