// Copyright (c) 2020 Xelaj Software
//
// This file is a part of go-dry package.
// See https://github.com/xelaj/go-dry/blob/master/LICENSE for details

package slices

import (
	"fmt"
	"reflect"
)

// индекс элемента T в []T
// выводит индекс найденого элемента, либо -1, если элемент не найден
func Index(slice, item any) int {
	ival := reflect.ValueOf(slice)
	if ival.Type().Kind() != reflect.Slice {
		panic("not a slice: " + ival.Type().String())
	}
	if ival.Type().Elem().String() != reflect.TypeOf(item).String() {
		panic("different types of slice and item")
	}

	for i := 0; i < ival.Len(); i++ {
		if reflect.DeepEqual(ival.Index(i).Interface(), item) {
			return i
		}
	}

	return -1
}

//func Index(slice []string, item string) int {
//	for i, elem := range slice {
//		if item == elem {
//			return i
//		}
//	}
//
//	return -1
//}

func Contains(slice, item any) bool {
	return Index(slice, item) != -1
}

func DeleteIndex(slice any, i int) any {
	return cutSliceWithMode(slice, i, i+1, true)
}

func Cut(slice any, i, j int) any {
	return cutSliceWithMode(slice, i, j, false)
}

func cutSliceWithMode(slice any, i, j int, deleteInsteadCut bool) any {
	panicIndexStr := fmt.Sprintf("[%v:%v]", i, j)
	if deleteInsteadCut {
		panicIndexStr = fmt.Sprintf("[%v]", i)
	}

	ival := reflect.ValueOf(slice)
	if ival.Type().Kind() != reflect.Slice {
		panic("not a slice: " + ival.Type().String())
	}

	if i > j {
		panic("end less than start " + panicIndexStr)
	}
	if i < 0 || j < 0 {
		panic("slice index " + panicIndexStr + " out of bounds")
	}
	if ival.Len()-1 < i || ival.Len() < j {
		panic(fmt.Sprintf("index out of range %v with length %v", panicIndexStr, ival.Len()))
	}

	return reflect.AppendSlice(ival.Slice(0, i), ival.Slice(j, ival.Len())).Interface()
}

// Expand всталвяет в массив slice в элемент i дополнительные пустые поля в количестве j
func Expand(slice any, i, j int) any {
	panicIndexStr := fmt.Sprintf("[%v]", i)

	ival := reflect.ValueOf(slice)
	if ival.Type().Kind() != reflect.Slice {
		panic("not a slice: " + ival.Type().String())
	}

	if i < 0 {
		panic("slice index " + panicIndexStr + " out of bounds")
	}
	if j < 0 {
		panic(fmt.Sprintf("can't expand slice on %v points", j))
	}
	if ival.Len()-1 < i {
		panic(fmt.Sprintf("index out of range %v with length %v", panicIndexStr, ival.Len()))
	}

	zeroitems := reflect.MakeSlice(ival.Type(), j, j)
	part := reflect.AppendSlice(zeroitems, ival.Slice(i, ival.Len()))
	return reflect.AppendSlice(ival.Slice(0, i), part).Interface()
}

func ToInterfaceSlice(in any) []any {
	if in == nil {
		return nil
	}

	ival := reflect.ValueOf(in)
	if ival.Type().Kind() != reflect.Slice {
		panic("not a slice: " + ival.Type().String())
	}

	res := make([]any, ival.Len())

	for i := 0; i < ival.Len(); i++ {
		res[i] = ival.Index(i).Interface()
	}
	return res
}

// TODO: неправильная имплементация
// []<T> -> map[<T>]struct{}
func Unique(in any) any {
	ival := reflect.ValueOf(in)
	if ival.Type().Kind() != reflect.Slice {
		panic("not a slice: " + ival.Type().String())
	}

	res := reflect.MakeMap(reflect.MapOf(ival.Type().Elem(), reflect.TypeOf(null{})))

	for i := 0; i < ival.Len(); i++ {
		res.SetMapIndex(ival.Index(i), reflect.ValueOf(null{}))
	}
	return res.Interface()
}

//func Unique(in []string) []string {
//	tmp := make(map[string]null)
//
//	for _, item := range in {
//		tmp[item] = null{}
//	}
//
//	res := make([]string, 0, len(in))
//	for _, item := range in {
//		_, ok := tmp[item]
//		if ok {
//			res = append(res, item)
//			delete(tmp, item)
//		}
//	}
//	return res
//}

// Assume element type is int.
// func Insert(s []int, k int, vs ...int) []int {
// 	if n := len(s) + len(vs); n <= cap(s) {
// 		s2 := s[:n]
// 		copy(s2[k+len(vs):], s[k:])
// 		copy(s2[k:], vs)
// 		return s2
// 	}
// 	s2 := make([]int, len(s) + len(vs))
// 	copy(s2, s[:k])
// 	copy(s2[k:], vs)
// 	copy(s2[k+len(vs):], s[k:])
// 	return s2
// }

