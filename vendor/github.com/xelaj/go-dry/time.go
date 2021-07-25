// Copyright (c) 2020 Xelaj Software
//
// This file is a part of go-dry package.
// See https://github.com/xelaj/go-dry/blob/master/LICENSE for details

package dry

import (
	"time"

	"github.com/xelaj/go-dry/timeutil"
)

func RangeDate(start, end time.Time) []time.Time {
	return timeutil.RangeDate(start, end)
}
