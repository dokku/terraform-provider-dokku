data "dokku_checks_property" "example" {
  app      = "node-js-app"
  property = "wait-to-retire"
}
