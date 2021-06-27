---
layout: "virtualbox"
page_title: "Virtualbox: NAT Net"
description: |
    Manages a Virtualbox NAT network
---

# virtualbox_natnetwork

Creates and manages a Virtualbox NAT network

## Example Usage

```hcl
resource "virtualbox_natnetwork" "default_net" {
  name    = "NAT Network"
  dhcp    = true
  network = "192.168.56.1/24"
}
```

## Argument Reference

The following arguments are supported:

- `name` - (Required) The name of the virtual NAT network.
  box).
- `dhcp` - (Optional) If DHCP is used for the network.
- `network` - (Required) The CIDR range used for the network.
