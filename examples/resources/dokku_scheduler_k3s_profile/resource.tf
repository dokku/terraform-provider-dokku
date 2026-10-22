# Define a worker profile with kubelet args
resource "dokku_scheduler_k3s_profile" "example" {
  name         = "edge-pool"
  role         = "worker"
  kubelet_args = ["max-pods=64", "eviction-hard=memory.available<5%"]
}

# Define a tainted server profile that accepts unknown hosts
resource "dokku_scheduler_k3s_profile" "example_2" {
  name                = "control-plane"
  role                = "server"
  taint_scheduling    = true
  allow_unknown_hosts = true
}
