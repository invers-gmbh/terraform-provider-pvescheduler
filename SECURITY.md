# Security Policy

## Supported versions

This provider is pre-1.0. Only the most recent release receives security fixes.
Older tags are not patched.

## Reporting a vulnerability

Please report security issues privately to [oss@invers.com](mailto:oss@invers.com).
Do not open a public issue or pull request for a suspected vulnerability, and do
not include working credentials in your report.

Include as much of the following as you can:

- The provider version and the Terraform version you were running
- A minimal configuration that reproduces the problem
- What an attacker could achieve, and what access they would need first
- Relevant logs or output, with API tokens and passwords removed

## What to expect

- We will tell you whether we accept the report, how we assess its severity, and
  when we expect to ship a fix.
- We will keep you updated while we work on it, and credit you in the release
  notes unless you would rather stay anonymous.
- Please give us a reasonable opportunity to publish a fix before disclosing the
  issue publicly.

## Scope

In scope: the provider code in this repository, in particular its handling of
PVE credentials, its TLS configuration, and what it writes to Terraform state.

Out of scope: vulnerabilities in Proxmox VE, in Terraform itself, or in
third-party dependencies. Please report those to their respective maintainers.
If a dependency issue affects this provider specifically, we do want to hear
about it.

## Handling of credentials and state

The provider reads PVE credentials from provider configuration or from the
`PROXMOX_VE_*` environment variables, and holds them in memory only for the
duration of a Terraform run. It does not write them to Terraform state.

Selected node names and their utilization figures at time of placement are
written to state. Terraform state should be treated as sensitive and stored
accordingly.

Setting `insecure_skip_verify` (or `PROXMOX_VE_INSECURE=true`) disables TLS
certificate verification for all calls to the PVE API. It is intended for
development against self-signed certificates and should not be used against
production clusters.
