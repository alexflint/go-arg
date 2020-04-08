package arg

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// the width of the left column
const colWidth = 25

// to allow monkey patching in tests
var stderr = os.Stderr

// Fail prints usage information to stderr and exits with non-zero status
func (p *Parser) Fail(msg string) {
	p.failWithCommand(msg, p.cmd)
}

// failWithCommand prints usage information for the given subcommand to stderr and exits with non-zero status
func (p *Parser) failWithCommand(msg string, cmd *command) {
	p.writeUsageForCommand(stderr, cmd)
	fmt.Fprintln(stderr, "error:", msg)
	osExit(-1)
}

// WriteUsage writes usage information to the given writer
func (p *Parser) WriteUsage(w io.Writer) {
	cmd := p.cmd
	if p.lastCmd != nil {
		cmd = p.lastCmd
	}
	p.writeUsageForCommand(w, cmd)
}

// writeUsageForCommand writes usage information for the given subcommand
func (p *Parser) writeUsageForCommand(w io.Writer, cmd *command) {
	var positionals, options []*spec
	for _, spec := range cmd.specs {
		if spec.positional {
			positionals = append(positionals, spec)
		} else {
			options = append(options, spec)
		}
	}

	if p.version != "" {
		fmt.Fprintln(w, p.version)
	}

	// make a list of ancestor commands so that we print with full context
	var ancestors []string
	ancestor := cmd
	for ancestor != nil {
		ancestors = append(ancestors, ancestor.name)
		ancestor = ancestor.parent
	}

	// print the beginning of the usage string
	fmt.Fprint(w, "Usage:")
	for i := len(ancestors) - 1; i >= 0; i-- {
		fmt.Fprint(w, " "+ancestors[i])
	}

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
		if spec.multiple {
			if !spec.required {
				fmt.Fprint(w, "[")
			}
			fmt.Fprintf(w, "%s [%s ...]", spec.placeholder, spec.placeholder)
			if !spec.required {
				fmt.Fprint(w, "]")
			}
		} else {
			fmt.Fprint(w, spec.placeholder)
		}
	}

	// if the program supports subcommands, give a hint to the user about their existence
	if len(cmd.subcommands) > 0 {
		fmt.Fprint(w, " <command> [<args>]")
	}

	fmt.Fprint(w, "\n")
}

func printTwoCols(w io.Writer, left, help string, defaultVal string) {
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
	if defaultVal != "" {
		fmt.Fprintf(w, " [default: %s]", defaultVal)
	}
	fmt.Fprint(w, "\n")
}

// WriteHelp writes the usage string followed by the full help string for each option
func (p *Parser) WriteHelp(w io.Writer) {
	cmd := p.cmd
	if p.lastCmd != nil {
		cmd = p.lastCmd
	}
	p.writeHelpForCommand(w, cmd)
}

// writeHelp writes the usage string for the given subcommand
func (p *Parser) writeHelpForCommand(w io.Writer, cmd *command) {
	var positionals, options, requireds, requiredIf []*spec
	for _, spec := range cmd.specs {
		if spec.required {
			requireds = append(requireds, spec)
		} else if len(spec.requiredIf) > 0 {
			requiredIf = append(requiredIf, spec)
		}
		if spec.positional {
			positionals = append(positionals, spec)
		} else {
			options = append(options, spec)
		}
	}

	if p.description != "" {
		fmt.Fprintln(w, p.description)
	}
	p.writeUsageForCommand(w, cmd)

	//write the list of required arguments
	if len(requireds) > 0 {
		fmt.Fprint(w, "\nRequired arguments:\n")
		for _, spec := range requireds {
			printTwoCols(w, spec.placeholder, spec.help, "")
		}
	}

	//write the list of required-if list arguments
	if len(requiredIf) > 0 {
		fmt.Fprint(w, "\nConditionally required arguments:\n")
		for _, spec := range requiredIf {
			requiredString := "required if: "
			for index, v := range spec.requiredIf {
				requiredString += v
				if index+1 < len(spec.requiredIf) {
					requiredString += " ,"
				} else {
					requiredString += " has be set"
				}
			}
			printTwoCols(w, spec.placeholder, requiredString, "")
		}
	}

	// write the list of positionals
	if len(positionals) > 0 {
		fmt.Fprint(w, "\nPositional arguments:\n")
		for _, spec := range positionals {
			printTwoCols(w, spec.placeholder, spec.help, "")
		}
	}

	// write the list of options
	if len(options) > 0 || cmd.parent == nil {
		fmt.Fprint(w, "\nOptions:\n")
		for _, spec := range options {
			p.printOption(w, spec)
		}
	}

	// obtain a flattened list of options from all ancestors
	var globals []*spec
	ancestor := cmd.parent
	for ancestor != nil {
		globals = append(globals, ancestor.specs...)
		ancestor = ancestor.parent
	}

	// write the list of global options
	if len(globals) > 0 {
		fmt.Fprint(w, "\nGlobal options:\n")
		for _, spec := range globals {
			p.printOption(w, spec)
		}
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
	if len(cmd.subcommands) > 0 {
		fmt.Fprint(w, "\nCommands:\n")
		for _, subcmd := range cmd.subcommands {
			printTwoCols(w, subcmd.name, subcmd.help, "")
		}
	}
}

func (p *Parser) printOption(w io.Writer, spec *spec) {
	left := synopsis(spec, "--"+spec.long)
	if spec.short != "" {
		left += ", " + synopsis(spec, "-"+spec.short)
	}
	printTwoCols(w, left, spec.help, spec.defaultVal)
}

func synopsis(spec *spec, form string) string {
	if spec.boolean {
		return form
	}
	return form + " " + spec.placeholder
}

func ptrTo(s string) *string {
	return &s
}
