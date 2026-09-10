# Archmage

Archmage is a configuration solution for game development: specifications for how to structure config data, define fields, and fill in each value; pipelines that export runtime data and generate strongly typed code; multi-language SDKs for accessing that data at runtime; and a collaborative editing workflow for teams.

- **Website** — <https://shadop.dev/archmage/>
- **Quickstart** — <https://docs.shadop.dev/archmage/guides/quickstart/>
- **Documentation** — <https://docs.shadop.dev/>
- **Download** — [Releases](https://github.com/shadowopera/archmage/releases)

## Features

### Define

You decide the layout of each set of config data and the data type of each field, as preferred.

- **Regular Table** — standard table layout: config entries extend horizontally; fields extend vertically.
- **Property Sheet** — presents fields as key–value pairs, designed for global configs such as feature toggles.
- **Index Table** — selects which worksheets take part in the pipeline; useful for spreadsheet files containing multiple worksheets.
- **Tree-Structured Data** — freeform nested data (YAML/JSON/...) for configs that do not fit neatly into rows and columns. When the structure satisfies certain criteria, Archmage treats it as a tree-backed regular table.
- **Virtual Table** — allows several regular tables to act as a unified logical table.
- **Enum Definition** — backs field definitions, value filling, and typed code generation for 12 languages.

### Fill In

A field’s data type is not just a label. It comes with carefully designed fill-in settings that make data input easier.

- **Basic Types** — the foundational types: integers, floating-point numbers, string, boolean, enum, datetime, duration, cross-table reference, localization, file path, and color
- **Passive Types** — require no manual input; values are derived automatically from references or context
- **Compact Types** — encode structured data into a single cell using a concise text format
- **Multi-Column Types** — span multiple columns to form a single logical field, with each column holding only one element
- **Subtable Types** — define embedded subtables within a regular table
- **Behavior Types** — act as functional directives that guide the processing carried out by the pipelines
- **Non-Leaf Types** — apply exclusively to `[]` or `{}` tree-structured data nodes, determining how Archmage processes these nodes and their children

### Export & Generate

Data export and code generation run as independent pipelines over the same config source.

- **Data Export** — parses and validates config files and outputs the runtime data
- **Code Generation** — renders config structures and enum definitions into strongly typed code
- **Filtering** — decides, together with your build flags, which columns, rows, fields, entries, and nodes are included
- **L10n Pipeline** — collects localizable strings during export and aggregates them into files ready for translation
- **Readability & AI** — turns opaque exported values into text that is readable by humans and AI alike, alongside a schema describing every field

### Integrate

Once runtime data is exported, your game loads it with a runtime SDK. The SDK parses files, assigns values to fields, and resolves references. You set up a few global callbacks.

### Collaborate

- Real-time, multi-person editing in Google Sheets — no file locks, no binary merge conflicts
- Google Sheets pulled down into your repository as spreadsheet files, fitting into your team's existing version control workflow

## SDKs

- **C#** — [sdk-cs](https://github.com/shadowopera/sdk-cs) · [Documentation](https://docs.shadop.dev/archmage/overview-cs/sdk-cs/)
- **Go** — [sdk-go](https://github.com/shadowopera/sdk-go) · [Documentation](https://docs.shadop.dev/archmage/overview-go/sdk-go/)

## About this repository

Archmage is closed source. This repository does not contain its source code.

- [Issues](https://github.com/shadowopera/archmage/issues) — Bug reports and feature requests
- [Discussions](https://github.com/shadowopera/archmage/discussions) — Community

## Free and paid use

Archmage is free to download and use, and no license key is required. Free use is restricted. See [Pricing](https://shadop.dev/archmage/pricing/) for details.

## License

Archmage is proprietary software. See [LICENSE](LICENSE).
