# Archmage

![Archmage](images/archmage.jpg)

**Archmage** is a configuration solution for game development: specifications for how to structure config data, define fields, and fill in each value; pipelines that export runtime data and generate strongly typed code; multi-language SDKs for accessing that data at runtime; and a collaborative editing workflow for teams.

- **Home** — <https://shadop.dev/archmage/>
- **Quickstart** — <https://docs.shadop.dev/archmage/guides/quickstart/>
- **Documentation** — <https://docs.shadop.dev/>
- **Download** — [Releases](https://github.com/shadowopera/archmage/releases)

## Features

### Define

You decide the layout of each set of config data and the data type of each field, as preferred.

- [Regular Table](https://docs.shadop.dev/archmage/specs/spreadsheet/regular-table/) — standard table layout: config entries extend horizontally; fields extend vertically.
- [Property Sheet](https://docs.shadop.dev/archmage/specs/spreadsheet/property-sheet/) — presents fields as key–value pairs, designed for global configs such as feature toggles.
- [Index Table](https://docs.shadop.dev/archmage/specs/spreadsheet/index-table/) — selects which worksheets take part in the pipeline; useful for spreadsheet files containing multiple worksheets.
- [Tree-Structured Data](https://docs.shadop.dev/archmage/specs/tree/overview/) — freeform nested data (YAML/JSON/...) for configs that do not fit neatly into rows and columns. When the structure satisfies certain criteria, Archmage treats it as a tree-backed regular table.
- [Virtual Table](https://docs.shadop.dev/archmage/specs/spreadsheet/virtual-table/) — allows several regular tables to act as a unified logical table.
- [Enum Definition](https://docs.shadop.dev/archmage/specs/enum/enum-definition/) — backs field definitions, value filling, and typed code generation for 12 languages.

### Fill In

A field’s data type is not just a label. It comes with carefully designed fill-in settings that make data input easier.

- [Basic Types](https://docs.shadop.dev/archmage/specs/types/basic/) — the foundational types: integers, floating-point numbers, string, boolean, enum, datetime, duration, cross-table reference, localization, file path, and color
- [Passive Types](https://docs.shadop.dev/archmage/specs/types/passive/) — require no manual input; values are derived automatically from references or context
- [Compact Types](https://docs.shadop.dev/archmage/specs/types/compact/) — encode structured data into a single cell using a concise text format
- [Multi-Column Types](https://docs.shadop.dev/archmage/specs/types/multi-column/) — span multiple columns to form a single logical field, with each column holding only one element
- [Subtable Types](https://docs.shadop.dev/archmage/specs/types/subtable/) — define embedded subtables within a regular table
- [Behavior Types](https://docs.shadop.dev/archmage/specs/types/behavior/) — act as functional directives that guide the processing carried out by the pipelines
- [Non-Leaf Types](https://docs.shadop.dev/archmage/specs/types/non-leaf/) — apply exclusively to `[]` or `{}` tree-structured data nodes, determining how Archmage processes these nodes and their children

### Export & Generate

Data export and code generation run as independent pipelines over the same config source.

- [Data Export](https://docs.shadop.dev/archmage/specs/workflow/data-export/) — parses and validates config files and outputs the runtime data
- [Code Generation](https://docs.shadop.dev/archmage/specs/workflow/code-generation/) — renders config structures and enum definitions into strongly typed code
- [Filtering](https://docs.shadop.dev/archmage/specs/workflow/filtering-mechanisms/) — decides, together with your build flags, which columns, rows, fields, entries, and nodes are included
- [L10n Pipeline](https://docs.shadop.dev/archmage/specs/workflow/l10n-pipeline/) — collects localizable strings during export and aggregates them into files ready for translation
- [Readability & AI](https://docs.shadop.dev/archmage/specs/workflow/readable/) — turns opaque exported values into text that is readable by humans and AI alike, alongside a schema describing every field

### Integrate

Once runtime data is exported, your game loads it with a runtime [SDK](#sdks). The SDK parses files, assigns values to fields, and resolves references. You set up a few global callbacks.

### Collaborate

- Real-time, multi-person editing in Google Sheets — no file locks, no binary merge conflicts
- [Google Sheets](https://docs.shadop.dev/archmage/specs/workflow/team-collaboration/) pulled down into your repository as spreadsheet files, fitting into your team's existing version control workflow

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
