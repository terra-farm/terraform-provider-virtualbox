---
layout: "virtualbox"
page_title: "Provider: Virtualbox"
sidebar_current: "docs-virtualbox-index"
description: |-
  Virtualbox provider for Terraform.
---

# Virtualbox Provider

The Virtualbox provider for Terraform allows to manage local virtualbox machines
using Terraform. The main purpose of this provider is to make you familiar with
Terraform and provisioning machines, without leaving your machine, therefore
saving you costs. However remember that your local environment might differ
from a cloud provider.

## Example Usage

```hcl
terraform {
  required_providers {
    virtualbox = {
      source = "terra-farm/virtualbox"
      version = "<latest-tag>"
    }
  }
}

# There are currently no configuration options for the provider itself.

resource "virtualbox_vm" "node" {
  count     = 2
  name      = format("node-%02d", count.index + 1)
  image     = "https://app.vagrantup.com/ubuntu/boxes/bionic64/versions/20180903.0.0/providers/virtualbox.box"
  cpus      = 2
  memory    = "512 mib"

  network_adapter {
    type           = "hostonly"
    host_interface = "vboxnet1"
  }
}

output "IPAddr" {
  value = element(virtualbox_vm.node.*.network_adapter.0.ipv4_address, 1)
}

output "IPAddr_2" {
  value = element(virtualbox_vm.node.*.network_adapter.0.ipv4_address, 2)
}
```
