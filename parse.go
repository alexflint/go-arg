package arg

import (
	"encoding"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	scalar "github.com/alexflint/go-scalar"
)

// path represents a sequence of steps to find the output location for an
// argument or subcommand in the final destination struct
type path struct {
	root   int                   // index of the destination struct
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
		root:   p.root,
		fields: subfields,
	}
}

// spec represents a command line option
type spec struct {
	dest          path
	field         reflect.StructField // the struct field from which this option was created
	long          string              // the --long form for this option, or empty if none
	short         string              // the -s short form for this option, or empty if none
	cardinality   cardinality         // determines how many tokens will be present (possible values: zero, one, multiple)
	required      bool                // if true, this option must be present on the command line
	positional    bool                // if true, this option will be looked for in the positional flags
	separate      bool                // if true, each slice and map entry will have its own --flag
	help          string              // the help text for this option
	env           string              // the name of the environment variable for this option, or empty for none
	defaultValue  reflect.Value       // default value for this option
	defaultString string              // default value for this option, in string form to be displayed in help text
	placeholder   string              // name of the data in help
}

// command represents a named subcommand, or the top-level command
type command struct {
	name        string
	help        string
	dest        path
	options     []*spec
	subcommands []*command
	groups      []*command
	parent      *command
}

// specs gets all the specs from this command plus all nested option groups,
// recursively through descendants
func (cmd command) specs() []*spec {
	var specs []*spec
	specs = append(specs, cmd.options...)
	for _, grpcmd := range cmd.groups {
		specs = append(specs, grpcmd.specs()...)
	}
	return specs
}

// ErrHelp indicates that -h or --help were provided
var ErrHelp = errors.New("help requested by user")

// ErrVersion indicates that --version was provided
var ErrVersion = errors.New("version requested by user")

// for monkey patching in example code
var mustParseExit = os.Exit

// MustParse processes command line arguments and exits upon failure
func MustParse(dest ...interface{}) *Parser {
	return mustParse(Config{Exit: mustParseExit}, dest...)
}

// mustParse is a helper that facilitates testing
func mustParse(config Config, dest ...interface{}) *Parser {
	if config.Exit == nil {
		config.Exit = os.Exit
	}
	if config.Out == nil {
		config.Out = os.Stdout
	}

	p, err := NewParser(config, dest...)
	if err != nil {
		fmt.Fprintln(config.Out, err)
		config.Exit(-1)
		return nil
	}

	p.MustParse(flags())
	return p
}

// Parse processes command line arguments and stores them in dest
func Parse(dest ...interface{}) error {
	p, err := NewParser(Config{}, dest...)
	if err != nil {
		return err
	}
	return p.Parse(flags())
}

// flags gets all command line arguments other than the first (program name)
func flags() []string {
	if len(os.Args) == 0 { // os.Args could be empty
		return nil
	}
	return os.Args[1:]
}

// Config represents configuration options for an argument parser
type Config struct {
	// Program is the name of the program used in the help text
	Program string

	// IgnoreEnv instructs the library not to read environment variables
	IgnoreEnv bool

	// IgnoreDefault instructs the library not to reset the variables to the
	// default values, including pointers to sub commands
	IgnoreDefault bool

	// StrictSubcommands intructs the library not to allow global commands after
	// subcommand
	StrictSubcommands bool

	// Exit is called to terminate the process with an error code (defaults to os.Exit)
	Exit func(int)

	// Out is where help text, usage text, and failure messages are printed (defaults to os.Stdout)
	Out io.Writer
}

