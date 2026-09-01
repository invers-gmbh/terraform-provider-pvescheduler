# pvescheduler_placement picks the least-loaded node once and locks the result
# in state, so VMs do not move on subsequent applies. To re-run node selection:
#
#   terraform state rm pvescheduler_placement.vm
#
# Scheduling is restricted to the provider block's nodes allowlist, if set.
resource "pvescheduler_placement" "vm" {
  # Exclude nodes that are reserved or under maintenance (optional).
  exclude = ["pve-storage-01"]

  # Tune scoring weights (defaults: memory_weight = 0.7, cpu_weight = 0.3).
  memory_weight = 0.7
  cpu_weight    = 0.3
}

# Feed the decision into your VM resource, for example:
#
#   node_name = pvescheduler_placement.vm.node_name
#
output "chosen_node" {
  value = pvescheduler_placement.vm.node_name
}

output "chosen_node_memory_pct" {
  value = pvescheduler_placement.vm.memory_usage_pct
}

output "chosen_node_cpu_pct" {
  value = pvescheduler_placement.vm.cpu_usage_pct
}
