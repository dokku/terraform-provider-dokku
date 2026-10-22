resource "dokku_app" "hello_world" {
  app = "hello-world"
}

# Sync a git repository to an app
resource "dokku_git_sync" "example" {
  app                = dokku_app.hello_world.app
  remote             = "https://github.com/dokku/smoke-test-app.git"
  build              = false
  build_if_changes   = false
  skip_deploy_branch = false
}