// Parser represents a set of command line options with destination values
type Parser struct {
	cmd         *command
	roots       []reflect.Value
	config      Config
	version     string
	description string
	epilogue    string

	// the following field changes during processing of command line arguments
	lastCmd *command
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

// NewParser constructs a parser from a list of destination structs
func NewParser(config Config, dests ...interface{}) (*Parser, error) {
	// fill in defaults
	if config.Exit == nil {
		config.Exit = os.Exit
	}
	if config.Out == nil {
		config.Out = os.Stdout
	}

	// first pick a name for the command for use in the usage text
	var name string
	switch {
	case config.Program != "":
		name = config.Program
	case len(os.Args) > 0:
		name = filepath.Base(os.Args[0])
	default:
		name = "program"
	}

	// construct a parser
	p := Parser{
		cmd:    &command{name: name},
		config: config,
	}

	// make a list of roots
	for _, dest := range dests {
		p.roots = append(p.roots, reflect.ValueOf(dest))
	}

	// process each of the destination values
	for _, dest := range dests {
		t := reflect.TypeOf(dest)
		if t.Kind() != reflect.Ptr {
			panic(fmt.Sprintf("%s is not a pointer (did you forget an ampersand?)", t))
		}

		err := p.cmd.parseFieldsFromStructPointer(t, false)
		if err != nil {
			return nil, err
		}

		// for backwards compatibility, add nonzero field values as defaults
		// this applies only to the top-level command, not to subcommands (this inconsistency
		// is the reason that this method for setting default values was deprecated)
		for _, spec := range p.cmd.specs() {
			// get the value
			defaultString, defaultValue, err := p.defaultVal(spec.dest)
			if err != nil {
				return nil, err
			}

			// if the value is the "zero value" (e.g. nil pointer, empty struct) then ignore
			if defaultString == "" {
				continue
			}

			// store as a default
			spec.defaultString = defaultString
			spec.defaultValue = defaultValue
		}

		if dest, ok := dest.(Versioned); ok {
			p.version = dest.Version()
		}
		if dest, ok := dest.(Described); ok {
			p.description = dest.Description()
		}
		if dest, ok := dest.(Epilogued); ok {
			p.epilogue = dest.Epilogue()
		}
	}

	return &p, nil
}

// parseFieldsFromStructPointer ensures the destination structure is a pointer
// to a struct. This function should be called when parsing commands or
// subcommands as they can only be a struct pointer.
func (cmd *command) parseFieldsFromStructPointer(t reflect.Type, insideGroup bool) error {
	// commands can only be created from pointers to structs
	if t.Kind() != reflect.Ptr {
		return fmt.Errorf("subcommands must be pointers to structs but %s is a %s",
			cmd.dest, t.Kind())
	}

	t = t.Elem()
	if t.Kind() != reflect.Struct {
		return fmt.Errorf("subcommands must be pointers to structs but %s is a pointer to %s",
			cmd.dest, t.Kind())
	}
	return cmd.parseStruct(t, insideGroup)
}

// parseFieldsFromStructOrStructPointer ensures the destination structure is
// either a pointer to a struct, or a struct. This function should be called
// when parsing option groups as they can only be a struct, or a pointer to one.
func (cmd *command) parseFieldsFromStructOrStructPointer(t reflect.Type, insideGroup bool) error {
	// option groups can only be created from structs or pointers to structs
	typeHint := ""
	if t.Kind() == reflect.Ptr {
		typeHint = "a pointer to "
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return fmt.Errorf("option groups must be structs or pointers to structs, but %s is %s%s",
			cmd.dest, typeHint, t.Kind())
	}

	return cmd.parseStruct(t, insideGroup)
}

// parseStruct populates the command instance based on the type and annotations
// of the target struct. As these command instances are used for either (sub)
// commands or option groups, please refer to the parseFieldsFromStructPointer
// or parseFieldsFromStructOrStructPointer respectively.
func (cmd *command) parseStruct(t reflect.Type, insideGroup bool) error {
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
		subdest := cmd.dest.Child(field)
		spec := spec{
			dest:  subdest,
			field: field,
			long:  strings.ToLower(field.Name),
		}

		help, exists := field.Tag.Lookup("help")
		if exists {
			spec.help = help
		}

		// Look at the tag
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
				spec.long = key[2:]
			case strings.HasPrefix(key, "-"):
				if len(key) != 2 {
					errs = append(errs, fmt.Sprintf("%s.%s: short arguments must be one character only",
						t.Name(), field.Name))
					return false
				}
				spec.short = key[1:]
			case key == "required":
				spec.required = true
			case key == "positional":
				spec.positional = true
			case key == "separate":
				spec.separate = true
			case key == "help": // deprecated
				spec.help = value
			case key == "env":
				// Use override name if provided
				if value != "" {
					spec.env = value
				} else {
					spec.env = strings.ToUpper(field.Name)
				}
			case key == "subcommand":
				subCmd := command{
					name:   value,
					dest:   subdest,
					parent: cmd,
					help:   field.Tag.Get("help"),
				}
				cmd.subcommands = append(cmd.subcommands, &subCmd)

				if insideGroup {
					errs = append(errs, fmt.Sprintf("%s.%s: %s subcommands cannot be part of option groups",
						t.Name(), field.Name, field.Type.String()))
					return false
				}

				// decide on a name for the subcommand
				if subCmd.name == "" {
					subCmd.name = strings.ToLower(field.Name)
				}

				// parse the subcommand recursively
				err := subCmd.parseFieldsFromStructPointer(field.Type, false)
				if err != nil {
					errs = append(errs, err.Error())
					return false
				}

				return true
			case key == "group":
				// parse the option group recursively
				optGrp := command{
					name:   value,
					dest:   subdest,
					parent: cmd,
					help:   field.Tag.Get("help"),
				}
				cmd.groups = append(cmd.groups, &optGrp)

				// decide on a name for the group
				if optGrp.name == "" {
					optGrp.name = strings.Title(field.Name)
				}

				err := optGrp.parseFieldsFromStructOrStructPointer(field.Type, true)
				if err != nil {
					errs = append(errs, err.Error())
					return false
				}

				return false
			default:
				errs = append(errs, fmt.Sprintf("unrecognized tag '%s' on field %s", key, tag))
				return false
			}
		}

		placeholder, hasPlaceholder := field.Tag.Lookup("placeholder")
		if hasPlaceholder {
			spec.placeholder = placeholder
		} else if spec.long != "" {
			spec.placeholder = strings.ToUpper(spec.long)
		} else {
			spec.placeholder = strings.ToUpper(spec.field.Name)
		}

		// check whether this field is supported. It's good to do this here rather than
		// wait until ParseValue because it means that a program with invalid argument
		// fields will always fail regardless of whether the arguments it received
		// exercised those fields.
		var err error
		spec.cardinality, err = cardinalityOf(field.Type)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s.%s: %s fields are not supported",
				t.Name(), field.Name, field.Type.String()))
			return false
		}

		defaultString, hasDefault := field.Tag.Lookup("default")
		if hasDefault {
			// we do not support default values for maps and slices
			if spec.cardinality == multiple {
				errs = append(errs, fmt.Sprintf("%s.%s: default values are not supported for slice or map fields",
					t.Name(), field.Name))
				return false
			}

			// a required field cannot also have a default value
			if spec.required {
				errs = append(errs, fmt.Sprintf("%s.%s: 'required' cannot be used when a default value is specified",
					t.Name(), field.Name))
				return false
			}

			// parse the default value
			spec.defaultString = defaultString
			if field.Type.Kind() == reflect.Ptr {
				// here we have a field of type *T and we create a new T, no need to dereference
				// in order for the value to be settable
				spec.defaultValue = reflect.New(field.Type.Elem())
			} else {
				// here we have a field of type T and we create a new T and then dereference it
				// so that the resulting value is settable
				spec.defaultValue = reflect.New(field.Type).Elem()
			}
			err := scalar.ParseValue(spec.defaultValue, defaultString)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s.%s: error processing default value: %v", t.Name(), field.Name, err))
				return false
			}
		}

		// add the spec to the list of specs
		cmd.options = append(cmd.options, &spec)

		// if this was an embedded field then we already returned true up above
		return false
	})

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	// check that we don't have both positionals and subcommands
	var hasPositional bool
	for _, spec := range cmd.options {
		if spec.positional {
			hasPositional = true
		}
	}
	if hasPositional && len(cmd.subcommands) > 0 {
		return fmt.Errorf("%s cannot have both subcommands and positional arguments",
			cmd.dest)

	}

	return nil
}

