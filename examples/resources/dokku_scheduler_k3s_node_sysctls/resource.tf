# Set node sysctls for every unprofiled node
resource "dokku_scheduler_k3s_node_sysctls" "example" {
  global = true
  sysctls = {
    "vm.max_map_count" = "262144"
    "vm.swappiness"    = "10"
  }
}

resource "dokku_scheduler_k3s_profile" "edge_workers" {
  name = "edge-workers"
  role = "worker"
}

# Replace the node sysctls of a node profile
resource "dokku_scheduler_k3s_node_sysctls" "example_2" {
  profile = "edge-workers"
  sysctls = {
    "vm.swappiness" = "60"
  }
  depends_on = [dokku_scheduler_k3s_profile.edge_workers]
}
