package arg

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertKind(t *testing.T, typ reflect.Type, expected kind) {
	actual, err := kindOf(typ)
	assert.Equal(t, expected, actual, "expected %v to have kind %v but got %v", typ, expected, actual)
	if expected == unsupported {
		assert.Error(t, err)
	}
}

func TestKindOf(t *testing.T) {
	var b bool
	var i int
	var s string
	var f float64
	var bs []bool
	var is []int
	var m map[string]int
	var unsupported1 struct{}
	var unsupported2 []struct{}
	var unsupported3 map[string]struct{}

	assertKind(t, reflect.TypeOf(b), binary)
	assertKind(t, reflect.TypeOf(i), regular)
	assertKind(t, reflect.TypeOf(s), regular)
	assertKind(t, reflect.TypeOf(f), regular)

	assertKind(t, reflect.TypeOf(&b), binary)
	assertKind(t, reflect.TypeOf(&s), regular)
	assertKind(t, reflect.TypeOf(&i), regular)
	assertKind(t, reflect.TypeOf(&f), regular)

	assertKind(t, reflect.TypeOf(bs), sequence)
	assertKind(t, reflect.TypeOf(is), sequence)

	assertKind(t, reflect.TypeOf(&bs), sequence)
	assertKind(t, reflect.TypeOf(&is), sequence)

	assertKind(t, reflect.TypeOf(m), mapping)
	assertKind(t, reflect.TypeOf(&m), mapping)

	assertKind(t, reflect.TypeOf(unsupported1), unsupported)
	assertKind(t, reflect.TypeOf(&unsupported1), unsupported)
	assertKind(t, reflect.TypeOf(unsupported2), unsupported)
	assertKind(t, reflect.TypeOf(&unsupported2), unsupported)
	assertKind(t, reflect.TypeOf(unsupported3), unsupported)
	assertKind(t, reflect.TypeOf(&unsupported3), unsupported)
}

type implementsTextUnmarshaler struct{}

func (*implementsTextUnmarshaler) UnmarshalText(text []byte) error {
	return nil
}

func TestCanParseTextUnmarshaler(t *testing.T) {
	var x implementsTextUnmarshaler
	var s []implementsTextUnmarshaler
	var m []implementsTextUnmarshaler
	assertKind(t, reflect.TypeOf(x), regular)
	assertKind(t, reflect.TypeOf(&x), regular)
	assertKind(t, reflect.TypeOf(s), sequence)
	assertKind(t, reflect.TypeOf(&s), sequence)
	assertKind(t, reflect.TypeOf(m), mapping)
	assertKind(t, reflect.TypeOf(&m), mapping)
}

func TestIsExported(t *testing.T) {
	assert.True(t, isExported("Exported"))
	assert.False(t, isExported("notExported"))
	assert.False(t, isExported(""))
	assert.False(t, isExported(string([]byte{255})))
}
