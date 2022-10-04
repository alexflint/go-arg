package arg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// Argument represents a command line argument
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

// Parser represents a set of command line options with destination values
type Parser struct {
	cmd      *Command      // the top-level command
	root     reflect.Value // destination struct to fill will values
	version  string        // version from the argument struct
	prologue string        // prologue for help text (from the argument struct)
	epilogue string        // epilogue for help text (from the argument struct)

	// the following fields are updated during processing of command line arguments
	leaf       *Command           // the subcommand we processed last
	accessible []*Argument        // concatenation of the leaf subcommand's arguments plus all ancestors' arguments
	seen       map[*Argument]bool // the arguments we encountered while processing command line arguments
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
	// copy the args for the root command into "accessible", which will
	// grow each time we open up a subcommand
	p.accessible = make([]*Argument, len(p.cmd.args))
	copy(p.accessible, p.cmd.args)

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

		// create a new destination path for this field
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
