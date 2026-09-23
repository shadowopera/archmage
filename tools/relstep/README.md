# relstep

relstep tracks the progress of a multi-step release in `release.json`, so an
interrupted release script can resume where it left off.

relstep only records which steps are done, and tells the script which step
comes next.

## Install

```bash
go install github.com/shadowopera/archmage/tools/relstep@latest
```

## Usage

```bash
relstep next -steps a,b,c [version]   # print the next pending step, or DONE
relstep mark -steps a,b,c <step>      # mark a step as done
relstep version                       # print the release version
```

- `next` creates `release.json` in the current directory on the first run.
  You must pass a version the first time. A leading `v` or `V` is removed.
- Steps run in the order given by `-steps`. They must be marked done in that
  order.
- The steps in `-steps` must match the steps in `release.json` exactly.
  If you change the step list during a release, relstep reports an error.
- On error, relstep writes the message to stderr and exits with status 1.

`release.json` looks like this:

```json
{
	"version": "1.2.3",
	"steps": {
		"a": 1,
		"b": 0,
		"c": 0
	}
}
```

Delete `release.json` after a release is finished to start the next one.

## Example

```bash
STEP_LIST=(
    checkVersion
    runTests
    tagRelease
)
STEPS=$(IFS=,; echo "${STEP_LIST[*]}")

VERSION_ARG="${1:-}"
while true; do
    STEP=$(relstep next -steps "$STEPS" $VERSION_ARG) || exit 1
    VERSION_ARG=""
    [[ "$STEP" == "DONE" ]] && break

    case "$STEP" in
        checkVersion) ... ;;
        runTests)     ... ;;
        tagRelease)   ... ;;
    esac
    relstep mark -steps "$STEPS" "$STEP" || exit 1
done
```
