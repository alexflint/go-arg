package arg

import (
	"encoding"
	"reflect"
	"unicode"
	"unicode/utf8"

	scalar "github.com/alexflint/go-scalar"
)

var textUnmarshalerType = reflect.TypeOf([]encoding.TextUnmarshaler{}).Elem()

// canParse returns true if the type can be parsed from a string
func canParse(t reflect.Type) (parseable, boolean, multiple bool) {
	parseable = scalar.CanParse(t)
	boolean = isBoolean(t)
	if parseable {
		return
	}

	// Look inside pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// Look inside slice types
	if t.Kind() == reflect.Slice {
		multiple = true
		t = t.Elem()
	}

	parseable = scalar.CanParse(t)
	boolean = isBoolean(t)
	if parseable {
		return
	}

	// Look inside pointer types (again, in case of []*Type)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	parseable = scalar.CanParse(t)
	boolean = isBoolean(t)
	if parseable {
		return
	}

	return false, false, false
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
