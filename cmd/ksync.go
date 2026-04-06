/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
	"github.com/spf13/cobra"
)

const kickstartZipURL = "https://github.com/nvim-lua/kickstart.nvim/archive/refs/heads/master.zip"

// Tree-sitter queries for extracting string values from Lua table fields.
// The alternate query handles grammars that name the child node "content" instead of implicitly.
const (
	tsFieldQuery    = `(field value: (string (string_content) @plugin_name))`
	tsFieldQueryAlt = `(field value: (string content: (string_content) @plugin_name))`
)

var (
	rootDir    string
	ksyncDebug bool

	// Matches GitHub-style owner/repo slugs.
	repoSlugRe = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)

	// Matches quoted owner/repo strings that follow `{`, `,`, or a newline —
	// the positions where lazy.nvim plugin specs appear.
	lazyPluginRe = regexp.MustCompile(`(?:[{,]|\n)\s*['"]([A-Za-z0-9._-]+/[A-Za-z0-9._-]+)['"]`)
)

// ANSI color codes for terminal output.
const (
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
	colorGray  = "\033[90m"
	colorReset = "\033[0m"
	colorBold  = "\033[1m"
)

type pluginDiff struct {
	onlyInKickstart []string
	onlyInYours     []string
	shared          []string
}

var ksyncCmd = &cobra.Command{
	Use:   "ksync",
	Short: "list plugin discrepancies between your plugin list and kickstart's",
	RunE: func(cmd *cobra.Command, args []string) error {
		remotePlugins, err := fetchKickstartPlugins()
		if err != nil {
			return err
		}

		localPlugins := collectLocalPlugins()
		diff := diffPlugins(remotePlugins, localPlugins)
		printDiff(diff)

		return nil
	},
}

func diffPlugins(kickstart, yours map[string]bool) pluginDiff {
	var diff pluginDiff

	for p := range kickstart {
		if yours[p] {
			diff.shared = append(diff.shared, p)
		} else {
			diff.onlyInKickstart = append(diff.onlyInKickstart, p)
		}
	}
	for p := range yours {
		if !kickstart[p] {
			diff.onlyInYours = append(diff.onlyInYours, p)
		}
	}

	sort.Strings(diff.onlyInKickstart)
	sort.Strings(diff.onlyInYours)
	sort.Strings(diff.shared)

	return diff
}

func printDiff(diff pluginDiff) {
	printSection := func(color, symbol, label string, plugins []string) {
		if len(plugins) == 0 {
			return
		}
		fmt.Printf("\n%s%s %s (%d)%s\n", colorBold, symbol, label, len(plugins), colorReset)
		for _, p := range plugins {
			fmt.Printf("  %s%s  %s%s\n", color, symbol, p, colorReset)
		}
	}

	printSection(colorRed, "✗", "only in kickstart", diff.onlyInKickstart)
	printSection(colorGreen, "✓", "only in yours", diff.onlyInYours)
	printSection(colorGray, "·", "shared", diff.shared)

	fmt.Printf(
		"\n%sSummary:%s  %s✗ %d missing%s  %s✓ %d extra%s  %s· %d shared%s\n",
		colorBold, colorReset,
		colorRed, len(diff.onlyInKickstart), colorReset,
		colorGreen, len(diff.onlyInYours), colorReset,
		colorGray, len(diff.shared), colorReset,
	)
}

func init() {
	home, _ := os.UserHomeDir()
	defaultNvimPath := filepath.Join(home, ".config", "nvim")

	rootCmd.AddCommand(ksyncCmd)
	ksyncCmd.PersistentFlags().StringVarP(&rootDir, "root directory", "R", defaultNvimPath, "Nvim config root")
	ksyncCmd.Flags().BoolVarP(&ksyncDebug, "debug", "d", false, "print parse/query diagnostics (stderr)")
}

// collectLocalPlugins walks every .lua file under rootDir and extracts plugin slugs.
func collectLocalPlugins() map[string]bool {
	plugins := make(map[string]bool)

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".lua") {
			return err
		}
		addPluginsFromFile(path, plugins)
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "error walking %q: %v\n", rootDir, err)
	}

	return plugins
}

