// Copyright (c) 2020 Xelaj Software
//
// This file is a part of go-dry package.
// See https://github.com/xelaj/go-dry/blob/master/LICENSE for details

package dry

import (
	"reflect"

	"github.com/xelaj/go-dry/slices"
)

// индекс элемента T в []T
// выводит индекс найденого элемента, либо -1, если элемент не найден
func SliceIndex(slice, item any) int {
	return slices.Index(slice, item)
}

func SliceContains(slice, item any) bool {
	return slices.Contains(slice, item)
}

func SliceDeleteIndex(slice any, i int) any {
	return slices.DeleteIndex(slice, i)
}

func SliceCut(slice any, i, j int) any {
	return slices.Cut(slice, i, j)
}

func SliceRemoveFunc(slice any, f func(i any) bool) any {
	return slices.RemoveFunc(slice, f)
}

// if f func returns true, than slice item will pop
func SlicePopFunc(slice any, f func(i any) bool) (res, popped any) {
	return slices.PopFunc(slice, f)
}

func SliceExpand(slice any, i, j int) any {
	return slices.Expand(slice, i, j)
}

// SliceToInterfaceSlice converts a slice of any type into a slice of interface{}.
func SliceToInterfaceSlice(in any) []any {
	return slices.ToInterfaceSlice(in)
}

// []<T> -> map[<T>]struct{}
func SliceUnique(in any) any {
	return slices.Unique(in)
}

// SliceForEach is special function, when you just know, that some variable is slice. try to not use this func
// instead, for _,_ := range _ is WAY MORE preferrable. This func is only is useful for rarest situtations
func SliceForEach(slice any, f func(index int, i any)) {
	slices.ForEach(slice, f)
}

// map[<K>]<V> -> []<K> // (K, V) could be any type
func MapKeys(in any) any {
	ival := reflect.ValueOf(in)
	if ival.Type().Kind() != reflect.Map {
		panic("not a map: " + ival.Type().String())
	}

	keys := ival.MapKeys()

	items := reflect.MakeSlice(reflect.SliceOf(ival.Type().Key()), len(keys), len(keys))
	for i, key := range keys {
		items.Index(i).Set(key)
	}

	return items.Interface()
}
