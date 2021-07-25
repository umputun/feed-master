// Copyright (c) 2020 Xelaj Software
//
// This file is a part of go-dry package.
// See https://github.com/xelaj/go-dry/blob/master/LICENSE for details

package dry

// FirstArg returns the first passed argument,
// can be used to extract first result value
// from a function call to pass it on to functions like fmt.Printf
func FirstArg(args ...any) any {
	return args[0]
}

func ConvertInt(n interface{}) int {
	switch n := n.(type) {
	case int:
		return n
	case int8:
		return int(n)
	case int16:
		return int(n)
	case int32:
		return int(n)
	case int64:
		return int(n)
	case uint:
		return int(n)
	case uintptr:
		return int(n)
	case uint8:
		return int(n)
	case uint16:
		return int(n)
	case uint32:
		return int(n)
	case uint64:
		return int(n)
	default:
		panic("value is not an integer in any way")
	}
}

func ConvertFloat(n interface{}) float64 {
	switch n := n.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	default:
		panic("value is not a float in any way")
	}
}
