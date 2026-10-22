resource "dokku_plugin" "redis" {
  name = "redis"
  url  = "https://github.com/dokku/dokku-redis.git"
}

resource "dokku_app" "my_app" {
  app = "my-app"
}

resource "dokku_service_create" "redis_my_redis" {
  service    = "redis"
  name       = "my-redis"
  depends_on = [dokku_plugin.redis]
}

resource "dokku_git_from_image" "my_app_git_from_image" {
  app   = dokku_app.my_app.app
  image = "dokku/smoke-test-app:dockerfile"
}

# Link a redis service named my-redis to my-app
resource "dokku_service_link" "example" {
  app        = dokku_app.my_app.app
  service    = dokku_service_create.redis_my_redis.service
  name       = dokku_service_create.redis_my_redis.name
  depends_on = [dokku_plugin.redis, dokku_git_from_image.my_app_git_from_image]
}

resource "dokku_plugin" "postgres" {
  name = "postgres"
  url  = "https://github.com/dokku/dokku-postgres.git"
}

resource "dokku_service_create" "postgres_my_db" {
  service    = "postgres"
  name       = "my-db"
  depends_on = [dokku_plugin.postgres]
}

# Link a postgres service named my-db to my-app
resource "dokku_service_link" "example_2" {
  app        = dokku_app.my_app.app
  service    = dokku_service_create.postgres_my_db.service
  name       = dokku_service_create.postgres_my_db.name
  depends_on = [dokku_plugin.postgres, dokku_git_from_image.my_app_git_from_image]
}
