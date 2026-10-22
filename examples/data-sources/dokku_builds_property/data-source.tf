data "dokku_builds_property" "example" {
  app      = "node-js-app"
  property = "retention"
}
