package arg

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	scalar "github.com/alexflint/go-scalar"
)

// ErrHelp indicates that -h or --help were provided
var ErrHelp = errors.New("help requested by user")

// ErrVersion indicates that --version was provided
var ErrVersion = errors.New("version requested by user")

// MustParse processes command line arguments and exits upon failure
func MustParse(dest interface{}) *Parser {
	p, err := NewParser(dest)
	if err != nil {
		fmt.Fprintln(stdout, err)
		osExit(-1)
		return nil // just in case osExit was monkey-patched
	}

	err = p.Parse(os.Args, os.Environ())
	switch {
	case err == ErrHelp:
		p.writeHelpForSubcommand(stdout, p.leaf)
		osExit(0)
	case err == ErrVersion:
		fmt.Fprintln(stdout, p.version)
		osExit(0)
	case err != nil:
		p.failWithSubcommand(err.Error(), p.leaf)
	}

	return p
}

// Parse processes command line arguments and stores them in dest
func Parse(dest interface{}, options ...ParserOption) error {
	p, err := NewParser(dest, options...)
	if err != nil {
		return err
	}
	return p.Parse(os.Args, os.Environ())
}

// Parse processes the given command line option, storing the results in the field
// of the structs from which NewParser was constructed
func (p *Parser) Parse(args, env []string) error {
	p.seen = make(map[*Argument]bool)

	// If -h or --help were specified then make sure help text supercedes other errors
	var help bool
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			help = true
		}
		if arg == "--" {
			break
		}
	}

	err := p.ProcessCommandLine(args)
	if err != nil {
		if help {
			return ErrHelp
		}
		return err
	}

	err = p.ProcessEnvironment(env)
	if err != nil {
		if help {
			return ErrHelp
		}
		return err
	}

	err = p.ProcessDefaults()
	if err != nil {
		if help {
			return ErrHelp
		}
		return err
	}

	return p.Validate()
}

// ProcessCommandLine scans arguments one-by-one, parses them and assigns
// the result to fields of the struct passed to NewParser. It returns
// an error if an argument is invalid or unknown, but not if a
// required argument is missing. To check that all required arguments
// are set, call Validate(). This function ignores the first element
// of args, which is assumed to be the program name itself. This function
// never overwrites arguments previously seen in a call to any Process*
// function.
func (p *Parser) ProcessCommandLine(args []string) error {
	positionals, err := p.ProcessOptions(args)
	if err != nil {
		return err
	}
	return p.ProcessPositionals(positionals)
}

// OverwriteWithCommandLine is like ProcessCommandLine but it overwrites
// any previously seen values.
func (p *Parser) OverwriteWithCommandLine(args []string) error {
	positionals, err := p.OverwriteWithOptions(args)
	if err != nil {
		return err
	}
	return p.OverwriteWithPositionals(positionals)
}

// ProcessOptions processes options but not positionals from the
// command line. Positionals are returned and can be passed to
// ProcessPositionals. This function ignores the first element of args,
// which is assumed to be the program name itself. Arguments seen
// in a previous call to any Process* or OverwriteWith* functions
// are ignored.
func (p *Parser) ProcessOptions(args []string) ([]string, error) {
	return p.processOptions(args, false)
}

// OverwriteWithOptions is like ProcessOptions except previously seen
// arguments are overwritten
func (p *Parser) OverwriteWithOptions(args []string) ([]string, error) {
	return p.processOptions(args, true)
}

