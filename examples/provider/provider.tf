terraform {
  required_providers {
    dokku = {
      source = "dokku/dokku"
    }
  }
}

provider "dokku" {
  host              = "dokku@dokku.example.com"
  ssh_identity_file = "~/.ssh/dokku_deploy"
}
