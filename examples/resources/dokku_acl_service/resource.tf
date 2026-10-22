resource "dokku_plugin" "acl" {
  name = "acl"
  url  = "https://github.com/dokku-community/dokku-acl.git"
}

resource "dokku_plugin" "redis" {
  name = "redis"
  url  = "https://github.com/dokku/dokku-redis.git"
}

resource "dokku_service_create" "redis_my_redis" {
  service    = "redis"
  name       = "my-redis"
  depends_on = [dokku_plugin.redis]
}

# Grant users access to a redis service
resource "dokku_acl_service" "example" {
  service    = dokku_service_create.redis_my_redis.name
  type       = dokku_service_create.redis_my_redis.service
  users      = ["alice", "bob"]
  depends_on = [dokku_plugin.acl, dokku_plugin.redis]
}