func (p *Parser) processOptions(args []string, overwrite bool) ([]string, error) {
	// union of args for the chain of subcommands encountered so far
	p.leaf = p.cmd

	// we will add to this list each time we expand a subcommand
	p.accumulatedArgs = make([]*Argument, len(p.leaf.args))
	copy(p.accumulatedArgs, p.leaf.args)

	// process each string from the command line
	var allpositional bool
	var positionals []string

	// must use explicit for loop, not range, because we manipulate i inside the loop
	for i := 1; i < len(args); i++ {
		token := args[i]

		// the "--" token indicates that all further tokens should be treated as positionals
		if token == "--" {
			allpositional = true
			continue
		}

		// check whether this is a positional argument
		if !isFlag(token) || allpositional {
			// each subcommand can have either subcommands or positionals, but not both
			if len(p.leaf.subcommands) == 0 {
				positionals = append(positionals, token)
				continue
			}

			// if we have a subcommand then make sure it is valid for the current context
			subcmd := findSubcommand(p.leaf.subcommands, token)
			if subcmd == nil {
				return nil, fmt.Errorf("invalid subcommand: %s", token)
			}

			// instantiate the field to point to a new struct
			v := p.val(subcmd.dest)
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem())) // we already checked that all subcommands are struct pointers
			}

			// add the new options to the set of allowed options
			p.accumulatedArgs = append(p.accumulatedArgs, subcmd.args...)
			p.leaf = subcmd
			continue
		}

		// check for special --help and --version flags
		switch token {
		case "-h", "--help":
			return nil, ErrHelp
		case "--version":
			return nil, ErrVersion
		}

		// check for an equals sign, as in "--foo=bar"
		var value string
		opt := strings.TrimLeft(token, "-")
		if pos := strings.Index(opt, "="); pos != -1 {
			value = opt[pos+1:]
			opt = opt[:pos]
		}

		// look up the arg for this option (note that the "args" slice changes as
		// we expand subcommands so it is better not to use a map)
		arg := findOption(p.accumulatedArgs, opt)
		if arg == nil {
			return nil, fmt.Errorf("unknown argument %s", token)
		}

		// deal with the case of multiple values
		if arg.cardinality == multiple {
			// if arg.separate is true then just parse one value and append it
			if arg.separate {
				if value == "" {
					if i+1 == len(args) {
						return nil, fmt.Errorf("missing value for %s", token)
					}
					if isFlag(args[i+1]) {
						return nil, fmt.Errorf("missing value for %s", token)
					}
					value = args[i+1]
					i++
				}

				err := appendToSliceOrMap(p.val(arg.dest), value)
				if err != nil {
					return nil, fmt.Errorf("error processing %s: %v", token, err)
				}
				p.seen[arg] = true
				continue
			}

			// if args.separate is not true then consume tokens until next --option
			var values []string
			if value == "" {
				for i+1 < len(args) && !isFlag(args[i+1]) && args[i+1] != "--" {
					values = append(values, args[i+1])
					i++
				}
			} else {
				values = append(values, value)
			}

			// this is the first time we can check p.seen because we need to correctly
			// increment i above, even when we then ignore the value
			if p.seen[arg] && !overwrite {
				continue
			}

			// store the values into the slice or map
			err := setSliceOrMap(p.val(arg.dest), values, !arg.separate)
			if err != nil {
				return nil, fmt.Errorf("error processing %s: %v", token, err)
			}
			continue
		}

		// if it's a flag and it has no value then set the value to true
		// use boolean because this takes account of TextUnmarshaler
		if arg.cardinality == zero && value == "" {
			value = "true"
		}

		// if we have something like "--foo" then the value is the next argument
		if value == "" {
			if i+1 == len(args) {
				return nil, fmt.Errorf("missing value for %s", token)
			}
			if isFlag(args[i+1]) {
				return nil, fmt.Errorf("missing value for %s", token)
			}
			value = args[i+1]
			i++
		}

		// this is the first time we can check p.seen because we need to correctly
		// increment i above, even when we then ignore the value
		if p.seen[arg] && !overwrite {
			continue
		}

		err := scalar.ParseValue(p.val(arg.dest), value)
		if err != nil {
			return nil, fmt.Errorf("error processing %s: %v", token, err)
		}
		p.seen[arg] = true
	}

	return positionals, nil
}

