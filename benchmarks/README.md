# magebench

A data-export and code-generation benchmark for [Archmage](https://shadop.dev/archmage/).

It is a black-box benchmark: it generates a large, realistic config corpus and
then drives the `archmage` binary installed on this machine against it. Nothing
here links against Archmage's source.

## Two steps

Generating the corpus and measuring Archmage are separate on purpose. The
corpus is fixed input; regenerating it between runs would put the generator's
own cost inside the measurement.

```bash
./step1.sh    # step 1 — write the corpus (~95 MB, a couple of seconds)
./step2.sh    # step 2 — measure Archmage against it
```

`step2.sh` refuses to run when the corpus is missing, has a different shape
than the one it expects, or was written by a different version of the
generator, rather than quietly reporting numbers from a smoke-sized or stale
corpus. The manifest records a fingerprint of the generator sources and
`go.sum`, so any change to the generator means running `./step1.sh` again.

```bash
./step2.sh          # archmage struct targets C# (default)
./step2.sh go       # archmage struct targets Go
./step2.sh cs -benchtime=10x -count=5
./step2.sh -benchtime=10x    # language omitted: C#
```

`step1.sh` wipes the corpus directory before writing. With `-out`, it refuses
a directory that holds anything besides an earlier corpus.

## Where things live

`/Volumes/Fuzz` is probed at startup. When that volume is mounted, both the
corpus and everything Archmage writes go to `/Volumes/Fuzz/archmage_bench/`;
otherwise they stay in this directory.

```
<root>/testdata/configs/   130 config files + l10n.xlsx
<root>/testdata/enums/     3 enum definition files
<root>/testdata/manifest.json
<root>/output/export/      archmage export output
<root>/output/struct/      archmage struct output
<root>/bench-report.md     summary, written after a successful run
```

The benchmark measures many separate runs of archmage, and before each one it
deletes the output directory — that deletion happens before the clock starts,
so it doesn't count toward the measured time. Archmage skips rewriting a file
whose content has not changed, so without this reset every run after the
first would find its output already there and finish artificially fast.

## The corpus

100 config files, 3000 config entries each, 20 logical fields each — 300,000
config entries and 6,000,000 field slots. The shape is fixed by a seed, so the
same command always produces the same bytes.

| | |
|---|---|
| Formats | 60 xlsx, 10 csv, 15 yaml, 15 json (60:10:15:15) |
| xlsx / csv | regular tables; one worksheet per xlsx file, no index table |
| yaml / json | tree-backed regular tables, each with a companion `.demo` file holding its meta nodes |
| Config IDs | `int64` throughout |
| Columns | about 26 per spreadsheet table — 20 logical fields, of which the multi-column ones span two to six columns each |

Every non-subtable data type is represented. Types are drawn from a weighted
distribution that leans the way real game config leans — mostly scalars, with
containers as seasoning — and a coverage pass guarantees each type lands in at
least two tables regardless of how the sampling falls.

Deliberately out of scope:

- **Subtable types** (`**[]`, `**[N]`, `**map`), excluded by request.
- **Validation and filtering options.** No `interval`, `regexp`, `unique`,
  `tags`, `xflags`. The only option in the corpus is `e=`, which `bitflags`
  columns require. This measures parse, transform and write, not the option
  machinery.
- **Behavior types other than `anchor`.** No `assert`, `condition`, `switch`
  or `placeholder`.
- **`path` root checking.** Paths are plain strings; no `root` option and no
  `ARCHMAGE_PATH_ROOT` in the environment.
- **`backref`.** It binds a single referrer, which random cross-table
  references overshoot immediately. `backref-n` is used instead, paired with a
  source table that is forced to reference back.

Some shaping choices worth knowing about:

- **Sparsity.** About 60% of fields declare a default export value, and about
  15% of those cells are left blank — real config tables are sparse. Roughly
  6.3 million cells and tree nodes end up populated.
- **l10n.** Four tables (one per format) carry a single localizable field, with
  a quarter of those cells blank, and one value in ten is a `{{key}}` reference
  into `l10n.xlsx`. That lands the l10n aggregate near ten thousand strings —
  a realistic size, and deliberately not the dominant cost of the run.
- **References.** Around 7% of fields are `ref@`, targeting any of the 100
  tables regardless of format. One table in five carries an `anchor`, and half
  the references to those tables are written as anchor names rather than
  numbers.
- **Multi-column headers** use the `<1` lookback operator rather than merged
  cells. It is the documented equivalent and reads identically in xlsx and csv.

## Method

Both cases run `archmage` as a subprocess, using the
minimal realistic flag set — no `--input-roots`, `--semver`, `--timezone`, or
filtering flags:

```
archmage export --banner -o <out> -e "<td>/enums/*.yaml" \
    --l10n <td>/configs/l10n.xlsx \
    "<td>/configs/*.xlsx" "*.csv" "*.yaml" "*.json"

archmage struct --banner -o <out> -t json-cs --namespace Conf \
    <same inputs>
```

With `./step2.sh go` the second command becomes `-t json-go --namespace conf`
— the Go template takes a package name where the C# one takes a namespace.

Each case warms up once, before its first round, with one full untimed run —
then spins every core for two seconds so the timed runs start with the CPU
already at speed. Later rounds go straight to timing.
The default is five iterations per case over three rounds, which suits
`benchstat`.

Alongside `ns/op`, the export case reports `entries/s` and `values/s`. The
struct case generates code and touches no config data, so it reports
`fields/s` instead — logical fields across all tables. The benchmark also
writes `bench-report.md` with the corpus shape, the machine, the Archmage
build, and the measured throughput.

## Layout

| File | |
|---|---|
| `main.go` | generator entry point and flags |
| `plan.go` | corpus planning: tables, field sampling, type coverage |
| `schema.go` | the `Field` model and the basic-type value generators |
| `fields.go` | the field catalogue: every type and how it is filled |
| `corpus.go` | tables, references, anchors, shared value pools |
| `enums.go` | enum definition planning and rendering |
| `write_sheet.go` | xlsx and csv writers |
| `write_tree.go` | yaml and json writers, including the demo files |
| `write_aux.go` | l10n.xlsx, enum files, manifest |
| `om.go` | ordered map, so tree files come out byte-identical |
| `bench_test.go` | the two benchmark cases and the harness |
| `report_test.go` | `bench-report.md` |
