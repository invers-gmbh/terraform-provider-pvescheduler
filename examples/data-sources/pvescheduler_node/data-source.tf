# pvescheduler_node is the ephemeral variant of pvescheduler_placement: it
# re-evaluates on every plan and locks nothing. Use it to inspect current node
# scores, not to pin a VM to a node.
#
# Scheduling is restricted to the provider block's nodes allowlist, if set.
data "pvescheduler_node" "best" {
  # Exclude nodes that are reserved or under maintenance (optional).
  exclude = ["pve-storage-01"]

  # Tune scoring weights (defaults: memory_weight = 0.7, cpu_weight = 0.3).
  memory_weight = 0.8
  cpu_weight    = 0.2
}

output "current_best_node" {
  value = data.pvescheduler_node.best.node_name
}
