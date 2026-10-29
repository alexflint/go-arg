package arg

import (
	"fmt"
	"reflect"
)

// MarshalDefaultValueError is returned when marshaling a default value to string fails.
type MarshalDefaultValueError struct {
	Dest any
	Err  error
}

func (e *MarshalDefaultValueError) Error() string {
	return fmt.Sprintf("%v: error marshaling default value to string: %v", e.Dest, e.Err)
}

func (e *MarshalDefaultValueError) Unwrap() error { return e.Err }

// SubcommandTypeError is returned when a subcommand is not a pointer to a struct.
type SubcommandTypeError struct {
	Dest    any
	Kind    any
	Pointer bool
}

func (e *SubcommandTypeError) Error() string {
	if e.Pointer {
		return fmt.Sprintf("subcommands must be pointers to structs but %v is a pointer to %v", e.Dest, e.Kind)
	}
	return fmt.Sprintf("subcommands must be pointers to structs but %v is a %v", e.Dest, e.Kind)
}

// SubcommandsAndPositionalsError is returned when both subcommands and positional arguments are present.
type SubcommandsAndPositionalsError struct {
	Dest any
}

func (e *SubcommandsAndPositionalsError) Error() string {
	return fmt.Sprintf("%v cannot have both subcommands and positional arguments", e.Dest)
}

// EnvVarParseError is returned when parsing an environment variable fails.
type EnvVarParseError struct {
	Env      string
	Err      error
	Multiple bool
}

func (e *EnvVarParseError) Error() string {
	if e.Multiple {
		return fmt.Sprintf("error processing environment variable %s with multiple values: %v", e.Env, e.Err)
	}
	return fmt.Sprintf("error processing environment variable %s: %v", e.Env, e.Err)
}
func (e *EnvVarParseError) Unwrap() error { return e.Err }

// InvalidSubcommandError is returned when an invalid subcommand is encountered.
type InvalidSubcommandError struct {
	Arg string
}

func (e *InvalidSubcommandError) Error() string {
	return fmt.Sprintf("invalid subcommand: %s", e.Arg)
}

// UnknownArgumentError is returned when an unknown argument is encountered.
type UnknownArgumentError struct {
	Arg string
}

func (e *UnknownArgumentError) Error() string {
	return fmt.Sprintf("unknown argument %s", e.Arg)
}

// MissingValueError is returned when a required value is missing for an argument.
type MissingValueError struct {
	Arg string
}

func (e *MissingValueError) Error() string {
	return fmt.Sprintf("missing value for %s", e.Arg)
}

// ArgumentProcessingError is returned when processing an argument fails.
type ArgumentProcessingError struct {
	Arg string
	Err error
}

func (e *ArgumentProcessingError) Error() string {
	return fmt.Sprintf("error processing %s: %v", e.Arg, e.Err)
}
func (e *ArgumentProcessingError) Unwrap() error { return e.Err }

// TooManyPositionalsError is returned when there are too many positional arguments.
type TooManyPositionalsError struct {
	Arg string
}

func (e *TooManyPositionalsError) Error() string {
	return fmt.Sprintf("too many positional arguments at '%s'", e.Arg)
}

// FieldNotWritableError is returned when a struct field cannot be set via reflection.
type FieldNotWritableError struct{}

func (e *FieldNotWritableError) Error() string {
	return "field is not writable"
}

// SetSliceOrMapTypeError is returned when setSliceOrMap is called on an unsupported type.
type SetSliceOrMapTypeError struct {
	Type reflect.Type
}

func (e *SetSliceOrMapTypeError) Error() string {
	return fmt.Sprintf("setSliceOrMap cannot insert values into a %v", e.Type)
}

// MapParseFormatError is returned when a string cannot be parsed into a map entry.
type MapParseFormatError struct {
	Input string
}

func (e *MapParseFormatError) Error() string {
	return fmt.Sprintf("cannot parse %q into a map, expected format key=value", e.Input)
}

// CardinalityTypeError is returned when a type is not supported for parsing.
type CardinalityTypeError struct {
	Type   reflect.Type
	Reason string
}

func (e *CardinalityTypeError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("cannot parse into %v because %s", e.Type, e.Reason)
	}
	return fmt.Sprintf("cannot parse into %v", e.Type)
}

// SubcommandNotFoundError is returned when a subcommand is not found.
type SubcommandNotFoundError struct {
	Name    string
	CmdName string
}

func (e *SubcommandNotFoundError) Error() string {
	return fmt.Sprintf("%q is not a subcommand of %s", e.Name, e.CmdName)
}
