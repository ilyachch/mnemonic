package main

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/ilyachch/mnemonic/internal/adapter/cli"
	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func main() {
	boot, err := app.New(app.Input{})
	if err != nil {
		exit(err)
	}
	defer func() {
		_ = boot.Close()
	}()

	root := cli.NewRootCommand(boot)
	root.DisableAutoGenTag = true

	var out bytes.Buffer
	out.WriteString("# CLI Reference\n\n")

	out.WriteString("## Command Directory\n\n")
	writeCommandTreeIndex(&out, root, 0)
	out.WriteString("\n---\n\n")

	if err := writeCommandTree(&out, root, true); err != nil {
		exit(err)
	}

	if _, err := os.Stdout.Write(out.Bytes()); err != nil {
		exit(err)
	}
}

func writeCommandTreeIndex(out *bytes.Buffer, cmd *cobra.Command, depth int) {
	if cmd.Hidden {
		return
	}

	indent := strings.Repeat("  ", depth)
	anchor := toAnchor(cmd.CommandPath())

	fmt.Fprintf(out, "%s- [%s](#%s)\n", indent, cmd.Name(), anchor)

	children := visibleCommands(cmd.Commands())
	for _, child := range children {
		writeCommandTreeIndex(out, child, depth+1)
	}
}

func writeCommandTree(out *bytes.Buffer, cmd *cobra.Command, first bool) error {
	if cmd.Hidden {
		return nil
	}

	var buf bytes.Buffer
	if err := doc.GenMarkdown(cmd, &buf); err != nil {
		return err
	}

	text := cleanupMarkdown(buf.String())
	text = wrapCommandHeading(text, cmd)

	if !first {
		out.WriteString("\n---\n\n")
	}
	out.WriteString(text)
	out.WriteString("\n")

	children := visibleCommands(cmd.Commands())
	for _, child := range children {
		if err := writeCommandTree(out, child, false); err != nil {
			return err
		}
	}

	return nil
}

func visibleCommands(commands []*cobra.Command) []*cobra.Command {
	var result []*cobra.Command

	for _, cmd := range commands {
		if !cmd.Hidden {
			result = append(result, cmd)
		}
	}

	slices.SortFunc(result, func(a, b *cobra.Command) int {
		return strings.Compare(a.CommandPath(), b.CommandPath())
	})

	return result
}

func cleanupMarkdown(s string) string {
	s = stripSeeAlso(s)
	s = strings.TrimSpace(s)

	return s
}

func wrapCommandHeading(s string, cmd *cobra.Command) string {
	path := cmd.CommandPath()

	s = strings.Replace(s, "## "+path, "## `"+path+"`", 1)
	s = strings.Replace(s, "# "+path, "# `"+path+"`", 1)

	return s
}

func stripSeeAlso(s string) string {
	lines := strings.Split(s, "\n")

	var out []string
	skip := false

	for _, line := range lines {
		if strings.HasPrefix(line, "### SEE ALSO") {
			skip = true
			continue
		}

		if skip && strings.HasPrefix(line, "## ") {
			skip = false
		}

		if !skip {
			out = append(out, line)
		}
	}

	return strings.Join(out, "\n")
}

func toAnchor(path string) string {
	path = strings.ToLower(path)
	var buf strings.Builder
	for _, r := range path {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			buf.WriteRune(r)
		} else if r == ' ' {
			buf.WriteRune('-')
		}
	}
	return buf.String()
}

func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
