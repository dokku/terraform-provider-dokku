data "dokku_acl_service" "example" {
  service = "my-redis"
  type    = "redis"
}
