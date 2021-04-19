package arg

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertCardinality(t *testing.T, typ reflect.Type, expected cardinality) {
	actual, err := cardinalityOf(typ)
	assert.Equal(t, expected, actual, "expected %v to have cardinality %v but got %v", typ, expected, actual)
	if expected == unsupported {
		assert.Error(t, err)
	}
}

func TestCardinalityOf(t *testing.T) {
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

	assertCardinality(t, reflect.TypeOf(b), zero)
	assertCardinality(t, reflect.TypeOf(i), one)
	assertCardinality(t, reflect.TypeOf(s), one)
	assertCardinality(t, reflect.TypeOf(f), one)

	assertCardinality(t, reflect.TypeOf(&b), zero)
	assertCardinality(t, reflect.TypeOf(&s), one)
	assertCardinality(t, reflect.TypeOf(&i), one)
	assertCardinality(t, reflect.TypeOf(&f), one)

	assertCardinality(t, reflect.TypeOf(bs), multiple)
	assertCardinality(t, reflect.TypeOf(is), multiple)

	assertCardinality(t, reflect.TypeOf(&bs), multiple)
	assertCardinality(t, reflect.TypeOf(&is), multiple)

	assertCardinality(t, reflect.TypeOf(m), multiple)
	assertCardinality(t, reflect.TypeOf(&m), multiple)

	assertCardinality(t, reflect.TypeOf(unsupported1), unsupported)
	assertCardinality(t, reflect.TypeOf(&unsupported1), unsupported)
	assertCardinality(t, reflect.TypeOf(unsupported2), unsupported)
	assertCardinality(t, reflect.TypeOf(&unsupported2), unsupported)
	assertCardinality(t, reflect.TypeOf(unsupported3), unsupported)
	assertCardinality(t, reflect.TypeOf(&unsupported3), unsupported)
}

type implementsTextUnmarshaler struct{}

func (*implementsTextUnmarshaler) UnmarshalText(text []byte) error {
	return nil
}

func TestCardinalityTextUnmarshaler(t *testing.T) {
	var x implementsTextUnmarshaler
	var s []implementsTextUnmarshaler
	var m []implementsTextUnmarshaler
	assertCardinality(t, reflect.TypeOf(x), one)
	assertCardinality(t, reflect.TypeOf(&x), one)
	assertCardinality(t, reflect.TypeOf(s), multiple)
	assertCardinality(t, reflect.TypeOf(&s), multiple)
	assertCardinality(t, reflect.TypeOf(m), multiple)
	assertCardinality(t, reflect.TypeOf(&m), multiple)
}

func TestIsExported(t *testing.T) {
	assert.True(t, isExported("Exported"))
	assert.False(t, isExported("notExported"))
	assert.False(t, isExported(""))
	assert.False(t, isExported(string([]byte{255})))
}
