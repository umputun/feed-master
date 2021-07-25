// Copyright (c) 2020 Xelaj Software
//
// This file is a part of go-dry package.
// See https://github.com/xelaj/go-dry/blob/master/LICENSE for details

// +build windows

package dry

func pathIsWritable(path string) bool {
	// currently it's too hard to implement windows checking (i really want to! but i don't have any windows
	// machine to test). We found this package github.com/hectane/go-acl, so we can implement this feature
	// using advapi32.dll.
	// TODO: implement it
	return true
}
