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
	var unsupported4 map[struct{}]string

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
	assertCardinality(t, reflect.TypeOf(unsupported4), unsupported)
	assertCardinality(t, reflect.TypeOf(&unsupported4), unsupported)
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

func TestCardinalityString(t *testing.T) {
	assert.Equal(t, "zero", zero.String())
	assert.Equal(t, "one", one.String())
	assert.Equal(t, "multiple", multiple.String())
	assert.Equal(t, "unsupported", unsupported.String())
	assert.Equal(t, "unknown(42)", cardinality(42).String())
}

func TestIsZero(t *testing.T) {
	var zero int
	var notZero = 3
	var nilSlice []int
	var nonNilSlice = []int{1, 2, 3}
	var nilMap map[string]string
	var nonNilMap = map[string]string{"foo": "bar"}
	var uncomparable = func() {}

	assert.True(t, isZero(reflect.ValueOf(zero)))
	assert.False(t, isZero(reflect.ValueOf(notZero)))

	assert.True(t, isZero(reflect.ValueOf(nilSlice)))
	assert.False(t, isZero(reflect.ValueOf(nonNilSlice)))

	assert.True(t, isZero(reflect.ValueOf(nilMap)))
	assert.False(t, isZero(reflect.ValueOf(nonNilMap)))

	assert.False(t, isZero(reflect.ValueOf(uncomparable)))
}
