resource "dokku_plugin" "postgres" {
  name = "postgres"
  url  = "https://github.com/dokku/dokku-postgres.git"
}

resource "dokku_service_create" "postgres_my_db" {
  service    = "postgres"
  name       = "my-db"
  depends_on = [dokku_plugin.postgres]
}

# Schedule daily backups of a postgres service to S3
resource "dokku_service_backup" "example" {
  service               = dokku_service_create.postgres_my_db.service
  name                  = dokku_service_create.postgres_my_db.name
  schedule              = "0 3 * * *"
  bucket                = "my-backup-bucket"
  aws_access_key_id     = "AKIAEXAMPLE"
  aws_secret_access_key = "examplesecret"
  aws_default_region    = "us-east-1"
  encryption_passphrase = "correct-horse-battery-staple"
  depends_on            = [dokku_plugin.postgres]
}