// ProcessPositionals processes a list of positional arguments. If
// this list contains tokens that begin with a hyphen they will still be
// treated as positional arguments. Arguments seen in a previous call
// to any Process* or OverwriteWith* functions are ignored.
func (p *Parser) ProcessPositionals(positionals []string) error {
	return p.processPositionals(positionals, false)
}

// OverwriteWithPositionals is like ProcessPositionals except previously
// seen arguments are overwritten.
func (p *Parser) OverwriteWithPositionals(positionals []string) error {
	return p.processPositionals(positionals, true)
}

func (p *Parser) processPositionals(positionals []string, overwrite bool) error {
	for _, arg := range p.accumulatedArgs {
		if !arg.positional {
			continue
		}
		if len(positionals) == 0 {
			break
		}
		if arg.cardinality == multiple {
			if !p.seen[arg] || overwrite {
				err := setSliceOrMap(p.val(arg.dest), positionals, true)
				if err != nil {
					return fmt.Errorf("error processing %s: %v", arg.field.Name, err)
				}
			}
			positionals = nil
		} else {
			if !p.seen[arg] || overwrite {
				err := scalar.ParseValue(p.val(arg.dest), positionals[0])
				if err != nil {
					return fmt.Errorf("error processing %s: %v", arg.field.Name, err)
				}
			}
			positionals = positionals[1:]
		}
		p.seen[arg] = true
	}

	if len(positionals) > 0 {
		return fmt.Errorf("too many positional arguments at '%s'", positionals[0])
	}

	return nil
}

// ProcessEnvironment processes environment variables from a list of strings
// of the form KEY=VALUE. You can pass in os.Environ(). It
// does not overwrite any fields with values already populated.
func (p *Parser) ProcessEnvironment(environ []string) error {
	return p.processEnvironment(environ, false)
}

// OverwriteWithEnvironment processes environment variables from a list
// of strings of the form "KEY=VALUE". Any existing values are overwritten.
func (p *Parser) OverwriteWithEnvironment(environ []string) error {
	return p.processEnvironment(environ, true)
}

// ProcessEnvironment processes environment variables from a list of strings
// of the form KEY=VALUE. You can pass in os.Environ(). It
// overwrites already-populated fields only if overwrite is true.
func (p *Parser) processEnvironment(environ []string, overwrite bool) error {
	// parse the list of KEY=VAL strings in environ
	env := make(map[string]string)
	for _, s := range environ {
		if i := strings.Index(s, "="); i >= 0 {
			env[s[:i]] = s[i+1:]
		}
	}

	// process arguments one-by-one
	for _, arg := range p.accumulatedArgs {
		if p.seen[arg] && !overwrite {
			continue
		}

		if arg.env == "" {
			continue
		}

		value, found := env[arg.env]
		if !found {
			continue
		}

		if arg.cardinality == multiple {
			// expect a CSV string in an environment
			// variable in the case of multiple values
			var values []string
			var err error
			if len(strings.TrimSpace(value)) > 0 {
				values, err = csv.NewReader(strings.NewReader(value)).Read()
				if err != nil {
					return fmt.Errorf(
						"error reading a CSV string from environment variable %s with multiple values: %v",
						arg.env,
						err,
					)
				}
			}
			if err = setSliceOrMap(p.val(arg.dest), values, !arg.separate); err != nil {
				return fmt.Errorf(
					"error processing environment variable %s with multiple values: %v",
					arg.env,
					err,
				)
			}
		} else {
			if err := scalar.ParseValue(p.val(arg.dest), value); err != nil {
				return fmt.Errorf("error processing environment variable %s: %v", arg.env, err)
			}
		}

		p.seen[arg] = true
	}

	return nil
}

// ProcessDefaults assigns default values to all fields that have default values and
// are not already populated.
func (p *Parser) ProcessDefaults() error {
	return p.processDefaults(false)
}

// OverwriteWithDefaults assigns default values to all fields that have default values,
// overwriting any previous value
func (p *Parser) OverwriteWithDefaults() error {
	return p.processDefaults(true)
}

