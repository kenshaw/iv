package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

// These tests read the documents and the agent skills of the repository, not
// the code. They hold the setup that D21 adopted from dbmeta D110.

// TestSkillsAreCopies holds dbmeta D110, which D21 adopts. Each skill in
// skills-lock.json is an ordinary folder under .agents/skills and under
// .claude/skills, and the two folders hold the same files.
//
// A symbolic link is refused because a Windows checkout writes one as a text
// file that holds the target path. Claude Code then finds a file where it
// expects a folder, and it loads no skill and reports nothing.
func TestSkillsAreCopies(t *testing.T) {
	var lock struct {
		Skills map[string]json.RawMessage `json:"skills"`
	}
	if err := json.Unmarshal([]byte(read(t, "skills-lock.json")), &lock); err != nil {
		t.Fatalf("reading skills-lock.json: %v", err)
	}
	if len(lock.Skills) == 0 {
		t.Fatal("skills-lock.json names no skill")
	}
	roots := []string{filepath.Join(".agents", "skills"), filepath.Join(".claude", "skills")}
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if _, ok := lock.Skills[e.Name()]; !ok {
				t.Errorf("%s is not in skills-lock.json, so nobody can install it again. "+
					"Add it with the command in CONTRIBUTING.md", filepath.Join(root, e.Name()))
			}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(lock.Skills)) {
		agents := skillFiles(t, filepath.Join(roots[0], name))
		claude := skillFiles(t, filepath.Join(roots[1], name))
		if agents == nil || claude == nil {
			continue
		}
		for _, path := range slices.Sorted(maps.Keys(agents)) {
			switch other, ok := claude[path]; {
			case !ok:
				t.Errorf("%s: %s is in %s and not in %s", name, path, roots[0], roots[1])
			case other != agents[path]:
				t.Errorf("%s: %s differs between %s and %s", name, path, roots[0], roots[1])
			}
		}
		for _, path := range slices.Sorted(maps.Keys(claude)) {
			if _, ok := agents[path]; !ok {
				t.Errorf("%s: %s is in %s and not in %s", name, path, roots[1], roots[0])
			}
		}
	}
}

// skillFiles returns the content of every file in one copy of a skill, keyed
// by its path inside the copy. It returns nil after it reports a copy that is
// missing or that is a symbolic link.
func skillFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	fi, err := os.Lstat(dir)
	switch {
	case err != nil:
		t.Errorf("%s is missing. Install the skill with the command in CONTRIBUTING.md", dir)
		return nil
	case !fi.IsDir():
		t.Errorf("%s is not a folder. A Windows checkout writes a symbolic link as a text file, "+
			"so install the skill with --copy. See D21", dir)
		return nil
	}
	out := make(map[string]string)
	err = fs.WalkDir(os.DirFS(dir), ".", func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.Type()&fs.ModeSymlink != 0:
			t.Errorf("%s is a symbolic link. See D21", filepath.Join(dir, path))
		case d.Type().IsRegular():
			body, err := fs.ReadFile(os.DirFS(dir), path)
			if err != nil {
				return err
			}
			out[path] = string(body)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	return out
}

// TestClaudeImportsAgents holds dbmeta D110, which D21 adopts. AGENTS.md
// holds the rules, because Codex and the other agents read it, and CLAUDE.md
// imports it, so that Claude Code reads the same rules. A symbolic link is
// refused for the reason in TestSkillsAreCopies.
func TestClaudeImportsAgents(t *testing.T) {
	fi, err := os.Lstat("CLAUDE.md")
	switch {
	case err != nil:
		t.Fatal(err)
	case !fi.Mode().IsRegular():
		t.Fatalf("CLAUDE.md is not an ordinary file, got mode %v. See D21", fi.Mode())
	}
	if s := read(t, "CLAUDE.md"); s != "@AGENTS.md\n" {
		t.Errorf("CLAUDE.md must hold exactly the line @AGENTS.md, got %q. "+
			"Put the rules in AGENTS.md. See D21", s)
	}
	if _, err := os.Stat("AGENTS.md"); err != nil {
		t.Errorf("CLAUDE.md imports AGENTS.md, which is missing: %v", err)
	}
}

// TestTheRootHoldsFourDocuments holds dbmeta D110, which D21 adopts. A
// document that appears in the root is one that nobody filed.
func TestTheRootHoldsFourDocuments(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{
		"README.md": true, "AGENTS.md": true, "CLAUDE.md": true, "CONTRIBUTING.md": true,
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if !allowed[e.Name()] {
			t.Errorf("%s is in the repository root. Only README, AGENTS, CLAUDE and CONTRIBUTING "+
				"belong there. Move it to docs/. See D21", e.Name())
		}
	}
	for _, name := range slices.Sorted(maps.Keys(allowed)) {
		if _, err := os.Stat(name); err != nil {
			t.Errorf("expected %s in the repository root. See D21", name)
		}
	}
}

// decision is one decision in docs/PLAN.md.
type decision struct {
	num    int
	title  string
	status string
	anchor string
}

// decisionHeading matches the heading of a decision:
//
//	### D8. The libvips binding is cshum/vipsgen. Decided. Amends D5.
//
// The status is Decided or Proposed, and then each decision that this one
// amends or that amends it.
var decisionHeading = regexp.MustCompile(`^### D(\d+)\. (.+?)\. ((?:Decided|Proposed)` +
	`(?:\. (?:Amends|Amended by) D\d+)*)\.$`)

