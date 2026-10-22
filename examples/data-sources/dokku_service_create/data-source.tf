data "dokku_service_create" "example" {
  service = "redis"
  name    = "my-redis"
}