// processDefaults assigns default values to all fields in all expanded subcommands.
// If overwrite is true then it overwrites existing values.
func (p *Parser) processDefaults(overwrite bool) error {
	for _, arg := range p.accumulatedArgs {
		if p.seen[arg] && !overwrite {
			continue
		}

		if arg.defaultVal == "" {
			continue
		}

		name := strings.ToLower(arg.field.Name)
		if arg.long != "" && !arg.positional {
			name = "--" + arg.long
		}

		err := scalar.ParseValue(p.val(arg.dest), arg.defaultVal)
		if err != nil {
			return fmt.Errorf("error processing default value for %s: %v", name, err)
		}
		p.seen[arg] = true
	}

	return nil
}

// Missing returns a list of required arguments that were not provided
func (p *Parser) Missing() []*Argument {
	var missing []*Argument
	for _, arg := range p.accumulatedArgs {
		if arg.required && !p.seen[arg] {
			missing = append(missing, arg)
		}
	}
	return missing
}

// Validate returns an error if any required arguments were missing
func (p *Parser) Validate() error {
	if missing := p.Missing(); len(missing) > 0 {
		name := strings.ToLower(missing[0].field.Name)
		if missing[0].long != "" && !missing[0].positional {
			name = "--" + missing[0].long
		}

		if missing[0].env == "" {
			return fmt.Errorf("%s is required", name)
		}
		return fmt.Errorf("%s is required (or environment variable %s)", name, missing[0].env)
	}

	return nil
}

// val returns a reflect.Value corresponding to the current value for the
// given path
func (p *Parser) val(dest path) reflect.Value {
	v := p.root
	for _, field := range dest.fields {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return reflect.Value{}
			}
			v = v.Elem()
		}

		v = v.FieldByIndex(field.Index)
	}
	return v
}

// path represents a sequence of steps to find the output location for an
// argument or subcommand in the final destination struct
type path struct {
	fields []reflect.StructField // sequence of struct fields to traverse
}

// String gets a string representation of the given path
func (p path) String() string {
	s := "args"
	for _, f := range p.fields {
		s += "." + f.Name
	}
	return s
}

// Child gets a new path representing a child of this path.
func (p path) Child(f reflect.StructField) path {
	// copy the entire slice of fields to avoid possible slice overwrite
	subfields := make([]reflect.StructField, len(p.fields)+1)
	copy(subfields, p.fields)
	subfields[len(subfields)-1] = f
	return path{
		fields: subfields,
	}
}

// walkFields calls a function for each field of a struct, recursively expanding struct fields.
func walkFields(t reflect.Type, visit func(field reflect.StructField, owner reflect.Type) bool) {
	walkFieldsImpl(t, visit, nil)
}

func walkFieldsImpl(t reflect.Type, visit func(field reflect.StructField, owner reflect.Type) bool, path []int) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		field.Index = make([]int, len(path)+1)
		copy(field.Index, append(path, i))
		expand := visit(field, t)
		if expand && field.Type.Kind() == reflect.Struct {
			var subpath []int
			if field.Anonymous {
				subpath = append(path, i)
			}
			walkFieldsImpl(field.Type, visit, subpath)
		}
	}
}

// isFlag returns true if a token is a flag such as "-v" or "--user" but not "-" or "--"
func isFlag(s string) bool {
	return strings.HasPrefix(s, "-") && strings.TrimLeft(s, "-") != ""
}

// findOption finds an option from its name, or returns nil if no arg is found
func findOption(args []*Argument, name string) *Argument {
	for _, arg := range args {
		if arg.positional {
			continue
		}
		if arg.long == name || arg.short == name {
			return arg
		}
	}
	return nil
}

// findSubcommand finds a subcommand using its name, or returns nil if no subcommand is found
func findSubcommand(cmds []*Command, name string) *Command {
	for _, cmd := range cmds {
		if cmd.name == name {
			return cmd
		}
	}
	return nil
}
