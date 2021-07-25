// Copyright (c) 2020 Xelaj Software
//
// This file is a part of go-dry package.
// See https://github.com/xelaj/go-dry/blob/master/LICENSE for details

package dry

import (
	"fmt"
	"path/filepath"
	"strings"
)

func PathWithoutExt(filename string) string {
	ext := filepath.Ext(filename)
	return strings.TrimSuffix(filename, ext)
}

// делить название файла на само название и расширение
// при этом на название не влияет путь, в котором расположен файл
func PathSplitExt(path string) (basepath, ext string) {
	filename := filepath.Base(path)
	if filename == "." {
		return "", ""
	}

	hidden := false
	if strings.HasPrefix(filename, ".") {
		hidden = true
		filename = strings.TrimPrefix(filename, ".")
	}

	ext = filepath.Ext(filename)
	basepath = filename[:len(filename)-len(ext)]
	if hidden {
		basepath = "." + basepath
	}
	ext = strings.TrimPrefix(ext, ".")

	return
}

func PathIsWritable(path string) bool {
	return pathIsWritable(path)
}

func PathNearestExisting(path string) string {
	if FileExists(path) {
		return path
	}

	testpath := path
	for testpath != "" {
		testpath, _ = filepath.Split(testpath)
		testpath = testpath[:len(testpath)-1] // removing trailing /
		if FileExists(testpath) {
			return testpath
		}
		fmt.Println(testpath)
	}

	return ""
}
