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

## Verifying a release

Releases are signed with the key in [.github/signing-key.asc](.github/signing-key.asc), fingerprint `7F8A 213F 3370 CEF0 69D9 813A 02A8 D8D8 D47E 4BC9`. Terraform checks the signature on `terraform init`. To check a release by hand, import the key and verify the checksums file against its signature:

```shell
gpg --import .github/signing-key.asc
gpg --verify terraform-provider-dokku_<VERSION>_SHA256SUMS.sig terraform-provider-dokku_<VERSION>_SHA256SUMS
shasum -a 256 -c terraform-provider-dokku_<VERSION>_SHA256SUMS
```

## License

MIT
