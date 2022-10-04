package arg

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	scalar "github.com/alexflint/go-scalar"
)

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

// Arg represents a command line argument
type Argument struct {
	dest        path
	field       reflect.StructField // the struct field from which this option was created
	long        string              // the --long form for this option, or empty if none
	short       string              // the -s short form for this option, or empty if none
	cardinality cardinality         // determines how many tokens will be present (possible values: zero, one, multiple)
	required    bool                // if true, this option must be present on the command line
	positional  bool                // if true, this option will be looked for in the positional flags
	separate    bool                // if true, each slice and map entry will have its own --flag
	help        string              // the help text for this option
	env         string              // the name of the environment variable for this option, or empty for none
	defaultVal  string              // default value for this option
	placeholder string              // name of the data in help
}

// Command represents a named subcommand, or the top-level command
type Command struct {
	name        string
	help        string
	dest        path
	args        []*Argument
	subcommands []*Command
	parent      *Command
}

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

// Parser represents a set of command line options with destination values
type Parser struct {
	cmd      *Command      // the top-level command
	root     reflect.Value // destination struct to fill will values
	version  string        // version from the argument struct
	prologue string        // prologue for help text (from the argument struct)
	epilogue string        // epilogue for help text (from the argument struct)

	// the following fields are updated during processing of command line arguments
	leaf            *Command           // the subcommand we processed last
	accumulatedArgs []*Argument        // concatenation of the leaf subcommand's arguments plus all ancestors' arguments
	seen            map[*Argument]bool // the arguments we encountered while processing command line arguments
}

// Versioned is the interface that the destination struct should implement to
// make a version string appear at the top of the help message.
type Versioned interface {
	// Version returns the version string that will be printed on a line by itself
	// at the top of the help message.
	Version() string
}

// Described is the interface that the destination struct should implement to
// make a description string appear at the top of the help message.
type Described interface {
	// Description returns the string that will be printed on a line by itself
	// at the top of the help message.
	Description() string
}

// Epilogued is the interface that the destination struct should implement to
// add an epilogue string at the bottom of the help message.
type Epilogued interface {
	// Epilogue returns the string that will be printed on a line by itself
	// at the end of the help message.
	Epilogue() string
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

// the ParserOption interface matches options for the parser constructor
type ParserOption interface {
	parserOption()
}

type programNameParserOption struct {
	s string
}

func (programNameParserOption) parserOption() {}

// WithProgramName overrides the name of the program as displayed in help test
func WithProgramName(name string) ParserOption {
	return programNameParserOption{s: name}
}

// NewParser constructs a parser from a list of destination structs
func NewParser(dest interface{}, options ...ParserOption) (*Parser, error) {
	// check the destination type
	t := reflect.TypeOf(dest)
	if t.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("%s is not a pointer (did you forget an ampersand?)", t))
	}

	// pick a program name for help text and usage output
	program := "program"
	if len(os.Args) > 0 {
		program = filepath.Base(os.Args[0])
	}

	// apply the options
	for _, opt := range options {
		switch opt := opt.(type) {
		case programNameParserOption:
			program = opt.s
		}
	}

	// build the root command from the struct
	cmd, err := cmdFromStruct(program, path{}, t)
	if err != nil {
		return nil, err
	}

	// construct the parser
	p := Parser{
		seen: make(map[*Argument]bool),
		root: reflect.ValueOf(dest),
		cmd:  cmd,
	}

	// check for version, prologue, and epilogue
	if dest, ok := dest.(Versioned); ok {
		p.version = dest.Version()
	}
	if dest, ok := dest.(Described); ok {
		p.prologue = dest.Description()
	}
	if dest, ok := dest.(Epilogued); ok {
		p.epilogue = dest.Epilogue()
	}

	return &p, nil
}

