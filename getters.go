package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

var errElemNotSet = errors.New("field Config.elem not set, use config.New() or config.SetStruct()")

// HasKey tests if the config struct has a key given
func HasKey(key string) bool { return c.HasKey(key) }

// HasKey tests if the config struct has a key given
func (c *Config) HasKey(key string) bool {
	return hasKey(c.elem, strings.Split(key, "."))
}

// IsEmpty returns true if the value stored at some
// key is a zero value or an empty value
func IsEmpty(key string) bool {
	return c.IsEmpty(key)
}

// IsEmpty returns true if the value stored at some
// key is a zero value or an empty value
func (c *Config) IsEmpty(key string) bool {
	val, err := c.get(key)
	if err != nil {
		return true
	}
	if !val.IsValid() {
		return false
	}
	return val.IsZero()
}

// Get will get a variable by key
func Get(key string) interface{} { return c.Get(key) }

// Get will get a variable by key
func (c *Config) Get(key string) interface{} {
	val, err := c.get(key)
	if err != nil {
		return nil
	}
	return val.Interface()
}

// GetErr will get the value stored at some key and return an error
// if something went wrong.
func GetErr(key string) (interface{}, error) { return c.GetErr(key) }

// GetErr will get the value stored at some key and return an error
// if something went wrong.
func (c *Config) GetErr(key string) (interface{}, error) {
	val, err := c.get(key)
	if err != nil {
		return nil, err
	}
	return val.Interface(), nil
}

func (c *Config) get(key string) (reflect.Value, error) {
	if c.elem.Kind() == reflect.Invalid {
		panic(errElemNotSet)
	}
	keys := strings.Split(key, ".")
	val, err := find(c.elem, keys)
	return val, err
}

// GetString will get the config value by name and
// return it as a string
func GetString(key string) string { return c.GetString(key) }

// GetString will get the config value by name and
// return it as a string. This function will also expand
// any environment variables in the value returned.
func (c *Config) GetString(key string) string {
	s, _ := c.GetStringErr(key)
	return s
}

// GetStringErr is the same as get string but it returns an error
// when something went wrong, mainly if the key does not exist
func GetStringErr(key string) (string, error) {
	return c.GetStringErr(key)
}

// GetStringErr is the same as get string but it returns an error
// when something went wrong, mainly if the key does not exist
func (c *Config) GetStringErr(key string) (string, error) {
	val, err := c.get(key)
	if err != nil {
		return "", err
	}
	return val.String(), nil
}

// Getenv wraps GetString with os.ExpandEnv
func Getenv(key string) string {
	return c.Getenv(key)
}

// Getenv wraps GetString with os.ExpandEnv
func (c *Config) Getenv(key string) string {
	return os.ExpandEnv(c.GetString(key))
}

// GetInt will get the int value of a key
func GetInt(key string) int { return c.GetInt(key) }

// GetInt will get the int value of a key
func (c *Config) GetInt(key string) int {
	i, _ := c.GetIntErr(key)
	return i
}

// GetIntErr will return an get an int but also return an error
// if something went wrong, main just missing keys and conversion errors
func GetIntErr(key string) (int, error) { return c.GetIntErr(key) }

// GetIntErr will return an get an int but also return an error
// if something went wrong, main just missing keys and conversion errors
func (c *Config) GetIntErr(key string) (int, error) {
	val, err := c.get(key)
	if err != nil {
		return 0, err
	}
	return int(val.Int()), nil
}

func (c *Config) GetInt64Err(key string) (int64, error) {
	v, err := c.get(key)
	if err != nil {
		return 0, err
	}
	return v.Int(), err
}
func (c *Config) GetInt64(key string) int64 {
	v, _ := c.GetInt64Err(key)
	return v
}
func GetInt64Err(key string) (int64, error) { return c.GetInt64Err(key) }
func GetInt64(key string) int64             { return c.GetInt64(key) }

func (c *Config) GetInt32Err(key string) (int32, error) {
	v, err := c.GetInt64Err(key)
	return int32(v), err
}
func (c *Config) GetInt32(key string) int32 {
	v, _ := c.GetInt32Err(key)
	return v
}
func GetInt32Err(key string) (int32, error) { return c.GetInt32Err(key) }
func GetInt32(key string) int32             { return c.GetInt32(key) }

func (c *Config) GetUint64Err(key string) (uint64, error) {
	v, err := c.get(key)
	if err != nil {
		return 0, err
	}
	return v.Uint(), nil
}
func (c *Config) GetUint64(key string) uint64 {
	v, _ := c.GetUint64Err(key)
	return v
}
func GetUint64Err(key string) (uint64, error) { return c.GetUint64Err(key) }
func GetUint64(key string) uint64             { return c.GetUint64(key) }

func (c *Config) GetUint32Err(key string) (uint32, error) {
	v, err := c.GetUint64Err(key)
	return uint32(v), err
}
func (c *Config) GetUint32(key string) uint32 {
	v, _ := c.GetUint32Err(key)
	return v
}
func GetUint32Err(key string) (uint32, error) { return c.GetUint32Err(key) }
func GetUint32(key string) uint32             { return c.GetUint32(key) }

