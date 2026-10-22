data "dokku_service_expose" "example" {
  service = "postgres"
  name    = "my-db"
}
