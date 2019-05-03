package arg

import (
	"encoding"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
)

// the width of the left column
const colWidth = 25

// Fail prints usage information to stderr and exits with non-zero status
func (p *Parser) Fail(msg string) {
	p.WriteUsage(os.Stderr)
	fmt.Fprintln(os.Stderr, "error:", msg)
	os.Exit(-1)
}

// WriteUsage writes usage information to the given writer
func (p *Parser) WriteUsage(w io.Writer) {
	var positionals, options []*spec
	for _, spec := range p.cmd.specs {
		if spec.positional {
			positionals = append(positionals, spec)
		} else {
			options = append(options, spec)
		}
	}

	if p.version != "" {
		fmt.Fprintln(w, p.version)
	}

	fmt.Fprintf(w, "Usage: %s", p.cmd.name)

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
			if !spec.required {
				fmt.Fprint(w, "[")
			}
			fmt.Fprintf(w, "%s [%s ...]", up, up)
			if !spec.required {
				fmt.Fprint(w, "]")
			}
		} else {
			fmt.Fprint(w, up)
		}
	}
	fmt.Fprint(w, "\n")
}

func printTwoCols(w io.Writer, left, help string, defaultVal *string) {
	lhs := "  " + left
	fmt.Fprint(w, lhs)
	if help != "" {
		if len(lhs)+2 < colWidth {
			fmt.Fprint(w, strings.Repeat(" ", colWidth-len(lhs)))
		} else {
			fmt.Fprint(w, "\n"+strings.Repeat(" ", colWidth))
		}
		fmt.Fprint(w, help)
	}
	if defaultVal != nil {
		fmt.Fprintf(w, " [default: %s]", *defaultVal)
	}
	fmt.Fprint(w, "\n")
}

// WriteHelp writes the usage string followed by the full help string for each option
func (p *Parser) WriteHelp(w io.Writer) {
	var positionals, options []*spec
	for _, spec := range p.cmd.specs {
		if spec.positional {
			positionals = append(positionals, spec)
		} else {
			options = append(options, spec)
		}
	}

	if p.description != "" {
		fmt.Fprintln(w, p.description)
	}
	p.WriteUsage(w)

	// write the list of positionals
	if len(positionals) > 0 {
		fmt.Fprint(w, "\nPositional arguments:\n")
		for _, spec := range positionals {
			printTwoCols(w, strings.ToUpper(spec.long), spec.help, nil)
		}
	}

	// write the list of options
	fmt.Fprint(w, "\nOptions:\n")
	for _, spec := range options {
		p.printOption(w, spec)
	}

	// write the list of built in options
	p.printOption(w, &spec{
		boolean: true,
		long:    "help",
		short:   "h",
		help:    "display this help and exit",
	})
	if p.version != "" {
		p.printOption(w, &spec{
			boolean: true,
			long:    "version",
			help:    "display version and exit",
		})
	}

	// write the list of subcommands
	if len(p.cmd.subcommands) > 0 {
		fmt.Fprint(w, "\nCommands:\n")
		for _, subcmd := range p.cmd.subcommands {
			printTwoCols(w, subcmd.name, subcmd.help, nil)
		}
	}
}

func (p *Parser) printOption(w io.Writer, spec *spec) {
	left := synopsis(spec, "--"+spec.long)
	if spec.short != "" {
		left += ", " + synopsis(spec, "-"+spec.short)
	}

	// If spec.dest is not the zero value then a default value has been added.
	var v reflect.Value
	if len(spec.dest.fields) > 0 {
		v = p.readable(spec.dest)
	}

	var defaultVal *string
	if v.IsValid() {
		z := reflect.Zero(v.Type())
		if (v.Type().Comparable() && z.Type().Comparable() && v.Interface() != z.Interface()) || v.Kind() == reflect.Slice && !v.IsNil() {
			if scalar, ok := v.Interface().(encoding.TextMarshaler); ok {
				if value, err := scalar.MarshalText(); err != nil {
					defaultVal = ptrTo(fmt.Sprintf("error: %v", err))
				} else {
					defaultVal = ptrTo(fmt.Sprintf("%v", string(value)))
				}
			} else {
				defaultVal = ptrTo(fmt.Sprintf("%v", v))
			}
		}
	}
	printTwoCols(w, left, spec.help, defaultVal)
}

func synopsis(spec *spec, form string) string {
	if spec.boolean {
		return form
	}
	return form + " " + strings.ToUpper(spec.long)
}

func ptrTo(s string) *string {
	return &s
}
