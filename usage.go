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
	if p.version != "" {
		fmt.Fprintln(w, p.version)
	}

	fmt.Fprintf(w, "usage: %s ", p.config.Program)

	// write the option component of the usage message
	for _, s := range p.spec {
		if !s.positional {
			s.WriteUsage(w)
		}
	}

	// write the positional component of the usage message
	for _, s := range p.spec {
		if s.positional {
			s.WriteUsagePositional(w)
		}
	}

	fmt.Fprintln(w, "")
}

// WriteHelp writes the usage string followed by the full help string for each option
func (p *Parser) WriteHelp(w io.Writer) {
	//write description
	if p.description != "" {
		fmt.Fprintln(w, p.description)
	}

	//write usage
	p.WriteUsage(w)

	//write positional
	var positionalHeader bool
	for _, s := range p.spec {
		if s.positional {
			if !positionalHeader {
				fmt.Fprintln(w, "\npositional arguments:")
				positionalHeader = true
			}
			s.WritePositional(w)
		}
	}

	//write options
	fmt.Fprintln(w, "\noptions:")
	for _, s := range p.spec {
		if !s.positional {
			s.WriteOption(w)
		}
	}
}

func (s *spec) WriteUsage(w io.Writer) {
	if s.long == "help" || s.long == "version" {
		return
	}

	var name string
	if s.short != "" {
		name = fmt.Sprintf("-%s", s.short)
	} else {
		name = fmt.Sprintf("--%s", s.long)
	}

	if s.required {
		fmt.Fprintf(w, "%s ", name)
	} else {
		fmt.Fprintf(w, "[%s] ", name)
	}
}

func (s *spec) WriteUsagePositional(w io.Writer) {
	name := s.long
	if name == "" {
		name = s.short
	}
	name = strings.ToUpper(name)
	if s.multiple {
		fmt.Fprintf(w, "[%s [%s ...]] ", name, name)
	} else {
		fmt.Fprintf(w, "%s ", name)
	}
}

func (s *spec) WritePositional(w io.Writer) {
	name := s.long
	if name == "" {
		name = s.short
	}

	if len(name) > 24 && s.help != "" {
		name = fmt.Sprintf("%s\n%28s", name, "")
	}

	fmt.Fprintf(w, "  %-26s %s\n", name, s.help)
}

func (s *spec) WriteOption(w io.Writer) {
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

	if len(long) > 24 && s.help != "" {
		long = fmt.Sprintf("%s\n%28s", long, "")
	}

	fmt.Fprintf(w, "%5s %s %s %s\n", short, long, s.help, def)
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

func synopsis(spec *spec, form string) string {
	if spec.boolean {
		return form
	}
	return form + " " + strings.ToUpper(spec.long)
}
