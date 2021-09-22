# How to build from source

1. `git clone git@github.com:terra-farm/terraform-provider-virtualbox.git`
1. `cd terraform-provider-virtualbox`
1. `go build`
1. `mv terraform-provider-virtualbox examples/`
1. `cd examples/`
1. `terraform init`
1. `terraform plan`
1. `terraform apply`

# Adding documentation

The website is hosted by the official [Terraform Registry](https://registry.terraform.io/providers/terra-farm/virtualbox/latest/docs).
The source for the documentation is located in the `/website` directory. It follows the standard provider
documentation format.

# Ask the community

If you have a change which you think will benefit the project, ask. This can be either done as a new issue, or by creating a PR with the changes included.

# Creating a release

To create a new release for the Terraform Registry, a maintainer only needs to create a new release
in the [Github UI](https://github.com/terra-farm/terraform-provider-virtualbox/releases/new).

This will automatically publish the release to the Terraform Registry assuming the `release` Github
Action passes.

## Updating signing certificate

Please follow the [GPG Signing Key](https://learn.hashicorp.com/tutorials/terraform/provider-release-publish?in=terraform/providers#generate-gpg-signing-key)
guide in the official Terraform Documentation. We try to follow the recommended guides as closely as possible.
