package arg

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

// set a value from a string
func setScalar(v reflect.Value, s string) error {
	if !v.CanSet() {
		return fmt.Errorf("field is not exported")
	}

	// If we have a nil pointer then allocate a new object
	if v.Kind() == reflect.Ptr && v.IsNil() {
		v.Set(reflect.New(v.Type().Elem()))
	}

	// Get the object as an interface
	scalar := v.Interface()

	// If it implements encoding.TextUnmarshaler then use that
	if scalar, ok := scalar.(encoding.TextUnmarshaler); ok {
		return scalar.UnmarshalText([]byte(s))
	}

	// If we have a pointer then dereference it
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Switch on concrete type
	switch scalar.(type) {
	case time.Duration:
		x, err := time.ParseDuration(s)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(x))
		return nil
	}

	// Switch on kind so that we can handle derived types
	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Bool:
		x, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		v.SetBool(x)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		x, err := strconv.ParseInt(s, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetInt(x)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		x, err := strconv.ParseUint(s, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetUint(x)
	case reflect.Float32, reflect.Float64:
		x, err := strconv.ParseFloat(s, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetFloat(x)
	default:
		return fmt.Errorf("cannot parse argument into %s", v.Type().String())
	}
	return nil
}
