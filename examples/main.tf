terraform {
  required_providers {
    pvescheduler = {
      source  = "invers-gmbh/pvescheduler"
      version = "0.1.0"
    }
  }
}

# Credentials can also be supplied via PROXMOX_VE_ENDPOINT, PROXMOX_VE_API_TOKEN,
# and PROXMOX_VE_INSECURE env vars, in which case this block can be left empty.
provider "pvescheduler" {
  endpoint             = "https://pve.example.com:8006"
  api_token            = "root@pam!mytoken=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  username             = ""
  passwrod             = ""
  insecure_skip_verify = true
}

# pvescheduler_placement picks the best node once and locks the result in state.
# Re-run node selection with: terraform state rm pvescheduler_placement.vm
resource "pvescheduler_placement" "vm" {
  # Restrict scheduling to a known-good subset of nodes (optional).
  nodes = ["pve01", "pve02", "pve03"]

  # Exclude nodes that are reserved or under maintenance (optional).
  exclude = ["pve-storage-01"]

  # Tune scoring weights (defaults: memory_weight=0.7, cpu_weight=0.3).
  memory_weight = 0.7
  cpu_weight    = 0.3
}

output "chosen_node" {
  value = pvescheduler_placement.vm.node_name
}

output "chosen_node_memory_pct" {
  value = pvescheduler_placement.vm.memory_usage_pct
}

output "chosen_node_cpu_pct" {
  value = pvescheduler_placement.vm.cpu_usage_pct
}

# pvescheduler_node is the ephemeral variant: re-evaluates on every plan.
# Use it when you want to inspect node scores without locking anything.
data "pvescheduler_node" "best" {
  nodes         = ["pve01", "pve02", "pve03"]
  exclude       = ["pve-storage-01"]
  memory_weight = 0.8
  cpu_weight    = 0.2
}

output "current_best_node" {
  value = data.pvescheduler_node.best.node_name
}
