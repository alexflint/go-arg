package arg

import (
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
	for _, spec := range p.spec {
		if spec.positional {
			positionals = append(positionals, spec)
		} else {
			options = append(options, spec)
		}
	}

	if p.version != "" {
		fmt.Fprintln(w, p.version)
	}

	fmt.Fprintf(w, "usage: %s", p.config.Program)

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

	if p.description != "" {
		fmt.Fprintln(w, p.description)
	}
	p.WriteUsage(w)

	// write the list of positionals
	if len(positionals) > 0 {
		fmt.Fprint(w, "\npositional arguments:\n")
		for _, spec := range positionals {
			name := spec.long
			if name == "" {
				name = spec.short
			}
			fmt.Fprintf(w, "  %-26s %s\n", name, spec.help)
		}
	}

	// write the list of options
	fmt.Fprint(w, "\noptions:\n")
	for _, spec := range options {
		fmt.Fprintln(w, spec)
	}

	fmt.Fprintln(w, &spec{boolean: true, long: "help", short: "h", help: "display this help and exit"})
	if p.version != "" {
		fmt.Fprintln(w, &spec{boolean: true, long: "version", help: "display version and exit"})
	}
}

func (s *spec) getValueDefault() string {
	v := s.dest
	if v.IsValid() {
		z := reflect.Zero(v.Type())
		if (v.Type().Comparable() && z.Type().Comparable() && v.Interface() != z.Interface()) || v.Kind() == reflect.Slice && !v.IsNil() {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

func (s *spec) fmtValueType() string {
	if s.dest.IsValid() {
		t := s.dest.Type().String()

		var valType string
		switch {
		case strings.Contains(t, "string"):
			valType = "s"
		case strings.Contains(t, "int"):
			valType = "n"
		case strings.Contains(t, "float"):
			valType = "f"
		case strings.Contains(t, "time"):
			valType = "t"
		}

		if t[0:2] == "[]" {
			return fmt.Sprintf("[%s]", valType)
		}
		if valType != "" {
			return fmt.Sprintf("<%s>", valType)
		}
	}
	return ""
}

func (s *spec) String() string {
	short := s.short
	if short != "" {
		short = fmt.Sprintf("-%s,", s.short)
	}

	val := s.fmtValueType()
	def := s.getValueDefault()

	if def != "" {
		val = def
		def = fmt.Sprintf("[default: %s]", def)
	}

	if val != "" {
		val = "=" + val
	}

	long := s.long
	if long != "" {
		long = fmt.Sprintf("--%-20s", s.long+val)
	} else {
		short = short + val
	}

	return fmt.Sprintf("%5s %s %s %s", short, long, s.help, def)
}

func synopsis(spec *spec, form string) string {
	if spec.boolean {
		return form
	}
	return form + " " + strings.ToUpper(spec.long)
}
