// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE-go file.

// This code is heavily based on text/template. List of changes below:
// - basicKind modified to return no error and widest kind instead of const
// - indirect modifed to return value only
// - gt, ge, lt, le, eq and ne are modified to compare floats to ints/uints,
//   use changed basicKind and not to panic
// - gt, ge, lt, le return bool and error
// - eq uses fmt.Sprint as a last resort to compare values of the same type

package govtl

import (
	"errors"
	"fmt"
	"reflect"
)

func basicKind(v reflect.Value) reflect.Kind {
	switch v.Kind() {
	case reflect.Bool:
		return reflect.Bool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflect.Int64
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return reflect.Uint64
	case reflect.Float32, reflect.Float64:
		return reflect.Float64
	case reflect.Complex64, reflect.Complex128:
		return reflect.Complex128
	case reflect.String:
		return reflect.String
	}
	return reflect.Invalid
}

func indirect(v reflect.Value) reflect.Value {
	for ; v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface; v = v.Elem() {
		if v.IsNil() {
			return v
		}
	}
	return v
}

var errNaNLeft = errors.New("left side of comparison operation is not a number")
var errNaNRight = errors.New("right side of comparison operation is not a number")

func lt(v1, v2 reflect.Value) (bool, error) {
	v1, v2 = indirect(v1), indirect(v2)
	if !isNumber(v1) {
		return false, errNaNLeft
	}
	if !isNumber(v2) {
		return false, errNaNRight
	}
	k1, k2 := basicKind(v1), basicKind(v2)
	if k1 != k2 {
		switch {
		case k1 == reflect.Int64 && k2 == reflect.Uint64:
			return v1.Int() < 0 || uint64(v1.Int()) < v2.Uint(), nil
		case k1 == reflect.Uint64 && k2 == reflect.Int64:
			return v2.Int() >= 0 && v1.Uint() < uint64(v2.Int()), nil
		case k1 == reflect.Float64 && k2 == reflect.Uint64:
			return v1.Float() < 0 || v1.Float() < float64(v2.Uint()), nil
		case k1 == reflect.Uint64 && k2 == reflect.Float64:
			return v2.Float() >= 0 && float64(v1.Uint()) < v2.Float(), nil
		case k1 == reflect.Float64 && k2 == reflect.Int64:
			return v1.Float() < float64(v2.Int()), nil
		case k1 == reflect.Int64 && k2 == reflect.Float64:
			return float64(v1.Int()) < v2.Float(), nil
		default:
			return false, nil
		}
	}
	switch k1 {
	case reflect.Float64:
		return v1.Float() < v2.Float(), nil
	case reflect.Int64:
		return v1.Int() < v2.Int(), nil
	case reflect.Uint64:
		return v1.Uint() < v2.Uint(), nil
	default:
		return false, nil
	}
}

func le(v1, v2 reflect.Value) (bool, error) {
	less, err := lt(v1, v2)
	if err != nil {
		return false, err
	}
	return less || eq(v1, v2), nil
}

func gt(v1, v2 reflect.Value) (bool, error) {
	v1, v2 = indirect(v1), indirect(v2)
	less, err := le(v1, v2)
	if err == nil {
		return !less, nil
	}
	return false, err
}

func ge(v1, v2 reflect.Value) (bool, error) {
	greater, err := gt(v1, v2)
	if err != nil {
		return false, err
	}
	return greater || eq(v1, v2), nil
}

func eq(v1, v2 reflect.Value) bool {
	v1, v2 = indirect(v1), indirect(v2)
	k1, k2 := basicKind(v1), basicKind(v2)
	if k1 != k2 {
		switch {
		case k1 == reflect.Int64 && k2 == reflect.Uint64:
			return v1.Int() >= 0 && uint64(v1.Int()) == v2.Uint()
		case k1 == reflect.Uint64 && k2 == reflect.Int64:
			return v2.Int() >= 0 && v1.Uint() == uint64(v2.Int())
		case k1 == reflect.Float64 && k2 == reflect.Uint64:
			return v1.Float() >= 0 && v1.Float() == float64(v2.Uint())
		case k1 == reflect.Float64 && k2 == reflect.Int64:
			return v1.Float() == float64(v2.Int())
		case k1 == reflect.Int64 && k2 == reflect.Float64:
			return float64(v1.Int()) == v2.Float()
		case k1 == reflect.Uint64 && k2 == reflect.Float64:
			return v2.Float() >= 0 && float64(v1.Uint()) == v2.Float()
		default:
			return false
		}
	}
	switch k1 {
	case reflect.Bool:
		return v1.Bool() == v2.Bool()
	case reflect.Float64:
		return v1.Float() == v2.Float()
	case reflect.Int64:
		return v1.Int() == v2.Int()
	case reflect.String:
		return v1.String() == v2.String()
	case reflect.Uint64:
		return v1.Uint() == v2.Uint()
	case reflect.Invalid:
		return false
	default:
		return fmt.Sprint(v1.Interface()) == fmt.Sprint(v2.Interface())
	}
}

func ne(v1, v2 reflect.Value) bool { return !eq(v1, v2) }