// planPath is the document that holds the plan and the decisions. A small
// library keeps its decisions there (dbmeta D111).
var planPath = filepath.Join("docs", "PLAN.md")

// decisions reads every decision in docs/PLAN.md, in the order of the
// document.
func decisions(t *testing.T) []decision {
	t.Helper()
	var out []decision
	for i, line := range strings.Split(read(t, planPath), "\n") {
		if !strings.HasPrefix(line, "### D") {
			continue
		}
		m := decisionHeading.FindStringSubmatch(line)
		if m == nil {
			t.Errorf("%s:%d: a decision heading is \"### D<n>. <title>. <status>.\", and the status "+
				"is Decided or Proposed, then \"Amends Dn\" or \"Amended by Dn\" if needed: %q",
				planPath, i+1, line)
			continue
		}
		num, _ := strconv.Atoi(m[1])
		out = append(out, decision{
			num:    num,
			title:  m[2],
			status: m[3],
			anchor: anchor(strings.TrimPrefix(line, "### ")),
		})
	}
	if len(out) == 0 {
		t.Fatalf("%s holds no decision, so there is nothing to test", planPath)
	}
	return out
}

// anchor returns the fragment that GitHub gives a heading. It keeps letters,
// digits, hyphens and underscores, changes each space to a hyphen and drops
// the rest.
func anchor(heading string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(heading) {
		switch {
		case r == ' ':
			b.WriteRune('-')
		case r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		}
	}
	return "#" + b.String()
}

// TestTheDecisionIndexIsComplete compares the index at the top of the
// decisions in docs/PLAN.md with their headings. A reader finds a decision by
// its number in that table, so a missing row or a stale status hides it.
func TestTheDecisionIndexIsComplete(t *testing.T) {
	rows := make(map[int]string)
	for _, m := range regexp.MustCompile(`(?m)^\| \[D(\d+)\]\(.*$`).FindAllStringSubmatch(read(t, planPath), -1) {
		num, _ := strconv.Atoi(m[1])
		if _, ok := rows[num]; ok {
			t.Errorf("%s has two rows for D%d", planPath, num)
		}
		rows[num] = m[0]
	}
	written := make(map[int]bool)
	for i, d := range decisions(t) {
		if d.num != i+1 {
			t.Errorf("%s: decision %d in the document is D%d. The numbers run from D1 with no gap",
				planPath, i+1, d.num)
		}
		written[d.num] = true
		want := fmt.Sprintf("| [D%d](%s) | %s | %s |", d.num, d.anchor, d.title, d.status)
		switch got, ok := rows[d.num]; {
		case !ok:
			t.Errorf("D%d has no row in the index in %s. Add:\n%s", d.num, planPath, want)
		case got != want:
			t.Errorf("D%d: the row in the index in %s is\n%s\nand the heading says\n%s", d.num, planPath, got, want)
		}
	}
	for _, num := range slices.Sorted(maps.Keys(rows)) {
		if !written[num] {
			t.Errorf("the index in %s has a row for D%d, and no heading holds it", planPath, num)
		}
	}
}

// TestAnAmendmentPointsBothWays fails unless both headings name each other
// when a decision amends an earlier one. A reader who finds the older decision
// otherwise gets a rule that no longer holds.
func TestAnAmendmentPointsBothWays(t *testing.T) {
	status := make(map[int]string)
	for _, d := range decisions(t) {
		status[d.num] = d.status
	}
	reverse := map[string]string{"Amends": "Amended by", "Amended by": "Amends"}
	link := regexp.MustCompile(`(Amends|Amended by) D(\d+)`)
	for _, num := range slices.Sorted(maps.Keys(status)) {
		for _, m := range link.FindAllStringSubmatch(status[num], -1) {
			other, _ := strconv.Atoi(m[2])
			want := fmt.Sprintf("%s D%d", reverse[m[1]], num)
			switch back, ok := status[other]; {
			case !ok:
				t.Errorf("D%d says %q, and D%d is not a decision", num, m[0], other)
			case !strings.Contains(back, want):
				t.Errorf("D%d says %q, and the heading of D%d does not say %q", num, m[0], other, want)
			}
		}
	}
}

// decisionRef matches a reference to a decision. A reference to a decision of
// another repository names that repository first, as in dbmeta D110. A bare
// number, as in D3, names a decision in docs/PLAN.md.
var decisionRef = regexp.MustCompile(`(?:\b([a-z0-9]+)\s+)?\bD(\d+)\b`)

// TestEveryDecisionReferenceExists fails on a bare reference, such as D3, to a
// decision that docs/PLAN.md does not hold. It reads the documents and the Go
// files of this repository, and not the skills, which are copies of another
// project.
func TestEveryDecisionReferenceExists(t *testing.T) {
	written := make(map[string]bool)
	for _, d := range decisions(t) {
		written[strconv.Itoa(d.num)] = true
	}
	// The repositories whose decisions a document here can cite.
	repos := map[string]bool{
		"dbmeta": true, "dbimp": true, "usql": true, "dburl": true, "tblfmt": true,
	}
	var paths []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir() && path != "." && (strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata" || d.Name() == "build"):
			return filepath.SkipDir
		case strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".go"):
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		for _, m := range decisionRef.FindAllStringSubmatch(read(t, path), -1) {
			if repos[m[1]] || written[m[2]] {
				continue
			}
			t.Errorf("%s: refers to D%s, which is not in %s. Name the repository first if it "+
				"is a decision of another repository, as in dbmeta D110", path, m[2], planPath)
		}
	}
}

// read returns the content of the file at path.
func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
