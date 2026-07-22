# terraform-provider-pvescheduler

Selects the least-loaded Proxmox VE node for VM placement based on weighted CPU and memory utilization. The selected node is locked in Terraform state after the first apply so VMs don't move on subsequent runs. To force re-scheduling, remove the resource from state:

```bash
terraform state rm pvescheduler_placement.vm
```

> [!WARNING]
> Disclaimer: This provider is being developed for internal use and we decided to open-source it. Be aware that this only is a best effort of implementing automatic node-placement for PVE in Terraform and NOT a load-balancer.

## Example

```hcl
provider "pvescheduler" {
  # Falls back to PROXMOX_VE_ENDPOINT and PROXMOX_VE_API_TOKEN env vars.
  nodes = ["pve01", "pve02", "pve03"] # optional global allowlist
}

resource "pvescheduler_placement" "vm" {
  exclude       = ["pve-storage-01"]
  memory_weight = 0.7 # default
  cpu_weight    = 0.3 # default
}

# Use the selected node in your proxmox VM resource:
# node_name = pvescheduler_placement.vm.node_name
```

## Requirements

**Runtime**
- Proxmox VE 9.x
- Terraform >= 1.3

**Development**
- Go 1.26

The provider targets the Terraform Plugin Framework protocol v6. Terraform versions below 1.3 are not supported.

## Scoring

```
score = memory_weight * (mem / maxmem) + cpu_weight * cpu
```

Lowest score wins. Offline nodes are always skipped.

## License

[MIT](./LICENSE)