func cmdFromStruct(name string, dest path, t reflect.Type) (*Command, error) {
	// commands can only be created from pointers to structs
	if t.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("subcommands must be pointers to structs but %s is a %s",
			dest, t.Kind())
	}

	t = t.Elem()
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("subcommands must be pointers to structs but %s is a pointer to %s",
			dest, t.Kind())
	}

	cmd := Command{
		name: name,
		dest: dest,
	}

	var errs []string
	walkFields(t, func(field reflect.StructField, t reflect.Type) bool {
		// check for the ignore switch in the tag
		tag := field.Tag.Get("arg")
		if tag == "-" {
			return false
		}

		// if this is an embedded struct then recurse into its fields, even if
		// it is unexported, because exported fields on unexported embedded
		// structs are still writable
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			return true
		}

		// ignore any other unexported field
		if !isExported(field.Name) {
			return false
		}

		// duplicate the entire path to avoid slice overwrites
		subdest := dest.Child(field)
		arg := Argument{
			dest:  subdest,
			field: field,
			long:  strings.ToLower(field.Name),
		}

		help, exists := field.Tag.Lookup("help")
		if exists {
			arg.help = help
		}

		defaultVal, hasDefault := field.Tag.Lookup("default")
		if hasDefault {
			arg.defaultVal = defaultVal
		}

		// Look at the tag
		var isSubcommand bool // tracks whether this field is a subcommand
		for _, key := range strings.Split(tag, ",") {
			if key == "" {
				continue
			}
			key = strings.TrimLeft(key, " ")
			var value string
			if pos := strings.Index(key, ":"); pos != -1 {
				value = key[pos+1:]
				key = key[:pos]
			}

			switch {
			case strings.HasPrefix(key, "---"):
				errs = append(errs, fmt.Sprintf("%s.%s: too many hyphens", t.Name(), field.Name))
			case strings.HasPrefix(key, "--"):
				arg.long = key[2:]
			case strings.HasPrefix(key, "-"):
				if len(key) != 2 {
					errs = append(errs, fmt.Sprintf("%s.%s: short arguments must be one character only",
						t.Name(), field.Name))
					return false
				}
				arg.short = key[1:]
			case key == "required":
				if hasDefault {
					errs = append(errs, fmt.Sprintf("%s.%s: 'required' cannot be used when a default value is specified",
						t.Name(), field.Name))
					return false
				}
				arg.required = true
			case key == "positional":
				arg.positional = true
			case key == "separate":
				arg.separate = true
			case key == "help": // deprecated
				arg.help = value
			case key == "env":
				// Use override name if provided
				if value != "" {
					arg.env = value
				} else {
					arg.env = strings.ToUpper(field.Name)
				}
			case key == "subcommand":
				// decide on a name for the subcommand
				cmdname := value
				if cmdname == "" {
					cmdname = strings.ToLower(field.Name)
				}

				// parse the subcommand recursively
				subcmd, err := cmdFromStruct(cmdname, subdest, field.Type)
				if err != nil {
					errs = append(errs, err.Error())
					return false
				}

				subcmd.parent = &cmd
				subcmd.help = field.Tag.Get("help")

				cmd.subcommands = append(cmd.subcommands, subcmd)
				isSubcommand = true
			default:
				errs = append(errs, fmt.Sprintf("unrecognized tag '%s' on field %s", key, tag))
				return false
			}
		}

		placeholder, hasPlaceholder := field.Tag.Lookup("placeholder")
		if hasPlaceholder {
			arg.placeholder = placeholder
		} else if arg.long != "" {
			arg.placeholder = strings.ToUpper(arg.long)
		} else {
			arg.placeholder = strings.ToUpper(arg.field.Name)
		}

		// Check whether this field is supported. It's good to do this here rather than
		// wait until ParseValue because it means that a program with invalid argument
		// fields will always fail regardless of whether the arguments it received
		// exercised those fields.
		if !isSubcommand {
			cmd.args = append(cmd.args, &arg)

			var err error
			arg.cardinality, err = cardinalityOf(field.Type)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s.%s: %s fields are not supported",
					t.Name(), field.Name, field.Type.String()))
				return false
			}
			if arg.cardinality == multiple && hasDefault {
				errs = append(errs, fmt.Sprintf("%s.%s: default values are not supported for slice or map fields",
					t.Name(), field.Name))
				return false
			}
		}

		// if this was an embedded field then we already returned true up above
		return false
	})

	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n"))
	}

	// check that we don't have both positionals and subcommands
	var hasPositional bool
	for _, arg := range cmd.args {
		if arg.positional {
			hasPositional = true
		}
	}
	if hasPositional && len(cmd.subcommands) > 0 {
		return nil, fmt.Errorf("%s cannot have both subcommands and positional arguments", dest)
	}

	return &cmd, nil
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

