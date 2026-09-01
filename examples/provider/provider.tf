# Credentials can also be supplied via the PROXMOX_VE_ENDPOINT and
# PROXMOX_VE_API_TOKEN environment variables, in which case those arguments can
# be omitted here. Username and password auth is available as an alternative to
# api_token via the username and password arguments.
provider "pvescheduler" {
  endpoint  = "https://pve.example.com:8006"
  api_token = "root@pam!mytoken=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

  # Global allowlist of nodes eligible for scheduling (optional).
  # If omitted, all online nodes are considered.
  nodes = ["pve01", "pve02", "pve03"]
}
