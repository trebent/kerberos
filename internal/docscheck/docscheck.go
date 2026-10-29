// Package docscheck ties the user documentation under docs/ to the code. Tests use it to fail
// when a documented key, default or example drifts from what the code actually does.
//
// Every config section and environment variable is documented in exactly one page (its owner),
// see docs/configuration.md. The helpers here read those owner pages.
package docscheck

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	errNoRepoRoot = errors.New("go.mod not found in any parent directory")

	jsonBlockRe = regexp.MustCompile("(?s)```json\n(.*?)```")
	helpLineRe  = regexp.MustCompile(`^(\S+) \(([^)]*)\)\s*:.*?(?:\(default: (.*)\))?$`)
)

// EnvVar is an environment variable as printed by envparser.Help().
type EnvVar struct {
	Name     string
	Required bool
	Default  string
}

// RepoRoot returns the repository root, found by walking up from the working directory to go.mod.
func RepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errNoRepoRoot
		}

		dir = parent
	}
}

// ReadDoc returns the contents of a file given relative to the repository root.
func ReadDoc(t testing.TB, rel string) string {
	t.Helper()

	root, err := RepoRoot()
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

// JSONBlocks returns the contents of every ```json fenced block in a markdown document.
func JSONBlocks(md string) []string {
	matches := jsonBlockRe.FindAllStringSubmatch(md, -1)
	blocks := make([]string, 0, len(matches))

	for _, m := range matches {
		blocks = append(blocks, m[1])
	}

	return blocks
}

// TableCell finds the first markdown table row whose first cell is exactly firstCell and returns
// the content of the cell under the column whose header is column. The boolean is false if no
// such row or column exists.
func TableCell(md, firstCell, column string) (string, bool) {
	var header []string

	for line := range strings.SplitSeq(md, "\n") {
		cells := splitRow(line)
		if cells == nil {
			header = nil
			continue
		}

		if header == nil {
			header = cells
			continue
		}

		if len(cells) == 0 {
			continue // the | --- | separator row under the header
		}

		if cells[0] != firstCell {
			continue
		}

		for i, h := range header {
			if h == column && i < len(cells) {
				return cells[i], true
			}
		}
	}

	return "", false
}

// MentionsKey reports whether md mentions a config key in backticks, either alone (`key`) or as a
// segment of a path (`a.key`, `a[].key.b`).
func MentionsKey(md, key string) bool {
	re := regexp.MustCompile("`([^`\\s]*[.\\]])?" + regexp.QuoteMeta(key) + "([.\\[][^`\\s]*)?`")
	return re.MatchString(md)
}

// ParseEnvHelp parses the output of envparser.Help() into its variables.
func ParseEnvHelp(help string) []EnvVar {
	var vars []EnvVar

	for line := range strings.SplitSeq(help, "\n") {
		m := helpLineRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}

		vars = append(vars, EnvVar{
			Name:     m[1],
			Required: strings.HasSuffix(m[2], ", required"),
			Default:  m[3],
		})
	}

	return vars
}

// CheckEnvTable asserts that every variable in help has a row in the environment variable table
// of the doc at docRel, and that the row's Default cell starts with the real default.
func CheckEnvTable(t *testing.T, docRel, help string) {
	t.Helper()

	md := ReadDoc(t, docRel)
	vars := ParseEnvHelp(help)

	if len(vars) == 0 {
		t.Fatal("no environment variables parsed from envparser help output")
	}

	for _, v := range vars {
		got, ok := TableCell(md, "`"+v.Name+"`", "Default")
		if !ok {
			t.Errorf(
				"%s: no table row with a Default column for environment variable %s",
				docRel,
				v.Name,
			)
			continue
		}

		want := "`" + v.Default + "`"

		switch {
		case v.Required:
			want = "required"
		case v.Default == "":
			want = "empty"
		}

		if !strings.HasPrefix(got, want) {
			t.Errorf(
				"%s: default of %s is documented as %q, the code says %s",
				docRel,
				v.Name,
				got,
				want,
			)
		}
	}
}

// splitRow splits a markdown table row into trimmed cells, or returns nil if line is not a row.
// Separator rows (| --- | --- |) are returned as nil too, so they never become a header.
func splitRow(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil
	}

	if strings.Trim(line, "|-: ") == "" {
		return []string{}
	}

	// An escaped pipe (\|) is cell content, not a column boundary.
	const escapedPipe = "\x00"

	line = strings.ReplaceAll(line, `\|`, escapedPipe)
	parts := strings.Split(strings.Trim(line, "|"), "|")
	cells := make([]string, 0, len(parts))

	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(strings.ReplaceAll(p, escapedPipe, "|")))
	}

	return cells
}
