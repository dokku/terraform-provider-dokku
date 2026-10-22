data "dokku_git_property" "example" {
  app      = "node-js-app"
  property = "deploy-branch"
}
