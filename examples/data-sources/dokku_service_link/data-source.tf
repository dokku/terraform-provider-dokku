data "dokku_service_link" "example" {
  app     = "my-app"
  service = "redis"
  name    = "my-redis"
}
