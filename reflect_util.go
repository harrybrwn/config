package config

import (
	"errors"
	"reflect"
	"strings"
)

func isZero(val reflect.Value) bool {
	return reflect.DeepEqual(
		val.Interface(),
		reflect.Zero(val.Type()).Interface(),
	)
}

func copyVal(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	var cp reflect.Value

	switch v.Kind() {
	case reflect.Array:
		t := reflect.ArrayOf(v.Len(), v.Type().Elem())
		cp = reflect.New(t).Elem()
		reflect.Copy(cp, v)
	case reflect.Slice:
		cp = reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
		reflect.Copy(cp, v)
	case reflect.Struct:
		cp = reflect.New(v.Type()).Elem()

		for i := 0; i < v.NumField(); i++ {
			vf := v.Field(i)
			cf := cp.Field(i)
			switch vf.Kind() {
			case reflect.Ptr:
				if vf.IsNil() {
					continue
				}
				fieldcopy := copyVal(vf.Elem())
				cf = reflect.New(vf.Elem().Type())
				cf.Elem().Set(fieldcopy)
				cp.Field(i).Set(cf)
			default:
				cp.Field(i).Set(copyVal(vf))
			}
		}
	case reflect.Map:
		cp = reflect.MakeMap(v.Type())
		for _, key := range v.MapKeys() {
			cp.SetMapIndex(key, copyVal(v.MapIndex(key)))
		}
	default:
		cp = reflect.New(v.Type()).Elem()
		cp.Set(v)
	}
	return cp
}

var errMismatchedTypes = errors.New("mismatched types")

// merge the fields of src into dst if they have not
// already been set.
func merge(dst, src reflect.Value) error {
	if src.Kind() == reflect.Ptr {
		src = src.Elem()
	}
	if dst.Kind() == reflect.Ptr {
		dst = dst.Elem()
	}
	if dst.Kind() != src.Kind() {
		return errMismatchedTypes
	}

	var err error
	switch dst.Kind() {
	case reflect.Struct:
		for i := 0; i < src.NumField(); i++ {
			vf := src.Field(i)
			cf := dst.Field(i)

			// If there is no value to set, then skip it
			if vf.IsZero() {
				continue
			}
			switch vf.Kind() {
			case reflect.Ptr:
				// Copy of nil is useless
				if vf.IsNil() {
					continue
				}
				if cf.IsNil() {
					cf = reflect.New(vf.Elem().Type())
				}
				err = merge(cf, vf)
				if err != nil {
					return err
				}
				dst.Field(i).Set(cf)
			default:
				err = merge(cf, vf)
				if err != nil {
					return err
				}
				dst.Field(i).Set(cf)
			}
		}

	case reflect.Map:
		var dstval, srcval reflect.Value
		if dst.IsNil() {
			dst.Set(reflect.MakeMap(src.Type()))
		}
		for _, key := range src.MapKeys() {
			dstval = dst.MapIndex(key)
			srcval = src.MapIndex(key)
			// if the key is not in dst, then
			// copy the value from the source map
			// and insert it into the dest
			if !dstval.IsValid() {
				cp := copyVal(srcval)
				if srcval.Kind() == reflect.Ptr {
					dst.SetMapIndex(key, cp.Addr())
				} else {
					dst.SetMapIndex(key, cp)
				}
			} else {
				err = merge(dstval, srcval)
				if err != nil {
					return err
				}
				dst.SetMapIndex(key, dstval)
			}
		}
	default:
		if dst.IsZero() {
			dst.Set(src)
		}
	}
	return nil
}

func set(obj interface{}, key string, val interface{}) error {
	objval := reflect.ValueOf(obj).Elem() // BUG: don't use Elem for everything
	field, err := find(objval, strings.Split(key, "."))
	if err != nil {
		return err
	}
	if !field.CanSet() {
		return errors.New("cannot set value")
	}

	var exptype reflect.Kind
	switch v := val.(type) {
	case string:
		exptype = reflect.String
		field.SetString(v)
	case []byte:
		exptype = reflect.Slice
		field.SetBytes(v)
	case bool:
		exptype = reflect.Bool
		field.SetBool(v)
	case complex64:
		exptype = reflect.Complex64
		field.SetComplex(complex128(v))
	case complex128:
		exptype = reflect.Complex128
		field.SetComplex(v)
	case int:
		exptype = reflect.Int
		field.SetInt(int64(v))
	case int8:
		exptype = reflect.Int8
		field.SetInt(int64(v))
	case int32:
		exptype = reflect.Int32
		field.SetInt(int64(v))
	case int64:
		exptype = reflect.Int64
		field.SetInt(int64(v))
	case float32:
		exptype = reflect.Float32
		field.SetFloat(float64(v))
	case float64:
		exptype = reflect.Float64
		field.SetFloat(v)
	default:
		field.Set(reflect.ValueOf(val))
		return nil
	}
	if field.Kind() != exptype {
		return ErrWrongType
	}
	return nil
}
