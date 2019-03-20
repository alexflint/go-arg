// Package ecp can help you convert environments into configurations
// it's an environment config parser
package ecp

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	GetKey      func(parentName, structName string, tag reflect.StructTag) (key string)
	LookupValue func(field reflect.Value, key string) (value string, exist bool)
	IgnoreKey   func(field reflect.Value, key string) bool

	// env get functions
	EnvGetKey = func(parentName, structName string, tag reflect.StructTag) (key string) {
		key = strings.ToUpper(parentName + "_" + structName)
		if e := tag.Get("env"); e != "" {
			key = strings.Split(e, ",")[0]
		}
		return
	}
	EnvLookupValue = func(field reflect.Value, key string) (value string, exist bool) {
		return os.LookupEnv(key)
	}
	EnvIgnore = func(field reflect.Value, key string) bool {
		return key == "-"
	}
)

func init() {
	GetKey = EnvGetKey
	IgnoreKey = EnvIgnore
	LookupValue = EnvLookupValue
}

func toValue(config interface{}) reflect.Value {
	value, ok := config.(reflect.Value)
	if !ok {
		value = reflect.Indirect(reflect.ValueOf(config))
	}
	return value
}

func getAll(configType reflect.Type, configValue reflect.Value,
	i int, parentName string) (field reflect.Value, structName string,
	keyName string, defaultV string) {

	field = configValue.Field(i)
	structName = configType.Field(i).Name
	tag := configType.Field(i).Tag

	// config often use yaml, so I use yaml
	if y := tag.Get("yaml"); y != "" {
		structName = strings.Split(y, ",")[0]
	}

	keyName = GetKey(parentName, structName, tag)

	defaultV = tag.Get("default")
	return
}

func parseSlice(v string, field reflect.Value) error {
	if v == "" {
		return nil
	}
	// either space nor commas is perfect, but I think space is better
	// since it's more natural: fmt.Println([]int{1, 2, 3}) = [1 2 3]
	stringSlice := strings.Split(v, " ") // split by space
	field.Set(reflect.MakeSlice(field.Type(),
		len(stringSlice), cap(stringSlice)))

	switch field.Type().String() {
	case "[]string":
		field.Set(reflect.ValueOf(stringSlice))
	case "[]int":
		intSlice := []int{}
		for _, s := range stringSlice {
			i, err := strconv.Atoi(s)
			if err != nil {
				return err
			}
			intSlice = append(intSlice, i)
		}
		field.Set(reflect.ValueOf(intSlice))
	case "[]float32":
		floatSlice := []float32{}
		for _, s := range stringSlice {
			i, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return err
			}
			floatSlice = append(floatSlice, float32(i))
		}
		field.Set(reflect.ValueOf(floatSlice))

	}
	return nil
}

func rangeOver(config interface{}, parseDefault,
	findKey bool, parentName, findName string) (reflect.Value, error) {
	configValue := toValue(config)
	configType := configValue.Type()
	for i := 0; i < configValue.NumField(); i++ {
		field, structName, keyName, defaultV := getAll(configType, configValue, i, parentName)

		if findKey {
			pName := ""
			if parentName == "" {
				pName = structName
			} else {
				pName = parentName + "." + structName
			}
			if !strings.HasPrefix(findName, pName) {
				continue
			}
			if findName == pName {
				return field, nil
			}
		}

		// ignore this key
		if IgnoreKey(field, structName) || IgnoreKey(field, keyName) {
			continue
		}

		v, exist := LookupValue(field, keyName)
		if parseDefault && !exist {
			v = defaultV
		}

		if !field.CanAddr() {
			continue
		}

		kind := field.Kind()
		if v == "" && kind != reflect.Struct {
			continue
		}

		switch kind {
		case reflect.String:
			if field.String() != "" && !exist {
				continue
			}
			field.SetString(v)

		case reflect.Float32, reflect.Float64:
			if field.Float() != 0 && !exist {
				continue
			}
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return field, fmt.Errorf("convert %s error: %s", keyName, err)
			}
			field.SetFloat(f)

		case reflect.Int, reflect.Int8,
			reflect.Int16, reflect.Int32, reflect.Int64:
			if field.Int() != 0 && !exist {
				continue
			}
			// since duration is int64 too, parse it first
			// if the duration contains `d` (day), we should support it
			// fix #6
			last := len(v) - 1
			if last > 0 && v[last] == 'd' {
				day := v[:last]
				dayN, err := strconv.Atoi(day)
				if err != nil {
					return field, fmt.Errorf("convert %s error: %s", keyName, err)
				}
				v = fmt.Sprintf("%dh", dayN*24)
			}
			d, err := time.ParseDuration(v)
			if err == nil {
				field.SetInt(int64(d))
				continue
			}
			vint, err := strconv.Atoi(v)
			if err != nil {
				return field, fmt.Errorf("convert %s error: %s", keyName, err)
			}
			field.SetInt(int64(vint))

		case reflect.Uint, reflect.Uint8,
			reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if field.Uint() != 0 && !exist {
				continue
			}
			vint, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return field, fmt.Errorf("convert %s error: %s", keyName, err)
			}
			field.SetUint(vint)

		case reflect.Bool:
			b, err := strconv.ParseBool(strings.ToLower(v))
			if err != nil {
				return field, err
			}
			if !exist && field.Bool() != b {
				continue
			}
			field.SetBool(b)

		case reflect.Slice:
			if !field.IsNil() && !exist {
				continue
			}
			if err := parseSlice(v, field); err != nil {
				return field, err
			}

		case reflect.Struct:
			pref := parentName + "_" + structName
			if findKey {
				pref = structName
				if parentName != "" {
					pref = parentName + "." + structName
				}
			}
			v, err := rangeOver(field, parseDefault, findKey, pref, findName)
			if err != nil {
				return field, err
			} else if findKey {
				return v, nil
			} else {
				field = v
			}

		case reflect.Ptr:
			// only set default value to nil pointer
			if field.Pointer() != 0 {
				continue
			}
			var defaultValue interface{}
			typeString := field.Type().String()[1:]
			switch typeString {
			case reflect.String.String():
				defaultValue = &defaultV

			case reflect.Int.String(), reflect.Int8.String(), reflect.Int16.String(),
				reflect.Int32.String(), reflect.Int64.String():
				vInt, err := strconv.ParseInt(defaultV, 10, 64)
				if err != nil {
					return field, fmt.Errorf("convert %s error: %s", keyName, err)
				}
				switch typeString {
				case reflect.Int.String():
					_int := int(vInt)
					defaultValue = &_int
				case reflect.Int8.String():
					_int8 := int8(vInt)
					defaultValue = &_int8
				case reflect.Int16.String():
					_int16 := int16(vInt)
					defaultValue = &_int16
				case reflect.Int32.String():
					_int32 := int32(vInt)
					defaultValue = &_int32
				case reflect.Int64.String():
					defaultValue = &vInt
				}

			case reflect.Bool.String():
				b, err := strconv.ParseBool(strings.ToLower(v))
				if err != nil {
					return field, err
				}
				defaultValue = &b
			}

			field.Set(reflect.ValueOf(defaultValue))
		}

	}
	return reflect.Value{}, nil
}

