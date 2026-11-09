# pvescheduler_placement_dynamic picks the least-loaded node, optionally
# restricted to one zone of the provider topology, and locks the result in
# state. Changing zone, exclude or a weight replaces the resource.
resource "pvescheduler_placement_dynamic" "placement" {
  # Must be a key of the provider topology (optional).
  zone = "par-a"

  # Exclude nodes that are reserved or under maintenance (optional).
  exclude = ["pve-storage-01"]

  # Tune scoring weights (defaults: memory_weight = 0.7, cpu_weight = 0.3).
  memory_weight = 0.7
  cpu_weight    = 0.3
}

# Spread replicas across zones, one placement per zone.
locals {
  zones = ["par-a", "par-b", "par-c"]
}

resource "pvescheduler_placement_dynamic" "replica" {
  count = length(local.zones)
  zone  = local.zones[count.index]
}

output "chosen_node" {
  value = pvescheduler_placement_dynamic.placement.node_name
}

output "replica_nodes" {
  value = pvescheduler_placement_dynamic.replica[*].node_name
}
