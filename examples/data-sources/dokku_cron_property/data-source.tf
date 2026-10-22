data "dokku_cron_property" "example" {
  app      = "node-js-app"
  property = "maintenance"
}
