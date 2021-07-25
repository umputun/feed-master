package dry

import (
	"path/filepath"
	"runtime"
)

var testMode bool

func IsTestMode() bool {
	return testMode
}

func TestGetCurrentPackagePath() string {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("No caller information")
	}
	return filepath.Dir(filename)
}
