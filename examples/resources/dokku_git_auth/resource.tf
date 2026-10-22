# Configure netrc credentials for a git host
resource "dokku_git_auth" "example" {
  host     = "github.com"
  username = "deploy-bot"
  password = "ghp_examplepat"
}