// ProcessCommandLine goes through arguments one-by-one, parses them,
// and assigns the result to the underlying struct field. It returns
// an error if an argument is invalid or an option is unknown, not if a
// required argument is missing. To check that all required arguments
// are set, call CheckRequired(). This function ignores the first element
// of args, which is assumed to be the program name itself.
func (p *Parser) ProcessCommandLine(args []string) error {
	positionals, err := p.ProcessOptions(args)
	if err != nil {
		return err
	}
	return p.ProcessPositionals(positionals)
}

// ProcessOptions process command line arguments but does not process
// positional arguments. Instead, it returns positionals. These can then
// be passed to ProcessPositionals. This function ignores the first element
// of args, which is assumed to be the program name itself.
func (p *Parser) ProcessOptions(args []string) ([]string, error) {
	// union of args for the chain of subcommands encountered so far
	curCmd := p.cmd
	p.leaf = curCmd

	// we will add to this list each time we expand a subcommand
	p.accumulatedArgs = make([]*Argument, len(curCmd.args))
	copy(p.accumulatedArgs, curCmd.args)

	// process each string from the command line
	var allpositional bool
	var positionals []string

	// must use explicit for loop, not range, because we manipulate i inside the loop
	for i := 1; i < len(args); i++ {
		token := args[i]
		if token == "--" {
			allpositional = true
			continue
		}

		if !isFlag(token) || allpositional {
			// each subcommand can have either subcommands or positionals, but not both
			if len(curCmd.subcommands) == 0 {
				positionals = append(positionals, token)
				continue
			}

			// if we have a subcommand then make sure it is valid for the current context
			subcmd := findSubcommand(curCmd.subcommands, token)
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

			curCmd = subcmd
			p.leaf = curCmd
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
		p.seen[arg] = true

		// deal with the case of multiple values
		if arg.cardinality == multiple {
			var values []string
			if value == "" {
				for i+1 < len(args) && !isFlag(args[i+1]) && args[i+1] != "--" {
					values = append(values, args[i+1])
					i++
					if arg.separate {
						break
					}
				}
			} else {
				values = append(values, value)
			}
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

		p.seen[arg] = true
		err := scalar.ParseValue(p.val(arg.dest), value)
		if err != nil {
			return nil, fmt.Errorf("error processing %s: %v", token, err)
		}
	}

	return positionals, nil
}

// ProcessPositionals processes a list of positional arguments. It is assumed
// that options such as --abc and --abc=123 have already been removed. If
// this list contains tokens that begin with a hyphen they will still be
// treated as positional arguments.
func (p *Parser) ProcessPositionals(positionals []string) error {
	for _, arg := range p.accumulatedArgs {
		if !arg.positional {
			continue
		}
		if len(positionals) == 0 {
			break
		}
		p.seen[arg] = true
		if arg.cardinality == multiple {
			err := setSliceOrMap(p.val(arg.dest), positionals, true)
			if err != nil {
				return fmt.Errorf("error processing %s: %v", arg.field.Name, err)
			}
			positionals = nil
		} else {
			err := scalar.ParseValue(p.val(arg.dest), positionals[0])
			if err != nil {
				return fmt.Errorf("error processing %s: %v", arg.field.Name, err)
			}
			positionals = positionals[1:]
		}
	}

	if len(positionals) > 0 {
		return fmt.Errorf("too many positional arguments at '%s'", positionals[0])
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

// isFlag returns true if a token is a flag such as "-v" or "--user" but not "-" or "--"
func isFlag(s string) bool {
	return strings.HasPrefix(s, "-") && strings.TrimLeft(s, "-") != ""
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
