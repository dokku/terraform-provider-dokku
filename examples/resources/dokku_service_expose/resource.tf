resource "dokku_plugin" "postgres" {
  name = "postgres"
  url  = "https://github.com/dokku/dokku-postgres.git"
}

resource "dokku_service_create" "postgres_my_db" {
  service    = "postgres"
  name       = "my-db"
  depends_on = [dokku_plugin.postgres]
}

# Expose a postgres service named my-db on host port 5432
resource "dokku_service_expose" "example" {
  service    = dokku_service_create.postgres_my_db.service
  name       = dokku_service_create.postgres_my_db.name
  ports      = ["5432"]
  depends_on = [dokku_plugin.postgres]
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

# Expose a redis service named my-redis on host port 6379
resource "dokku_service_expose" "example_2" {
  service    = dokku_service_create.redis_my_redis.service
  name       = dokku_service_create.redis_my_redis.name
  ports      = ["6379"]
  depends_on = [dokku_plugin.redis]
}
