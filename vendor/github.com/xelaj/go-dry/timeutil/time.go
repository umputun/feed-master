// Copyright (c) 2020 Xelaj Software
//
// This file is a part of go-dry package.
// See https://github.com/xelaj/go-dry/blob/master/LICENSE for details

package timeutil

import "time"

func RangeDate(start, end time.Time) []time.Time {
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())

	res := make([]time.Time, 0)

	// end.AddDate(0,0,1), это потому что мы ранжируем включительно
	for curr := start; curr.Before(end.AddDate(0, 0, 1)); curr = curr.AddDate(0, 0, 1) {
		res = append(res, curr)
	}

	return res
}
