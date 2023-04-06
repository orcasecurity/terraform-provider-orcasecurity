Orca Security Terraform Provider
==================

- Documentation: https://registry.terraform.io/providers/orcasecurity/orcasecurity/latest/docs

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) 1.x
-	[Go](https://golang.org/doc/install) 1.18 (to build the provider plugin)

Building The Provider
---------------------

Clone repository to: `$GOPATH/src/github.com/orcasecurity/terraform-provider-orcasecurity`

```sh
$ mkdir -p $GOPATH/src/github.com/orcasecurity; 
$ cd $GOPATH/src/github.com/orcasecurity
$ git clone git@github.com:orcasecurity/terraform-provider-orcasecurity
```

Enter the provider directory and build the provider

```sh
$ cd $GOPATH/src/github.com/orcasecurity/terraform-provider-orcasecurity
$ make build
```

Using the provider
----------------------

See the [Orca Security Provider documentation](https://registry.terraform.io/providers/orcasecurity/orcasecurity/latest/docs) to get started using the Orca Security provider.

Developing the Provider
---------------------------

See [CONTRIBUTING.md](./CONTRIBUTING.md) for information about contributing to this project.
