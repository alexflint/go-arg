package arguments

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// MustParse processes command line arguments and exits upon failure.
func MustParse(dest interface{}) {
	err := Parse(dest)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Parse processes command line arguments and stores the result in args.
func Parse(dest interface{}) error {
	return ParseFrom(dest, os.Args)
}

// ParseFrom processes command line arguments and stores the result in args.
func ParseFrom(dest interface{}, args []string) error {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("%s is not a pointer type", v.Type().Name()))
	}
	v = v.Elem()

	// Parse the spec
	spec, err := extractSpec(v.Type())
	if err != nil {
		return err
	}

	// Process args
	err = processArgs(v, spec, args)
	if err != nil {
		return err
	}

	// Validate
	return validate(spec)
}

// spec represents information about an argument extracted from struct tags
type spec struct {
	field      reflect.StructField
	index      int
	long       string
	short      string
	multiple   bool
	required   bool
	positional bool
	help       string
	wasPresent bool
}

// extractSpec gets specifications for each argument from the tags in a struct
func extractSpec(t reflect.Type) ([]*spec, error) {
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("%s is not a struct pointer", t.Name()))
	}

	var specs []*spec
	for i := 0; i < t.NumField(); i++ {
		// Check for the ignore switch in the tag
		field := t.Field(i)
		tag := field.Tag.Get("arg")
		if tag == "-" {
			continue
		}

		spec := spec{
			long:  strings.ToLower(field.Name),
			field: field,
			index: i,
		}

		// Get the scalar type for this field
		scalarType := field.Type
		if scalarType.Kind() == reflect.Slice {
			spec.multiple = true
			scalarType = scalarType.Elem()
			if scalarType.Kind() == reflect.Ptr {
				scalarType = scalarType.Elem()
			}
		}

		// Check for unsupported types
		switch scalarType.Kind() {
		case reflect.Array, reflect.Chan, reflect.Func, reflect.Interface,
			reflect.Map, reflect.Ptr, reflect.Struct,
			reflect.Complex64, reflect.Complex128:
			return nil, fmt.Errorf("%s.%s: %s fields are not supported", t.Name(), field.Name, scalarType.Kind())
		}

		// Look at the tag
		if tag != "" {
			for _, key := range strings.Split(tag, ",") {
				var value string
				if pos := strings.Index(key, ":"); pos != -1 {
					value = key[pos+1:]
					key = key[:pos]
				}

				switch {
				case strings.HasPrefix(key, "--"):
					spec.long = key[2:]
				case strings.HasPrefix(key, "-"):
					if len(key) != 2 {
						return nil, fmt.Errorf("%s.%s: short arguments must be one character only", t.Name(), field.Name)
					}
					spec.short = key[1:]
				case key == "required":
					spec.required = true
				case key == "positional":
					spec.positional = true
				case key == "help":
					spec.help = value
				default:
					return nil, fmt.Errorf("unrecognized tag '%s' on field %s", key, tag)
				}
			}
		}
		specs = append(specs, &spec)
	}
	return specs, nil
}

// processArgs processes arguments using a pre-constructed spec
func processArgs(dest reflect.Value, specs []*spec, args []string) error {
	// construct a map from arg name to spec
	specByName := make(map[string]*spec)
	for _, spec := range specs {
		if spec.long != "" {
			specByName[spec.long] = spec
		}
		if spec.short != "" {
			specByName[spec.short] = spec
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

		if !strings.HasPrefix(arg, "-") || allpositional {
			positionals = append(positionals, arg)
			continue
		}

		// check for an equals sign, as in "--foo=bar"
		var value string
		opt := strings.TrimLeft(arg, "-")
		if pos := strings.Index(opt, "="); pos != -1 {
			value = opt[pos+1:]
			opt = opt[:pos]
		}

		// lookup the spec for this option
		spec, ok := specByName[opt]
		if !ok {
			return fmt.Errorf("unknown argument %s", arg)
		}
		spec.wasPresent = true

		// deal with the case of multiple values
		if spec.multiple {
			var values []string
			if value == "" {
				for i++; i < len(args) && !strings.HasPrefix(args[i], "-"); i++ {
					values = append(values, args[i])
				}
			} else {
				values = append(values, value)
			}
			setSlice(dest, spec, values)
			continue
		}

		// if it's a flag and it has no value then set the value to true
		if spec.field.Type.Kind() == reflect.Bool && value == "" {
			value = "true"
		}

		// if we have something like "--foo" then the value is the next argument
		if value == "" {
			if i+1 == len(args) || strings.HasPrefix(args[i+1], "-") {
				return fmt.Errorf("missing value for %s", arg)
			}
			value = args[i+1]
			i++
		}

		err := setScalar(dest.Field(spec.index), value)
		if err != nil {
			return fmt.Errorf("error processing %s: %v", arg, err)
		}
	}
	return nil
}

// validate an argument spec after arguments have been parse
func validate(spec []*spec) error {
	for _, arg := range spec {
		if arg.required && !arg.wasPresent {
			return fmt.Errorf("--%s is required", strings.ToLower(arg.field.Name))
		}
	}
	return nil
}

// parse a value as the apropriate type and store it in the struct
func setSlice(dest reflect.Value, spec *spec, values []string) error {
	// TODO
	return nil
}

// set a value from a string
func setScalar(v reflect.Value, s string) error {
	if !v.CanSet() {
		return fmt.Errorf("field is not writable")
	}

	switch v.Kind() {
	case reflect.String:
		v.Set(reflect.ValueOf(s))
	case reflect.Bool:
		x, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(x))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		x, err := strconv.ParseInt(s, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(x).Convert(v.Type()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		x, err := strconv.ParseUint(s, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(x).Convert(v.Type()))
	case reflect.Float32, reflect.Float64:
		x, err := strconv.ParseFloat(s, v.Type().Bits())
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(x).Convert(v.Type()))
	default:
		return fmt.Errorf("not a scalar type: %s", v.Kind())
	}
	return nil
}