// Parse processes the given command line option, storing the results in the field
// of the structs from which NewParser was constructed
func (p *Parser) Parse(args []string) error {
	err := p.process(args)
	if err != nil {
		// If -h or --help were specified then make sure help text supercedes other errors
		for _, arg := range args {
			if arg == "-h" || arg == "--help" {
				return ErrHelp
			}
			if arg == "--" {
				break
			}
		}
	}
	return err
}

func (p *Parser) MustParse(args []string) {
	err := p.Parse(args)
	switch {
	case err == ErrHelp:
		p.writeHelpForSubcommand(p.config.Out, p.lastCmd)
		p.config.Exit(0)
	case err == ErrVersion:
		fmt.Fprintln(p.config.Out, p.version)
		p.config.Exit(0)
	case err != nil:
		p.failWithSubcommand(err.Error(), p.lastCmd)
	}
}

// process environment vars for the given arguments
func (p *Parser) captureEnvVars(specs []*spec, wasPresent map[*spec]bool) error {
	for _, spec := range specs {
		if spec.env == "" {
			continue
		}

		value, found := os.LookupEnv(spec.env)
		if !found {
			continue
		}

		if spec.cardinality == multiple {
			// expect a CSV string in an environment
			// variable in the case of multiple values
			var values []string
			var err error
			if len(strings.TrimSpace(value)) > 0 {
				values, err = csv.NewReader(strings.NewReader(value)).Read()
				if err != nil {
					return fmt.Errorf(
						"error reading a CSV string from environment variable %s with multiple values: %v",
						spec.env,
						err,
					)
				}
			}
			if err = setSliceOrMap(p.val(spec.dest), values, !spec.separate); err != nil {
				return fmt.Errorf(
					"error processing environment variable %s with multiple values: %v",
					spec.env,
					err,
				)
			}
		} else {
			if err := scalar.ParseValue(p.val(spec.dest), value); err != nil {
				return fmt.Errorf("error processing environment variable %s: %v", spec.env, err)
			}
		}
		wasPresent[spec] = true
	}

	return nil
}