// fetchKickstartPlugins downloads the kickstart.nvim zip and extracts all plugin slugs.
func fetchKickstartPlugins() (map[string]bool, error) {
	body, err := downloadZip(kickstartZipURL)
	if err != nil {
		return nil, err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	plugins := make(map[string]bool)
	for _, f := range zipReader.File {
		if !strings.HasSuffix(f.Name, ".lua") {
			continue
		}
		content, err := readZipFile(f)
		if err != nil {
			return nil, err
		}
		for _, p := range extractPluginsFromLua(content, f.Name) {
			plugins[p] = true
		}
	}

	return plugins, nil
}

func downloadZip(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// extractPluginsFromLua finds owner/repo slugs using tree-sitter first, then a strict regex fallback.
func extractPluginsFromLua(content []byte, sourcePath string) []string {
	seen := make(map[string]bool)
	var results []string

	add := func(s string) {
		if repoSlugRe.MatchString(s) && !seen[s] {
			seen[s] = true
			results = append(results, s)
		}
	}

	tsPlugins := extractViaTreeSitter(content, sourcePath)
	for _, p := range tsPlugins {
		add(p)
	}

	regexPlugins := extractViaRegex(content)
	debugLog(sourcePath, fmt.Sprintf("strict-regex distinct=%d", len(regexPlugins)))
	for _, p := range regexPlugins {
		add(p)
	}

	debugLog(sourcePath, fmt.Sprintf("total repos=%d", len(results)))
	return results
}

func extractViaTreeSitter(content []byte, sourcePath string) []string {
	lang := grammars.LuaLanguage()
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(content)

	if err != nil {
		debugLog(sourcePath, fmt.Sprintf("parse error: %v", err))
		return nil
	}
	if tree == nil || tree.RootNode() == nil {
		debugLog(sourcePath, "nil tree or root")
		return nil
	}

	root := tree.RootNode()
	src := tree.Source()
	if len(src) == 0 {
		src = content
	}

	treeLang := tree.Language()
	if treeLang == nil {
		treeLang = lang
	}

	debugLog(sourcePath, fmt.Sprintf("root.HasError=%v", root.HasError()))

	query, err := buildQuery(treeLang, sourcePath)
	if err != nil || query == nil {
		return nil
	}

	var results []string
	matchCount := 0
	cursor := query.Exec(root, treeLang, src)

	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}
		matchCount++
		for _, cap := range match.Captures {
			t := cap.Text(src)
			debugLog(sourcePath, fmt.Sprintf("field string %q", t))
			results = append(results, t)
		}
	}

	debugLog(sourcePath, fmt.Sprintf("query rounds=%d repos from ts=%d", matchCount, len(results)))
	return results
}

func buildQuery(lang *gotreesitter.Language, sourcePath string) (*gotreesitter.Query, error) {
	q, err := gotreesitter.NewQuery(tsFieldQuery, lang)
	if err != nil {
		q, err = gotreesitter.NewQuery(tsFieldQueryAlt, lang)
	}
	if err != nil {
		debugLog(sourcePath, fmt.Sprintf("NewQuery: %v", err))
		return nil, err
	}
	return q, nil
}

// extractViaRegex finds owner/repo slugs using the lazy.nvim positional regex,
// after stripping Lua comments to avoid matching repos mentioned in doc strings.
func extractViaRegex(content []byte) []string {
	sanitized := stripLuaComments(content)
	matches := lazyPluginRe.FindAllStringSubmatch(string(sanitized), -1)

	seen := make(map[string]bool)
	var results []string

	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		s := m[1]
		if repoSlugRe.MatchString(s) && !seen[s] {
			seen[s] = true
			results = append(results, s)
		}
	}

	return results
}

func addPluginsFromFile(path string, dest map[string]bool) {
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, p := range extractPluginsFromLua(content, path) {
		dest[p] = true
	}
}

func debugLog(path, msg string) {
	if !ksyncDebug {
		return
	}
	if path != "" {
		fmt.Fprintf(os.Stderr, "[ksync] %s: %s\n", path, msg)
	} else {
		fmt.Fprintf(os.Stderr, "[ksync] %s\n", msg)
	}
}

// stripLuaComments removes -- and --[[ ]] comments while preserving string literals.
func stripLuaComments(src []byte) []byte {
	var out bytes.Buffer
	i := 0

	for i < len(src) {
		switch {
		case i+2 < len(src) && src[i] == '-' && src[i+1] == '-' && src[i+2] == '[':
			// Long block comment: --[=*[ ... ]=*]
			n := longBracketLen(src[i+2:])
			if n == 0 {
				out.WriteByte(src[i])
				i++
				continue
			}
			chunk := src[i : i+2+n]
			out.Write(bytes.Repeat([]byte{'\n'}, bytes.Count(chunk, []byte{'\n'})))
			i += 2 + n

		case i+1 < len(src) && src[i] == '-' && src[i+1] == '-':
			// Short line comment: -- ...
			for i < len(src) && src[i] != '\n' {
				i++
			}
			if i < len(src) {
				out.WriteByte('\n')
				i++
			}

		case i+1 < len(src) && src[i] == '[' && (src[i+1] == '[' || src[i+1] == '='):
			// Long string literal: [=*[ ... ]=*]
			n := longBracketLen(src[i:])
			if n > 0 {
				out.Write(src[i : i+n])
				i += n
				continue
			}
			fallthrough

		case src[i] == '\'' || src[i] == '"':
			// Short string literal — copy verbatim including escaped chars.
			q := src[i]
			out.WriteByte(q)
			i++
			for i < len(src) {
				if src[i] == '\\' && i+1 < len(src) {
					out.Write(src[i : i+2])
					i += 2
					continue
				}
				out.WriteByte(src[i])
				if src[i] == q {
					i++
					break
				}
				i++
			}

		default:
			out.WriteByte(src[i])
			i++
		}
	}

	return out.Bytes()
}

// longBracketLen returns the total byte length of a [=*[ ... ]=*] segment
// starting at src[0]. Returns 0 if src does not begin a valid long bracket.
func longBracketLen(src []byte) int {
	if len(src) < 2 || src[0] != '[' {
		return 0
	}

	eqCount := 0
	i := 1
	for i < len(src) && src[i] == '=' {
		eqCount++
		i++
	}
	if i >= len(src) || src[i] != '[' {
		return 0
	}
	i++ // skip opening second `[`

	// Scan for the matching closing bracket ]=*]
	for i < len(src) {
		if src[i] != ']' {
			i++
			continue
		}
		j := i + 1
		closeEq := 0
		for j < len(src) && src[j] == '=' {
			closeEq++
			j++
		}
		if j < len(src) && src[j] == ']' && closeEq == eqCount {
			return j + 1
		}
		i++
	}

	return len(src)
}
