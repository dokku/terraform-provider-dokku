data "dokku_ps_property" "example" {
  app      = "node-js-app"
  property = "restart-policy"
}