// process goes through arguments one-by-one, parses them, and assigns the result to
// the underlying struct field
func (p *Parser) process(args []string) error {
	// track the options we have seen
	wasPresent := make(map[*spec]bool)

	// union of specs for the chain of subcommands encountered so far
	curCmd := p.cmd
	p.lastCmd = curCmd

	// make a copy of the specs because we will add to this list each time we expand a subcommand
	specs := curCmd.specs()

	// deal with environment vars
	if !p.config.IgnoreEnv {
		err := p.captureEnvVars(specs, wasPresent)
		if err != nil {
			return err
		}
	}

	// process each string from the command line
	var allpositional bool
	var positionals []string

	// must use explicit for loop, not range, because we manipulate i inside the loop
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			allpositional = true
			continue
		}

		if !isFlag(arg) || allpositional {
			// each subcommand can have either subcommands or positionals, but not both
			if len(curCmd.subcommands) == 0 {
				positionals = append(positionals, arg)
				continue
			}

			// if we have a subcommand then make sure it is valid for the current context
			subcmd := findSubcommand(curCmd.subcommands, arg)
			if subcmd == nil {
				return fmt.Errorf("invalid subcommand: %s", arg)
			}

			// ensure the command struct exists (is not a nil pointer)
			p.val(subcmd.dest)

			// add the new options to the set of allowed options
			if p.config.StrictSubcommands {
				specs = make([]*spec, len(subcmd.specs()))
				copy(specs, subcmd.specs())
			} else {
				specs = append(specs, subcmd.specs()...)
			}

			// capture environment vars for these new options
			if !p.config.IgnoreEnv {
				err := p.captureEnvVars(subcmd.specs(), wasPresent)
				if err != nil {
					return err
				}
			}

			curCmd = subcmd
			p.lastCmd = curCmd
			continue
		}

		// check for special --help and --version flags
		switch arg {
		case "-h", "--help":
			return ErrHelp
		case "--version":
			return ErrVersion
		}

		// check for an equals sign, as in "--foo=bar"
		var value string
		opt := strings.TrimLeft(arg, "-")
		if pos := strings.Index(opt, "="); pos != -1 {
			value = opt[pos+1:]
			opt = opt[:pos]
		}

		// lookup the spec for this option (note that the "specs" slice changes as
		// we expand subcommands so it is better not to use a map)
		spec := findOption(specs, opt)
		if spec == nil {
			return fmt.Errorf("unknown argument %s", arg)
		}
		wasPresent[spec] = true

		// deal with the case of multiple values
		if spec.cardinality == multiple {
			var values []string
			if value == "" {
				for i+1 < len(args) && !isFlag(args[i+1]) && args[i+1] != "--" {
					values = append(values, args[i+1])
					i++
					if spec.separate {
						break
					}
				}
			} else {
				values = append(values, value)
			}
			err := setSliceOrMap(p.val(spec.dest), values, !spec.separate)
			if err != nil {
				return fmt.Errorf("error processing %s: %v", arg, err)
			}
			continue
		}

		// if it's a flag and it has no value then set the value to true
		// use boolean because this takes account of TextUnmarshaler
		if spec.cardinality == zero && value == "" {
			value = "true"
		}

		// if we have something like "--foo" then the value is the next argument
		if value == "" {
			if i+1 == len(args) {
				return fmt.Errorf("missing value for %s", arg)
			}
			if !nextIsNumeric(spec.field.Type, args[i+1]) && isFlag(args[i+1]) {
				return fmt.Errorf("missing value for %s", arg)
			}
			value = args[i+1]
			i++
		}

		err := scalar.ParseValue(p.val(spec.dest), value)
		if err != nil {
			return fmt.Errorf("error processing %s: %v", arg, err)
		}
	}

	// process positionals
	for _, spec := range specs {
		if !spec.positional {
			continue
		}
		if len(positionals) == 0 {
			break
		}
		wasPresent[spec] = true
		if spec.cardinality == multiple {
			err := setSliceOrMap(p.val(spec.dest), positionals, true)
			if err != nil {
				return fmt.Errorf("error processing %s: %v", spec.field.Name, err)
			}
			positionals = nil
		} else {
			err := scalar.ParseValue(p.val(spec.dest), positionals[0])
			if err != nil {
				return fmt.Errorf("error processing %s: %v", spec.field.Name, err)
			}
			positionals = positionals[1:]
		}
	}
	if len(positionals) > 0 {
		return fmt.Errorf("too many positional arguments at '%s'", positionals[0])
	}

	// fill in defaults and check that all the required args were provided
	for _, spec := range specs {
		if wasPresent[spec] {
			continue
		}

		name := strings.ToLower(spec.field.Name)
		if spec.long != "" && !spec.positional {
			name = "--" + spec.long
		}

		if spec.required {
			msg := fmt.Sprintf("%s is required", name)
			if spec.env != "" {
				msg += " (or environment variable " + spec.env + ")"
			}
			return errors.New(msg)
		}

		if spec.defaultValue.IsValid() && !p.config.IgnoreDefault {
			// One issue here is that if the user now modifies the value then
			// the default value stored in the spec will be corrupted. There
			// is no general way to "deep-copy" values in Go, and we still
			// support the old-style method for specifying defaults as
			// Go values assigned directly to the struct field, so we are stuck.
			p.val(spec.dest).Set(spec.defaultValue)
		}
	}

	return nil
}

