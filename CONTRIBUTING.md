# How to build from source

1. git clone https://github.com/terra-farm/terraform-provider-virtualbox
1. cd terraform-provider-virtualbox
1. go build
1. mv terraform-provider-virtualbox examples/
1. cd examples/
1. terraform init
1. terraform plan
1. terraform apply

# Adding documentation

The website is built with [Antora](https://antora.org/) with content in [Asciidoc](http://asciidoc.org/) rather than Markdown because of its more extensive tag set.

All content lives under the [modules](modules) subfolder in this repository. In here, there are 2 subfolders:

* `ROOT`: General documentation regarding this provider
* `reference`: documentation for all resources and data sources.

You do not have to do anything special if you change existing documentation. If you want to create new pages
for the site, create a new file with extension `.adoc` and add a cross-reference to the file `nav.adoc`.
`nav.adoc` represents the table of contents. Position the newly created file at the correct place in the 
table of contents.

*NOTE:* When documentation changes are integrated on the `master` branch, these will not become visible on the
website. A rebuilt of the [master site](https://github.com/terra-farm/terra-farm.github.io) is needed to pull 
the documentation changes for each provider.

# Creating a release

To create and publish a new release on Github, a committer only needs to check one item:

* Verify that the commit for the release builds succesfully on [Travis CI](https://travis-ci.org/terra-farm/terraform-provider-virtualbox).

If the commit builds correctly, then tag this commit as version `vX.Y.Z`. Make sure the version number
starts with the lowercase `v`. Push the tag to the Github remote and Travis will build again but also
publish the binaries as a release on Github.
