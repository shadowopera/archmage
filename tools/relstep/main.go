// Relstep tracks the progress of a multi-step release in release.json,
// so an interrupted release script can resume where it left off.
//
// Usage:
//
//	relstep next -steps a,b,c [version]
//	relstep mark -steps a,b,c <step>
//	relstep version
//
// The next command creates release.json on the first run (version required)
// and prints the next pending step, or DONE when every step is marked.
// The mark command marks a step as done. The version command prints the
// release version.
//
// Steps are executed in the order given by -steps, and the step set must
// match the one recorded in release.json exactly.
package main

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"
)

const releaseFile = "release.json"

const usage = `usage:
  relstep next -steps a,b,c [version]
  relstep mark -steps a,b,c <step>
  relstep version`

var _semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)

type release struct {
	Version string         `json:"version"`
	Steps   map[string]int `json:"steps"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout, releaseFile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, w io.Writer, file string) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	switch cmd, args := args[0], args[1:]; cmd {
	case "next":
		steps, rest, err := parseSteps(cmd, args)
		if err != nil {
			return err
		}
		if len(rest) > 1 {
			return errors.New("usage: relstep next -steps a,b,c [version]")
		}
		version := ""
		if len(rest) == 1 {
			version = strings.TrimPrefix(strings.TrimPrefix(rest[0], "v"), "V")
			if !_semverRe.MatchString(version) {
				return errors.New("invalid semver: " + rest[0])
			}
		}
		return runNext(w, file, steps, version)
	case "mark":
		steps, rest, err := parseSteps(cmd, args)
		if err != nil {
			return err
		}
		if len(rest) != 1 {
			return errors.New("usage: relstep mark -steps a,b,c <step>")
		}
		return runMark(file, steps, rest[0])
	case "version":
		if len(args) != 0 {
			return errors.New("usage: relstep version")
		}
		rel, err := loadRelease(file)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, rel.Version)
		return nil
	default:
		return errors.New("unknown command: " + cmd + "\n" + usage)
	}
}

// parseSteps parses the -steps flag and returns the step names in order
// along with the remaining positional arguments.
func parseSteps(cmd string, args []string) ([]string, []string, error) {
	fs := flag.NewFlagSet("relstep "+cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	list := fs.String("steps", "", "comma-separated step names in execution order")
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}
	if *list == "" {
		return nil, nil, errors.New("missing -steps")
	}

	steps := strings.Split(*list, ",")
	seen := make(map[string]bool, len(steps))
	for _, s := range steps {
		if s == "" {
			return nil, nil, errors.New("empty step name in -steps: " + *list)
		}
		if seen[s] {
			return nil, nil, errors.New("duplicate step in -steps: " + s)
		}
		seen[s] = true
	}
	return steps, fs.Args(), nil
}

func runNext(w io.Writer, file string, steps []string, version string) error {
	if _, err := os.Stat(file); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if version == "" {
			return errors.New("release.json not found, please specify a version")
		}
		rel := &release{Version: version, Steps: make(map[string]int, len(steps))}
		for _, s := range steps {
			rel.Steps[s] = 0
		}
		if err := saveRelease(file, rel, steps); err != nil {
			return err
		}
	}

	rel, err := loadRelease(file)
	if err != nil {
		return err
	}
	if err := validate(rel, steps); err != nil {
		return err
	}

	if version != "" && version != rel.Version {
		if next(rel, steps) == "" {
			return errors.New("release " + rel.Version + " already completed, please delete release.json first")
		}
		return errors.New("version mismatch: release.json has " + rel.Version + ", got " + version)
	}

	if s := next(rel, steps); s != "" {
		fmt.Fprintln(w, s)
	} else {
		fmt.Fprintln(w, "DONE")
	}
	return nil
}

func runMark(file string, steps []string, step string) error {
	rel, err := loadRelease(file)
	if err != nil {
		return err
	}
	if err := validate(rel, steps); err != nil {
		return err
	}
	if _, ok := rel.Steps[step]; !ok {
		return errors.New("unknown step: " + step)
	}
	rel.Steps[step] = 1
	if err := validate(rel, steps); err != nil {
		return err
	}
	return saveRelease(file, rel, steps)
}

func loadRelease(file string) (*release, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var rel release
	if err := json.Unmarshal(data, &rel); err != nil {
		return nil, errors.New("failed to parse release.json: " + err.Error())
	}
	return &rel, nil
}

// saveRelease writes rel with its steps in the order given by steps.
func saveRelease(file string, rel *release, steps []string) error {
	var buf bytes.Buffer
	enc := jsontext.NewEncoder(&buf, jsontext.WithIndent("\t"))
	toks := []jsontext.Token{
		jsontext.BeginObject,
		jsontext.String("version"), jsontext.String(rel.Version),
		jsontext.String("steps"), jsontext.BeginObject,
	}
	for _, s := range steps {
		toks = append(toks, jsontext.String(s), jsontext.Int(int64(rel.Steps[s])))
	}
	toks = append(toks, jsontext.EndObject, jsontext.EndObject)
	for _, tok := range toks {
		if err := enc.WriteToken(tok); err != nil {
			return err
		}
	}
	return os.WriteFile(file, buf.Bytes(), 0644)
}

// validate checks the version format, that the recorded steps match steps
// exactly, and that done steps form a prefix of steps.
func validate(rel *release, steps []string) error {
	if !_semverRe.MatchString(rel.Version) {
		return errors.New("invalid semver format in release.json: " + rel.Version)
	}

	var missing, extra []string
	for _, s := range steps {
		if _, ok := rel.Steps[s]; !ok {
			missing = append(missing, s)
		}
	}
	for s := range rel.Steps {
		if !slices.Contains(steps, s) {
			extra = append(extra, s)
		}
	}
	if len(missing) > 0 || len(extra) > 0 {
		slices.Sort(extra)
		return fmt.Errorf("steps in release.json do not match -steps (missing: [%s], extra: [%s])",
			strings.Join(missing, ", "), strings.Join(extra, ", "))
	}

	seenZero := false
	for _, s := range steps {
		v := rel.Steps[s]
		if v != 0 && v != 1 {
			return fmt.Errorf("invalid step value for %s: %d (must be 0 or 1)", s, v)
		}
		if seenZero && v == 1 {
			return fmt.Errorf("invalid step order: %s is done but a previous step is not", s)
		}
		if v == 0 {
			seenZero = true
		}
	}
	return nil
}

// next returns the first pending step, or "" if all steps are done.
func next(rel *release, steps []string) string {
	for _, s := range steps {
		if rel.Steps[s] == 0 {
			return s
		}
	}
	return ""
}
