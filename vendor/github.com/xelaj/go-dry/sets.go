package dry

import "reflect"

func SetUnify(a, b any) any {
	aval := reflect.ValueOf(a)
	if aval.Type().Kind() != reflect.Slice {
		panic("first element is not a slice: " + aval.Type().String())
	}
	bval := reflect.ValueOf(b)
	if bval.Type().Kind() != reflect.Slice {
		panic("second element is not a slice: " + bval.Type().String())
	}

	if aval.Type().Elem() != bval.Type().Elem() {
		panic("slices has different types")
	}

	pretotal := reflect.MakeMap(reflect.MapOf(aval.Type().Elem(), reflect.TypeOf(null{})))

	for i := 0; i < aval.Len(); i++ {
		pretotal.SetMapIndex(aval.Index(i), reflect.ValueOf(null{}))
	}
	for i := 0; i < bval.Len(); i++ {
		pretotal.SetMapIndex(bval.Index(i), reflect.ValueOf(null{}))
	}

	res := reflect.New(aval.Type()).Elem()

	for _, key := range pretotal.MapKeys() {
		res = reflect.Append(res, key)
	}

	return res.Interface()
}

// пока только строчки, потом сделаю нормальный интерфейс
func SetsEqual(a, b []string) bool {
	// If one is nil, the other must also be nil.
	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
