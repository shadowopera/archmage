# Archmage

Archmage runs two pipelines over the same source files: `archmage export` produces runtime configs, and `archmage struct` generates strongly typed C# / Go code. Additionally, `archmage enum` generates typed enum code for 12 languages — C#, Go, Java, Python, TypeScript, JavaScript, Lua, GDScript, C++, Rust, PHP, and Protocol Buffers.

- **Website** — <https://shadop.dev/archmage/>
- **Documentation** — <https://docs.shadop.dev/archmage/>
- **Download** — [Releases](https://github.com/shadowopera/archmage/releases)

## About this repository

Archmage is closed source. This repository does not contain the source code.

- [Issues](https://github.com/shadowopera/archmage/issues) — Bugs and feature requests
- [Discussions](https://github.com/shadowopera/archmage/discussions) — Community and support

## Getting started

Download the latest release for your platform from [Releases](https://github.com/shadowopera/archmage/releases), put `archmage` on your `PATH`, then scaffold a project:

```bash
archmage init --language cs     # or: --language go
```

`init` writes enum definitions, sample configurations, and ready-to-run scripts:

```bash
scripts/export.sh      # export data
scripts/enum.sh        # generate enum code
scripts/struct.sh      # generate struct definitions
scripts/all.sh         # run the three scripts above in order
```

On Windows, run the `.ps1` counterparts. See the generated `ARCHMAGE.md`, or [the documentation](https://docs.shadop.dev/archmage/).

## Free and paid use

Archmage is free to download and use, and no license key is required. Free use is capped:

| | Free | Startup / Business |
|---|---|---|
| **Enum code generation** | **Unlimited** | Unlimited |
| Excel worksheets & YAML / JSON / … files <sup>1</sup> | 10 | Unlimited |
| Config entries per regular table | 50 | Unlimited |
| Generated config code files | 10 | Unlimited |

<sup>1</sup> Counted together: each Excel worksheet, and each tree-structured data file — `.yaml`, `.json`, `.json5`, `.js`, `.toml`, `.xml`.

## SDKs

- **C#** — [sdk-cs](https://github.com/shadowopera/sdk-cs) · [Documentation](https://docs.shadop.dev/archmage/overview-cs/readme/)
- **Go** — [sdk-go](https://github.com/shadowopera/sdk-go) · [Documentation](https://docs.shadop.dev/archmage/overview-go/readme/)

The Go import path is `shadop.dev/pkg/sdk-go`.

## License

Archmage is proprietary software. See [LICENSE](LICENSE).
