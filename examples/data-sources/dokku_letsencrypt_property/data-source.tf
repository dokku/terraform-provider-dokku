data "dokku_letsencrypt_property" "example" {
  app      = "node-js-app"
  property = "email"
}