// note Parse function will overwrite the existing value if there is a
// environment configration matched with the struct name or the "env" tag
// name.

// Parse the configuration through environments starting with the prefix
// or you can ignore the prefix and the default prefix key will be `ECP`
// ecp.Parse(&config) or ecp.Parse(&config, "PREFIX")
func Parse(config interface{}, prefix ...string) error {
	if prefix == nil {
		prefix = []string{"ECP"}
	}
	_, err := rangeOver(config, false, false, prefix[0], "")
	return err
}

// the default value of the config is set by a tag named "default"
// for example, you can define a struct like:
//
//    type config struct {
//        One   string   `default:"1"`
//        Two   int      `default:"2"`
//        Three []string `default:"1,2,3"`
//    }
//    c := &config{}
//
// then you can use ecp.Default(&c) to parse the default value to the struct.
// note, the Default function will not overwrite the existing value, if the
// config key has already been set no matter whether it has a default tag.
// And the default value will be nil (nil of the type) if the "default" tag is
// empty.

// Default set config with its default value
func Default(config interface{}) error {
	_, err := rangeOver(config, true, false, "", "")
	return err
}

// List function will also fill up the value of the environment key
// it the "default" tag has value

// List all the config environments
func List(config interface{}, prefix ...string) []string {
	list := []string{}

	if prefix == nil {
		prefix = []string{"ECP"}
	}
	parentName := prefix[0]

	configValue := toValue(config)
	configType := configValue.Type()
	for i := 0; i < configValue.NumField(); i++ {
		field, structName, keyName,
			d := getAll(configType, configValue, i, prefix[0])
		if structName == "-" || keyName == "" {
			continue
		}
		switch field.Kind() {
		case reflect.Struct:
			p := GetKey(parentName, structName, "")
			list = append(list, List(field, p)...)
		default:
			if strings.Contains(d, " ") {
				d = fmt.Sprintf("\"%s\"", d)
			}
			list = append(list, keyName+"="+d)
		}
	}

	return list
}

// Get the value of the keyName in that struct
func getValue(config interface{}, keyName string) (reflect.Value, error) {
	v, err := rangeOver(config, false, true, "", keyName)
	if err != nil {
		return reflect.Value{}, err
	}

	if !v.IsValid() {
		return reflect.Value{}, fmt.Errorf("invalid type %v", v)
	}
	return v, err
}

// Get the value of the keyName in that struct
func Get(config interface{}, keyName string) (interface{}, error) {
	v, err := getValue(config, keyName)
	if err != nil {
		return nil, err
	}

	return v.Interface(), nil
}

// GetBool returns bool
func GetBool(config interface{}, keyName string) (bool, error) {
	v, err := getValue(config, keyName)
	if err != nil {
		return false, err
	}

	if vv, ok := v.Interface().(bool); ok {
		return vv, nil
	}
	return false, fmt.Errorf("value is not bool, it's %s", v.Kind())
}

// GetInt64 returns int64
func GetInt64(config interface{}, keyName string) (int64, error) {
	v, err := getValue(config, keyName)
	if err != nil {
		return -1, err
	}

	if vv, ok := v.Interface().(int); ok {
		return int64(vv), nil
	}
	if vv, ok := v.Interface().(int32); ok {
		return int64(vv), nil
	}
	if vv, ok := v.Interface().(int64); ok {
		return vv, nil
	}
	return -1, fmt.Errorf("value is not int64, it's %s", v.Kind())
}

// GetString returns string
func GetString(config interface{}, keyName string) (string, error) {
	v, err := getValue(config, keyName)
	if err != nil {
		return "", err
	}

	if vv, ok := v.Interface().(string); ok {
		return vv, nil
	}
	return "", fmt.Errorf("value is not string, it's %s", v.Kind())
}

// GetFloat64 returns float64
func GetFloat64(config interface{}, keyName string) (float64, error) {
	v, err := getValue(config, keyName)
	if err != nil {
		return -1, err
	}

	if vv, ok := v.Interface().(float32); ok {
		return float64(vv), nil
	}
	if vv, ok := v.Interface().(float64); ok {
		return vv, nil
	}
	return -1, fmt.Errorf("value is not float64, it's %s", v.Kind())
}