func nextIsNumeric(t reflect.Type, s string) bool {
	switch t.Kind() {
	case reflect.Ptr:
		return nextIsNumeric(t.Elem(), s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		v := reflect.New(t)
		err := scalar.ParseValue(v, s)
		return err == nil
	default:
		return false
	}
}

// isFlag returns true if a token is a flag such as "-v" or "--user" but not "-" or "--"
func isFlag(s string) bool {
	return strings.HasPrefix(s, "-") && strings.TrimLeft(s, "-") != ""
}

// defaultVal returns the string representation of the value at dest if it is
// reachable without traversing nil pointers, but only if it does not represent
// the default value for the type.
func (p *Parser) defaultVal(dest path) (string, reflect.Value, error) {
	v := p.roots[dest.root]
	for _, field := range dest.fields {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return "", v, nil
			}
			v = v.Elem()
		}

		v = v.FieldByIndex(field.Index)
	}

	if !v.IsValid() || isZero(v) {
		return "", v, nil
	}

	if defaultVal, ok := v.Interface().(encoding.TextMarshaler); ok {
		str, err := defaultVal.MarshalText()
		if err != nil {
			return "", v, fmt.Errorf("%v: error marshaling default value to string: %w", dest, err)
		}
		return string(str), v, nil
	}

	return fmt.Sprintf("%v", v), v, nil
}

// val returns a reflect.Value corresponding to the current value for the
// given path initiating nil pointers in the path
func (p *Parser) val(dest path) reflect.Value {
	v := p.roots[dest.root]
	for _, field := range dest.fields {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}

		v = v.FieldByIndex(field.Index)
	}

	// Don't return a nil-pointer
	if v.Kind() == reflect.Ptr && v.IsNil() {
		v.Set(reflect.New(v.Type().Elem()))
	}

	return v
}

// findOption finds an option from its name, or returns null if no spec is found
func findOption(specs []*spec, name string) *spec {
	for _, spec := range specs {
		if spec.positional {
			continue
		}
		if spec.long == name || spec.short == name {
			return spec
		}
	}
	return nil
}

// findSubcommand finds a subcommand using its name, or returns null if no subcommand is found
func findSubcommand(cmds []*command, name string) *command {
	for _, cmd := range cmds {
		if cmd.name == name {
			return cmd
		}
	}
	return nil
}
