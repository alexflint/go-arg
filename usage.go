package arg

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
)

// Usage prints usage information to stdout information and exits with status zero
func Usage(dest ...interface{}) {
	if err := WriteUsage(os.Stdout, dest...); err != nil {
		fmt.Println(err)
	}
	os.Exit(0)
}

// Fail prints usage information to stdout and exits with non-zero status
func Fail(msg string, dest ...interface{}) {
	fmt.Println(msg)
	if err := WriteUsage(os.Stdout, dest...); err != nil {
		fmt.Println(err)
	}
	os.Exit(1)
}

// WriteUsage writes usage information to the given writer
func WriteUsage(w io.Writer, dest ...interface{}) error {
	spec, err := extractSpec(dest...)
	if err != nil {
		return err
	}
	writeUsage(w, spec)
	return nil
}

func synopsis(spec *spec, form string) string {
	if spec.dest.Kind() == reflect.Bool {
		return form
	} else {
		return form + " " + strings.ToUpper(spec.long)
	}
}

// writeUsage writes usage information to the given writer
func writeUsage(w io.Writer, specs []*spec) {
	var positionals, options []*spec
	for _, spec := range specs {
		if spec.positional {
			positionals = append(positionals, spec)
		} else {
			options = append(options, spec)
		}
	}

	fmt.Fprint(w, "usage: ")

	// write the option component of the one-line usage message
	for _, spec := range options {
		if !spec.required {
			fmt.Fprint(w, "[")
		}
		fmt.Fprint(w, synopsis(spec, "--"+spec.long))
		if !spec.required {
			fmt.Fprint(w, "]")
		}
		fmt.Fprint(w, " ")
	}

	// write the positional component of the one-line usage message
	for _, spec := range positionals {
		up := strings.ToUpper(spec.long)
		if spec.multiple {
			fmt.Fprintf(w, "[%s [%s ...]]", up)
		} else {
			fmt.Fprint(w, up)
		}
		fmt.Fprint(w, " ")
	}
	fmt.Fprint(w, "\n")

	// write the list of positionals
	if len(positionals) > 0 {
		fmt.Fprint(w, "\npositional arguments:\n")
		for _, spec := range positionals {
			fmt.Fprintf(w, "  %s\n", spec.long)
		}
	}

	// write the list of options
	if len(options) > 0 {
		fmt.Fprint(w, "\noptions:\n")
		const colWidth = 25
		for _, spec := range options {
			left := fmt.Sprint(synopsis(spec, "--"+spec.long))
			if spec.short != "" {
				left += ", " + fmt.Sprint(synopsis(spec, "-"+spec.short))
			}
			fmt.Print(left)
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
	}
}