// func Reverse() {}

// https://godoc.org/math/rand#Shuffle это должен быть алиас для встроенной функции но только чтоб не нужно
// было прописывать функцию свапа как в стандартном пакете
// func Shuffle() {}

//// moveToFront moves needle to the front of haystack, in place if possible.
//func moveToFront(needle string, haystack []string) []string {
//	if len(haystack) == 0 || haystack[0] == needle {
//		return haystack
//	}
//	var prev string
//	for i, elem := range haystack {
//		switch {
//		case i == 0:
//			haystack[0] = needle
//			prev = elem
//		case elem == needle:
//			haystack[i] = prev
//			return haystack
//		default:
//			haystack[i] = prev
//			prev = elem
//		}
//	}
//	return append(haystack, prev)
//}

// func slidingWindow(size int, input []int) [][]int {
// 	// returns the input slice as the first element
// 	if len(input) <= size {
// 		return [][]int{input}
// 	}
//
// 	// allocate slice at the precise size we need
// 	r := make([][]int, 0, len(input)-size+1)
//
// 	for i, j := 0, size; j <= len(input); i, j = i+1, j+1 {
// 		r = append(r, input[i:j])
// 	}
//
// 	return r
// }

// SliceUnique должна удалять продублированные элементы при этом НЕ ИЗМЕНЯЯ ПОРЯДОК! мапа то порядок же изменяет
// FIXME: сейчас не работает
// []<T> -> []<T>
//func SliceUnique(in any) any {
//	ival := reflect.ValueOf(in)
//	if ival.Type().Kind() != reflect.Slice {
//		panic("not a slice: " + ival.Type().String())
//	}
//
//	res := reflect.MakeMap(reflect.MapOf(ival.Type().Elem(), reflect.TypeOf(null{})))
//
//	for i := 0; i < ival.Len(); i++ {
//		res.SetMapIndex(ival.Index(i), reflect.ValueOf(null{}))
//	}
//	return res.Interface()
//}

// func Map() {} аналог какоо нибудь дартовского [].map(), ремапим слайс и отдаем итог

var reflectInt = reflect.TypeOf(int(0))

// @interface SliceRemoveFunc(slice []<T>, f func(<T>)bool) []<T>
func RemoveFunc(slice any, f func(i any) bool) any {
	res, _ := PopFunc(slice, f)
	return res
}

//func RemoveFunc(slice []string, f func(i string) bool) []string {
//	res, _ := SlicePopFunc(slice, f)
//	return res
//}

// if f func returns true, than slice item will pop
// @interface SlicePopFunc(slice []<T>, f func(<T>)bool) (res, popped []<T>)
func PopFunc(slice any, f func(i any) bool) (res, popped any) {
	sVal := reflect.ValueOf(slice)
	if sVal.Type().Kind() != reflect.Slice {
		panic("not a slice: " + sVal.Type().Kind().String())
	}

	poppedVal := reflect.MakeSlice(sVal.Type(), 0, sVal.Len())
	mapView := reflect.MakeMap(reflect.MapOf(reflectInt, sVal.Type().Elem()))
	keys := make([]int, 0, sVal.Len())
	for i := 0; i < sVal.Len(); i++ {
		item := sVal.Index(i)
		if !f(item.Interface()) {
			mapView.SetMapIndex(reflect.ValueOf(i), item)
			keys = append(keys, i)
		} else {
			poppedVal = reflect.Append(poppedVal, item)
		}
	}
	// keys are already sorted

	resVal := reflect.MakeSlice(sVal.Type(), len(keys), len(keys))
	for i, iOld := range keys {
		resVal.Index(i).Set(mapView.MapIndex(reflect.ValueOf(iOld)))
	}

	return resVal.Interface(), poppedVal.Interface()
}

//func PopFunc(slice []string, f func(i string) bool) []string {
//	mapView := make(map[int]string)
//	keys := make([]int, 0, len(slice))
//	for i, item := range slice {
//		if f(item) {
//			mapView[i] = item
//			keys = append(keys, i)
//		}
//	}
//	// keys are already sorted
//
//	res := make([]string, len(keys))
//	for i, iOld := range keys {
//		res[i] = mapView[iOld]
//	}
//
//	return res
//}


// ForEach is special function, when you just know, that some variable is slice. try to not use this func
// instead, for _,_ := range _ is WAY MORE preferrable. This func is only is useful for rarest situtations
func ForEach(slice any, f func(index int, i any)) {
	ival := reflect.ValueOf(slice)
	if ival.Type().Kind() != reflect.Slice {
		panic("not a slice: " + ival.Type().String())
	}

	for i := 0; i < ival.Len(); i++ {
		f(i, ival.Index(i).Interface())
	}
}

//func ForEach(slice []string, f func(index int, i string)) {
//	for i, item := range slice {
//		f(i, item)
//	}
//}
