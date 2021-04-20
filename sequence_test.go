package arg

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetSliceWithoutClearing(t *testing.T) {
	xs := []int{10}
	entries := []string{"1", "2", "3"}
	err := setSlice(reflect.ValueOf(&xs).Elem(), entries, false)
	require.NoError(t, err)
	assert.Equal(t, []int{10, 1, 2, 3}, xs)
}

func TestSetSliceAfterClearing(t *testing.T) {
	xs := []int{100}
	entries := []string{"1", "2", "3"}
	err := setSlice(reflect.ValueOf(&xs).Elem(), entries, true)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, xs)
}

func TestSetSliceInvalid(t *testing.T) {
	xs := []int{100}
	entries := []string{"invalid"}
	err := setSlice(reflect.ValueOf(&xs).Elem(), entries, true)
	assert.Error(t, err)
}

func TestSetSlicePtr(t *testing.T) {
	var xs []*int
	entries := []string{"1", "2", "3"}
	err := setSlice(reflect.ValueOf(&xs).Elem(), entries, true)
	require.NoError(t, err)
	require.Len(t, xs, 3)
	assert.Equal(t, 1, *xs[0])
	assert.Equal(t, 2, *xs[1])
	assert.Equal(t, 3, *xs[2])
}

func TestSetSliceTextUnmarshaller(t *testing.T) {
	// textUnmarshaler is a struct that captures the length of the string passed to it
	var xs []*textUnmarshaler
	entries := []string{"a", "aa", "aaa"}
	err := setSlice(reflect.ValueOf(&xs).Elem(), entries, true)
	require.NoError(t, err)
	require.Len(t, xs, 3)
	assert.Equal(t, 1, xs[0].val)
	assert.Equal(t, 2, xs[1].val)
	assert.Equal(t, 3, xs[2].val)
}

func TestSetMapWithoutClearing(t *testing.T) {
	m := map[string]int{"foo": 10}
	entries := []string{"a=1", "b=2"}
	err := setMap(reflect.ValueOf(&m).Elem(), entries, false)
	require.NoError(t, err)
	require.Len(t, m, 3)
	assert.Equal(t, 1, m["a"])
	assert.Equal(t, 2, m["b"])
	assert.Equal(t, 10, m["foo"])
}

func TestSetMapAfterClearing(t *testing.T) {
	m := map[string]int{"foo": 10}
	entries := []string{"a=1", "b=2"}
	err := setMap(reflect.ValueOf(&m).Elem(), entries, true)
	require.NoError(t, err)
	require.Len(t, m, 2)
	assert.Equal(t, 1, m["a"])
	assert.Equal(t, 2, m["b"])
}

func TestSetMapWithKeyPointer(t *testing.T) {
	// textUnmarshaler is a struct that captures the length of the string passed to it
	var m map[*string]int
	entries := []string{"abc=123"}
	err := setMap(reflect.ValueOf(&m).Elem(), entries, true)
	require.NoError(t, err)
	require.Len(t, m, 1)
}

func TestSetMapWithValuePointer(t *testing.T) {
	// textUnmarshaler is a struct that captures the length of the string passed to it
	var m map[string]*int
	entries := []string{"abc=123"}
	err := setMap(reflect.ValueOf(&m).Elem(), entries, true)
	require.NoError(t, err)
	require.Len(t, m, 1)
	assert.Equal(t, 123, *m["abc"])
}

func TestSetMapTextUnmarshaller(t *testing.T) {
	// textUnmarshaler is a struct that captures the length of the string passed to it
	var m map[textUnmarshaler]*textUnmarshaler
	entries := []string{"a=123", "aa=12", "aaa=1"}
	err := setMap(reflect.ValueOf(&m).Elem(), entries, true)
	require.NoError(t, err)
	require.Len(t, m, 3)
	assert.Equal(t, &textUnmarshaler{3}, m[textUnmarshaler{1}])
	assert.Equal(t, &textUnmarshaler{2}, m[textUnmarshaler{2}])
	assert.Equal(t, &textUnmarshaler{1}, m[textUnmarshaler{3}])
}

func TestSetMapInvalidKey(t *testing.T) {
	var m map[int]int
	entries := []string{"invalid=123"}
	err := setMap(reflect.ValueOf(&m).Elem(), entries, true)
	assert.Error(t, err)
}

func TestSetMapInvalidValue(t *testing.T) {
	var m map[int]int
	entries := []string{"123=invalid"}
	err := setMap(reflect.ValueOf(&m).Elem(), entries, true)
	assert.Error(t, err)
}

func TestSetMapMalformed(t *testing.T) {
	// textUnmarshaler is a struct that captures the length of the string passed to it
	var m map[string]string
	entries := []string{"missing_equals_sign"}
	err := setMap(reflect.ValueOf(&m).Elem(), entries, true)
	assert.Error(t, err)
}

func TestSetSliceOrMapErrors(t *testing.T) {
	var err error
	var dest reflect.Value

	// converting a slice to a reflect.Value in this way will make it read only
	var cannotSet []int
	dest = reflect.ValueOf(cannotSet)
	err = setSliceOrMap(dest, nil, false)
	assert.Error(t, err)

	// check what happens when we pass in something that is not a slice or a map
	var notSliceOrMap string
	dest = reflect.ValueOf(&notSliceOrMap).Elem()
	err = setSliceOrMap(dest, nil, false)
	assert.Error(t, err)

	// check what happens when we pass in a pointer to something that is not a slice or a map
	var stringPtr *string
	dest = reflect.ValueOf(&stringPtr).Elem()
	err = setSliceOrMap(dest, nil, false)
	assert.Error(t, err)
}
