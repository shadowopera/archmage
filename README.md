# Archmage

![Archmage](images/archmage.jpg)

**Archmage** is a configuration solution for game development. It brings together specifications on how to structure config data, define fields, and fill in values; pipelines that export runtime data and generate strongly-typed code; multi-language SDKs for loading and accessing that data at runtime; and a collaborative editing workflow for teams.

- **Home** — <https://shadop.dev/archmage/>
- **Quickstart** — <https://docs.shadop.dev/archmage/guides/quickstart/>
- **Documentation** — <https://docs.shadop.dev/>
- **Download** — [Releases](https://github.com/shadowopera/archmage/releases)

## Features

### Define

You decide the shape of each config data set and the data type of each field.

- [Regular Table](https://docs.shadop.dev/archmage/specs/spreadsheet/regular-table/) — standard table layout: config entries extend horizontally; fields extend vertically.
- [Property Sheet](https://docs.shadop.dev/archmage/specs/spreadsheet/property-sheet/) — presents fields as key–value pairs, designed for global configs such as feature toggles.
- [Index Table](https://docs.shadop.dev/archmage/specs/spreadsheet/index-table/) — selects which worksheets take part in the pipeline; useful for spreadsheet files containing multiple worksheets.
- [Tree-Structured Data](https://docs.shadop.dev/archmage/specs/tree/overview/) — freeform nested data (YAML/JSON/...) for configs that do not fit neatly into rows and columns. When the structure satisfies certain criteria, Archmage treats it as a tree-backed regular table.
- [Virtual Table](https://docs.shadop.dev/archmage/specs/spreadsheet/virtual-table/) — allows several regular tables to act as a unified logical table.
- [Data Types](https://docs.shadop.dev/archmage/specs/types/overview/) — a rich, unified type system for config fields, including 50+ data types in 7 categories.
- [Enum Definition](https://docs.shadop.dev/archmage/specs/enum/enum-definition/) — backs enum field definitions, value filling, and typed code generation for 12+ languages.

### Fill In

A field's data type is not just a label. It comes with carefully designed fill-in settings that make data input easier.

- [Basic Types](https://docs.shadop.dev/archmage/specs/types/basic/) — the foundational types: integers, floating-point numbers, string, boolean, enum, datetime, duration, cross-table reference, localization, file path, and color.
- [Passive Types](https://docs.shadop.dev/archmage/specs/types/passive/) — require no manual input; values are derived automatically from references or context.
- [Compact Types](https://docs.shadop.dev/archmage/specs/types/compact/) — encode structured data into a single cell using a concise text format.
- [Multi-Column Types](https://docs.shadop.dev/archmage/specs/types/multi-column/) — span multiple columns to form a single logical field, with each column holding only one element.
- [Subtable Types](https://docs.shadop.dev/archmage/specs/types/subtable/) — each defines an embedded subtable for each config entry within a regular table.
- [Behavior Types](https://docs.shadop.dev/archmage/specs/types/behavior/) — act as functional directives that guide the processing carried out by the pipelines.
- [Non-Leaf Types](https://docs.shadop.dev/archmage/specs/types/non-leaf/) — apply exclusively to `[]` or `{}` tree-structured data nodes, determining how Archmage processes these nodes and their children.

### Export & Generate

Data export and code generation run as independent pipelines over the same config source.

- [Data Export](https://docs.shadop.dev/archmage/specs/workflow/data-export/) — parses and validates config files and outputs the runtime data. A rich set of validation rules is available out of the box.
- [Code Generation](https://docs.shadop.dev/archmage/specs/workflow/code-generation/) — renders enum definitions and config structures into strongly-typed code, and emits a `ConfigAtlas`.
- [Filtering](https://docs.shadop.dev/archmage/specs/workflow/filtering-mechanisms/) — decides, together with your build flags, which columns, rows, fields, entries, and nodes are included.
- [L10n Pipeline](https://docs.shadop.dev/archmage/specs/workflow/l10n-pipeline/) — collects localizable strings during export and aggregates them into files ready for translation.
- [Readability & AI](https://docs.shadop.dev/archmage/specs/workflow/readable/) — turns opaque numbers into text that is readable by humans and AI agents, alongside a schema describing every field.

### Integrate

Since runtime data and generated config types originate from the same source, their structures naturally align. Once the data and the types are in place, you bring them together with a runtime [SDK](#sdks). The SDK parses each file using the corresponding config type, populates your `ConfigAtlas`, and resolves cross-table references. You can also set up a few global callbacks to make coding easier.

### Leverage

Your game logic works with the `ConfigAtlas`. Every field is typed, and you enjoy the convenience built into the types, especially those the Archmage SDK provides. Take `spell.Rune`: it is a [`ref`](https://docs.shadop.dev/archmage/specs/types/basic/#ref) field, and `spell.Rune.Ref` gives you the rune's config entry directly, ready to use in your code. No ID lookup, no manual wiring.

### Collaborate

- Edit together live in [Google Sheets](https://docs.shadop.dev/archmage/specs/workflow/team-collaboration/) — no file locks, no binary merge conflicts.
- Those sheets are then pulled down into your repository as spreadsheet files, fitting into your team's existing version control workflow.

## Supported input file types

- **Spreadsheets**  — `.xlsx`, `.xlsm`, `.csv`
- **Tree-structured data** — `.yaml` or `.yml`, `.json`, `.json5`, `.js`, `.toml`, `.xml`

## SDKs

- **C#** — [GitHub](https://github.com/shadowopera/sdk-cs) · [Overview](https://docs.shadop.dev/archmage/overview-cs/sdk-cs/) with Documentation about SDK, Unity, Unity Editor, Generated Code
- **Go** — [GitHub](https://github.com/shadowopera/sdk-go) · [Overview](https://docs.shadop.dev/archmage/overview-go/sdk-go/) · [Documentation](https://pkg.go.dev/shadop.dev/pkg/sdk-go/archmage)

## About this repository

Archmage is closed source. This repository does not contain its source code.

- [Issues](https://github.com/shadowopera/archmage/issues) — Bug reports and feature requests
- [Discussions](https://github.com/shadowopera/archmage/discussions) — Community

## Free and paid use

Archmage is free to download and use. Free use is restricted. See [Pricing](https://shadop.dev/archmage/pricing/) for details.

## Cross-Language Enum Code Generation, Free and Unlimited

Available to everyone, `archmage enum` operates completely outside the paid tiers, serving as a 100% unrestricted tool. One [enum definition](https://docs.shadop.dev/archmage/specs/enum/enum-definition/), one set of enum capabilities — Archmage carries them into 12+ programming languages, each rendered in that language's own idioms and type system.

### Definition

- **Flexible Underlying Types**: `int`, `int8/16/32/64`, `uint`, `uint8/16/32/64`.
- **Conditional Generation**: Scenario flags for filtering enum types and enum items.
- **Bitflags**: Bitwise-combinable enum types.
- **Auto-Numbering**: `iota` auto-assigns values: 0, 1, 2, 3...; for bitflags: 0, 1, 2, 4, 8...
- **External Types**: Same enum type: native in one language, external in another.

### Generated Code

- **Named Types**: Each enum type is defined as a named type.
- **Typed Constants**: Each enum item is defined as a typed constant.
- **Underlying Types**: From `int8` to `uint64`; parsing enforces the value range.
- **Enum ↔ String Conversion**: In both directions — by default the item name, overridable in the definition.
- **Multiple Parsing Formats**: Item name in any case, TypeName(n), decimal, and hex are all supported.
- **Safe Parsing**: Failures trigger explicit errors or exceptions instead of silent zero values.
- **JSON Integer Serialization**: Enum values serialize as integers in JSON.
- **Sentinel**: Boundary markers such as `Count`, excluded from the valid values.
- **Duplicate Values**: Multiple items can share the same integer value; all names remain valid for parsing.
- **Bitflag Enums**: Parses and formats flag combinations, e.g., `"frost, fire"`.
- **AllValues**: Returns the full set of valid enum values as an ordered list.
- **IsValid**: Checks if a given value corresponds to a defined enum item.

## License

Archmage is proprietary software. See [LICENSE](LICENSE).
