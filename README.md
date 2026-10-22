# terraform-provider-dokku

A Terraform provider for [Dokku](https://dokku.com), built on the [docket](https://github.com/dokku/docket) SDK. Every resource and data source is one docket task type, generated at runtime from docket's task catalog, so the provider supports exactly what the docket release it is built with supports. That release is the `github.com/dokku/docket` version in `go.mod`.

The provider documentation is in [docs](docs/index.md), and is published to the Terraform Registry.

## Requirements

- Terraform
- Go 1.26 or later, to build the provider
- A Dokku server, or Dokku on the machine Terraform runs on

## Building

```shell
go build ./...
go install .
```

To use a local build, point Terraform at it with a `dev_overrides` block in `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "dokku/dokku" = "/path/to/your/go/bin"
  }
  direct {}
}
```

## Developing

`go generate ./...` rewrites `examples/` from the examples in docket's catalog, then renders `docs/` with tfplugindocs. CI fails when either is out of date.

`go test ./...` runs the unit tests. They need the `terraform` binary but no server: every command goes to the stateful fake dokku in `github.com/dokku/docket/sdk/dokkutest`.

`make testacc` runs the acceptance tests against a real Dokku server. The provider reads its settings from the environment, so set `DOKKU_HOST` to reach a remote server, or `DOKKU_SUDO=true` to run `dokku` locally through `sudo`. Tests for a plugin skip when the plugin is not installed. CI installs the server with docket's `setup-dokku` action, pinned at the docket version in `go.mod`, so it runs the Dokku and plugin versions that docket release was tested against.

## License

MIT