func (c *Config) GetUintErr(key string) (uint, error) {
	v, err := c.GetUint64Err(key)
	return uint(v), err
}
func (c *Config) GetUint(key string) uint {
	v, _ := c.GetUintErr(key)
	return v
}
func GetUintErr(key string) (uint, error) { return c.GetUintErr(key) }
func GetUint(key string) uint             { return c.GetUint(key) }

func (c *Config) GetFloatErr(key string) (float64, error) {
	v, err := c.get(key)
	if err != nil {
		return 0.0, err
	}
	return v.Float(), nil
}
func (c *Config) GetFloat(key string) float64 {
	val, err := c.get(key)
	if err != nil {
		return 0.0
	}
	return val.Float()
}
func GetFloatErr(key string) (float64, error) { return c.GetFloatErr(key) }
func GetFloat(key string) float64             { return c.GetFloat(key) }

func (c *Config) GetFloat64Err(key string) (float64, error) { return c.GetFloatErr(key) }
func (c *Config) GetFloat64(key string) float64             { return c.GetFloat(key) }
func GetFloat64Err(key string) (float64, error)             { return c.GetFloatErr(key) }
func GetFloat64(key string) float64                         { return c.GetFloat(key) }

func (c *Config) GetFloat32Err(key string) (float32, error) {
	v, err := c.GetFloatErr(key)
	return float32(v), err
}
func (c *Config) GetFloat32(key string) float32 {
	v, _ := c.GetFloat32Err(key)
	return float32(v)
}
func GetFloat32Err(key string) (float32, error) { return c.GetFloat32Err(key) }
func GetFloat32(key string) float32             { return c.GetFloat32(key) }

// GetBool will get the boolean value at the given key
func GetBool(key string) bool { return c.GetBool(key) }

// GetBool will get the boolean value at the given key
func (c *Config) GetBool(key string) bool {
	val, err := c.get(key)
	if err != nil {
		return false
	}
	return val.Bool()
}

// GetBoolErr will get a boolean value but return an error
// is something went wrong.
func GetBoolErr(key string) (bool, error) {
	return c.GetBoolErr(key)
}

// GetBoolErr will get a boolean value but return an error
// is something went wrong.
func (c *Config) GetBoolErr(key string) (bool, error) {
	val, err := c.get(key)
	if err != nil {
		return false, err
	}
	return val.Bool(), nil
}

// GetIntSlice will get a slice of ints from a key
func GetIntSlice(key string) []int { return c.GetIntSlice(key) }

// GetIntSlice will get a slice of ints from a key
//
// Warning: will panic if the key does not reference
// a []int
func (c *Config) GetIntSlice(key string) []int {
	val, err := c.get(key)
	if err != nil {
		return nil
	}
	if val.Kind() != reflect.Slice {
		return nil
	}
	ret, ok := val.Interface().([]int)
	if !ok {
		return nil
	}
	return ret
}

// GetInt64Slice will return a slice of int64.
//
// Warning: will panic if the key given does not
// reference a []int64
func GetInt64Slice(key string) []int64 { return c.GetInt64Slice(key) }

// GetInt64Slice will return a slice of int64.
//
// Warning: will panic if the key given does not
// reference a []int64
func (c *Config) GetInt64Slice(key string) []int64 {
	res, err := c.get(key)
	if err != nil {
		return nil
	}
	if res.Kind() != reflect.Slice {
		return nil
	}
	ret, ok := res.Interface().([]int64)
	if !ok {
		return nil
	}
	return ret
}

// GetStringMap will get a map of string keys to string values
func GetStringMap(key string) map[string]string {
	return c.GetStringMap(key)
}

// GetStringMap will get a map of string keys to string values
func (c *Config) GetStringMap(key string) map[string]string {
	res, err := c.get(key)
	if err != nil {
		return nil
	}
	if res.Kind() != reflect.Map {
		return nil
	}
	m := make(map[string]string)
	iter := res.MapRange()
	for iter.Next() {
		m[iter.Key().String()] = iter.Value().String()
	}
	return m
}

func find(val reflect.Value, keyPath []string) (reflect.Value, error) {
	var err error
	typ := val.Type()
	n := typ.NumField()
	for i := 0; i < n; i++ {
		typFld := typ.Field(i)
		// if the first key is the same as the fieldname
		if isCorrectLabel(keyPath[0], typFld) {
			value := val.Field(i)
			if len(keyPath) > 1 {
				return find(value, keyPath[1:])
			}
			if !isZero(value) {
				// if the field has been set then we return it
				return value, nil
			}

			defvalue, err := getDefaultValue(&typFld, &value)
			switch err {
			case errNoDefaultValue:
				return value, nil
			case nil:
				return defvalue, nil
			default: // err != nil
				return defvalue, err
			}
		}
	}
	if err == nil {
		err = ErrFieldNotFound
	}
	return nilval, err
}

