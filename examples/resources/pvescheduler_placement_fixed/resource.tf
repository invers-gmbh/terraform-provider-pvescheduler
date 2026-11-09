# pvescheduler_placement_fixed pins a VM to one explicitly named node. No
# scoring happens; the node only has to exist in the cluster at creation time.
# Changing node_name replaces the resource.
resource "pvescheduler_placement_fixed" "vm" {
  node_name = "pve01"
}

# Feed the decision into your VM resource, for example:
#
#   node_name = pvescheduler_placement_fixed.vm.node_name
#
output "pinned_node" {
  value = pvescheduler_placement_fixed.vm.node_name
}
