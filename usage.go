package arg

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DoOnFail represents a user-defined response to parsing process failings
// By default, it prints usage information to stderr and exits with negative status
var DoOnFail = func(p *Parser, msg string) {
	p.WriteUsage(os.Stderr)
	fmt.Fprintln(os.Stderr, "error:", msg)
	os.Exit(-1)
}

// Fail utilizes DoOnFail to react to parsing process failings
func (p *Parser) Fail(msg string) {
	DoOnFail(p, msg)
}

// WriteUsage writes usage information to the given writer
func (p *Parser) WriteUsage(w io.Writer) {
	var positionals, options []*spec
	for _, spec := range p.spec {
		if spec.positional {
			positionals = append(positionals, spec)
		} else {
			options = append(options, spec)
		}
	}

	fmt.Fprintf(w, "usage: %s", filepath.Base(os.Args[0]))

	// write the option component of the usage message
	for _, spec := range options {
		// prefix with a space
		fmt.Fprint(w, " ")
		if !spec.required {
			fmt.Fprint(w, "[")
		}
		fmt.Fprint(w, synopsis(spec, "--"+spec.long))
		if !spec.required {
			fmt.Fprint(w, "]")
		}
	}

	// write the positional component of the usage message
	for _, spec := range positionals {
		// prefix with a space
		fmt.Fprint(w, " ")
		up := strings.ToUpper(spec.long)
		if spec.multiple {
			fmt.Fprintf(w, "[%s [%s ...]]", up, up)
		} else {
			fmt.Fprint(w, up)
		}
	}
	fmt.Fprint(w, "\n")
}

// WriteHelp writes the usage string followed by the full help string for each option
func (p *Parser) WriteHelp(w io.Writer) {
	var positionals, options []*spec
	for _, spec := range p.spec {
		if spec.positional {
			positionals = append(positionals, spec)
		} else {
			options = append(options, spec)
		}
	}

	p.WriteUsage(w)

	// write the list of positionals
	if len(positionals) > 0 {
		fmt.Fprint(w, "\npositional arguments:\n")
		for _, spec := range positionals {
			fmt.Fprintf(w, "  %s\n", spec.long)
		}
	}

	// write the list of options
	fmt.Fprint(w, "\noptions:\n")
	for _, spec := range options {
		printOption(w, spec)
	}

	// write the list of built in options
	printOption(w, &spec{isBool: true, long: "help", short: "h", help: "display this help and exit"})
}

func printOption(w io.Writer, spec *spec) {
	const colWidth = 25
	left := "  " + synopsis(spec, "--"+spec.long)
	if spec.short != "" {
		left += ", " + synopsis(spec, "-"+spec.short)
	}
	fmt.Fprint(w, left)
	if spec.help != "" {
		if len(left)+2 < colWidth {
			fmt.Fprint(w, strings.Repeat(" ", colWidth-len(left)))
		} else {
			fmt.Fprint(w, "\n"+strings.Repeat(" ", colWidth))
		}
		fmt.Fprint(w, spec.help)
	}
	fmt.Fprint(w, "\n")
}

func synopsis(spec *spec, form string) string {
	if spec.isBool {
		return form
	}
	return form + " " + strings.ToUpper(spec.long)
}
