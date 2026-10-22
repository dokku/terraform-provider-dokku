# Add a deploy key
resource "dokku_ssh_key" "example" {
  name = "deploy-bot"
  key  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINioKrRalhe/VF8s43pjp8jpl6LGwv6tF0F5FvKPjUer deploy-bot"
}