func hasKey(val reflect.Value, keyPath []string) bool {
	typ := val.Type()
	n := typ.NumField()
	for i := 0; i < n; i++ {
		typFld := typ.Field(i)
		if isCorrectLabel(keyPath[0], typFld) {
			if len(keyPath) == 1 {
				return true
			}
			return hasKey(val.Field(i), keyPath[1:])
		}
	}
	return false
}

func setDefaults(val reflect.Value) (err error) {
	var seterr error
	typ := val.Type()
	n := typ.NumField()
	for i := 0; i < n; i++ {
		fldVal := val.Field(i)  // field's value
		fldType := typ.Field(i) // field's type

		// make recursive calls
		if fldVal.Kind() == reflect.Struct {
			err := setDefaults(fldVal)
			if seterr == nil {
				seterr = err
			}
			continue
		}

		// if the field has been set already, then
		// it is a significant value to the user
		// do not override with defaults
		// Also, if the field is not exported then we
		// cannot call isZero on it.
		if (fldType.Name[0] >= 64 && fldType.Name[0] <= 90) && !isZero(fldVal) {
			continue
		}

		defval, err := getDefaultValue(&fldType, &fldVal)
		switch err {
		case nil: // break out of switch
		case errNoDefaultValue:
			continue
		default:
			return err
		}
		if fldVal.CanSet() {
			fldVal.Set(defval)
		} else {
			if seterr == nil {
				seterr = fmt.Errorf("cannot set value for field '%s'", fldType.Name)
			}
			continue
		}
	}
	return seterr
}

var errNoDefaultValue = errors.New("no default value found")

func getDefaultValue(fld *reflect.StructField, fldval *reflect.Value) (def reflect.Value, err error) {
	val := fld.Tag.Get("default")
	env := fld.Tag.Get("env")
	if env != "" {
		val = os.Getenv(env)
	}
	if val == "" {
		return nilval, errNoDefaultValue
	}
	return valueFromString(val, fld, fldval)
}

func valueFromString(
	val string,
	fld *reflect.StructField,
	fldval *reflect.Value,
) (result reflect.Value, err error) {
	var (
		ival  int64
		uival uint64
		fval  float64
	)

	switch fld.Type.Kind() {
	case reflect.String:
		return reflect.ValueOf(val), nil
	case reflect.Int:
		ival, err = strconv.ParseInt(val, 10, 64)
		result = reflect.ValueOf(int(ival))
	case reflect.Int8:
		ival, err = strconv.ParseInt(val, 10, 8)
		result = reflect.ValueOf(int8(ival))
	case reflect.Int16:
		ival, err = strconv.ParseInt(val, 10, 16)
		result = reflect.ValueOf(int16(ival))
	case reflect.Int32:
		ival, err = strconv.ParseInt(val, 10, 32)
		result = reflect.ValueOf(int32(ival))
	case reflect.Int64:
		ival, err = strconv.ParseInt(val, 10, 64)
		result = reflect.ValueOf(int64(ival))
	case reflect.Uint:
		uival, err = strconv.ParseUint(val, 10, 64)
		result = reflect.ValueOf(uint(uival))
	case reflect.Uint8:
		uival, err = strconv.ParseUint(val, 10, 8)
		result = reflect.ValueOf(uint8(uival))
	case reflect.Uint16:
		uival, err = strconv.ParseUint(val, 10, 16)
		result = reflect.ValueOf(uint16(uival))
	case reflect.Uint32:
		uival, err = strconv.ParseUint(val, 10, 32)
		result = reflect.ValueOf(uint32(uival))
	case reflect.Uint64:
		uival, err = strconv.ParseUint(val, 10, 64)
		result = reflect.ValueOf(uival)
	case reflect.Float32:
		fval, err = strconv.ParseFloat(val, 32)
		result = reflect.ValueOf(float32(fval))
	case reflect.Float64:
		fval, err = strconv.ParseFloat(val, 64)
		result = reflect.ValueOf(fval)
	case reflect.Bool:
		var bval bool
		bval, err = strconv.ParseBool(val)
		result = reflect.ValueOf(bval)
	case reflect.Slice:
		switch fldval.Interface().(type) {
		case []byte:
			result = reflect.ValueOf([]byte(val))
		default:
			panic(fmt.Sprintf("don't know how to parse %v yet", fld.Type.Kind()))
		}
	case reflect.Complex64:
		// TODO
	case reflect.Complex128:
		// TODO
	case reflect.Func:
	default:
		return nilval, errors.New("unknown default config type")
	}
	if err != nil {
		return nilval, fmt.Errorf("could not parse default value: %v", err)
	}
	return result, err
}

func isCorrectLabel(key string, field reflect.StructField) bool {
	if len(key) == 0 {
		return false
	}

	var parts []string
	// TODO don't look for the "json" tag if the filetype
	// has been set as yaml and vice versa.
	for _, tag := range []string{"config", "yaml", "json"} {
		parts = strings.Split(field.Tag.Get(tag), ",")
		if len(parts) > 0 && parts[0] == key {
			return true
		}
	}
	return field.Name == key
}
