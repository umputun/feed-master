// Copyright (c) 2020 Xelaj Software
//
// This file is a part of go-dry package.
// See https://github.com/xelaj/go-dry/blob/master/LICENSE for details

// +build test

package dry

var _ = func() bool {
	testMode = true
	return true
}()
