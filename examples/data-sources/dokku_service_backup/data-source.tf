data "dokku_service_backup" "example" {
  service = "postgres"
  name    = "my-db"
}
