data "dokku_caddy_property" "example" {
  app      = "node-js-app"
  property = "tls-internal"
}
