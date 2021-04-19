package arg

import (
	"encoding"
	"fmt"
	"reflect"
	"unicode"
	"unicode/utf8"

	scalar "github.com/alexflint/go-scalar"
)

var textUnmarshalerType = reflect.TypeOf([]encoding.TextUnmarshaler{}).Elem()

// kind is used to track the various kinds of options:
//  - regular is an ordinary option that will be parsed from a single token
//  - binary is an option that will be true if present but does not expect an explicit value
//  - sequence is an option that accepts multiple values and will end up in a slice
//  - mapping is an option that acccepts multiple key=value strings and will end up in a map
type kind int

const (
	regular kind = iota
	binary
	sequence
	mapping
	unsupported
)

func (k kind) String() string {
	switch k {
	case regular:
		return "regular"
	case binary:
		return "binary"
	case sequence:
		return "sequence"
	case mapping:
		return "mapping"
	case unsupported:
		return "unsupported"
	default:
		return fmt.Sprintf("unknown(%d)", int(k))
	}
}

// kindOf returns true if the type can be parsed from a string
func kindOf(t reflect.Type) (kind, error) {
	if scalar.CanParse(t) {
		if isBoolean(t) {
			return binary, nil
		} else {
			return regular, nil
		}
	}

	// look inside pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// look inside slice and map types
	switch t.Kind() {
	case reflect.Slice:
		if !scalar.CanParse(t.Elem()) {
			return unsupported, fmt.Errorf("cannot parse into %v because we cannot parse into %v", t, t.Elem())
		}
		return sequence, nil
	case reflect.Map:
		if !scalar.CanParse(t.Key()) {
			return unsupported, fmt.Errorf("cannot parse into %v because we cannot parse into the key type %v", t, t.Elem())
		}
		if !scalar.CanParse(t.Elem()) {
			return unsupported, fmt.Errorf("cannot parse into %v because we cannot parse into the value type %v", t, t.Elem())
		}
		return mapping, nil
	default:
		return unsupported, fmt.Errorf("cannot parse into %v", t)
	}
}

// isBoolean returns true if the type can be parsed from a single string
func isBoolean(t reflect.Type) bool {
	switch {
	case t.Implements(textUnmarshalerType):
		return false
	case t.Kind() == reflect.Bool:
		return true
	case t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Bool:
		return true
	default:
		return false
	}
}

// isExported returns true if the struct field name is exported
func isExported(field string) bool {
	r, _ := utf8.DecodeRuneInString(field) // returns RuneError for empty string or invalid UTF8
	return unicode.IsLetter(r) && unicode.IsUpper(r)
}
