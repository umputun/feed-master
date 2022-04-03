// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mocks

import (
	"sync"

	"github.com/umputun/feed-master/app/feed"
)

// StoreMock is a mock implementation of api.Store.
//
// 	func TestSomethingThatUsesStore(t *testing.T) {
//
// 		// make and configure a mocked api.Store
// 		mockedStore := &StoreMock{
// 			LoadFunc: func(fmFeed string, max int, skipJunk bool) ([]feed.Item, error) {
// 				panic("mock out the Load method")
// 			},
// 		}
//
// 		// use mockedStore in code that requires api.Store
// 		// and then make assertions.
//
// 	}
type StoreMock struct {
	// LoadFunc mocks the Load method.
	LoadFunc func(fmFeed string, max int, skipJunk bool) ([]feed.Item, error)

	// calls tracks calls to the methods.
	calls struct {
		// Load holds details about calls to the Load method.
		Load []struct {
			// FmFeed is the fmFeed argument value.
			FmFeed string
			// Max is the max argument value.
			Max int
			// SkipJunk is the skipJunk argument value.
			SkipJunk bool
		}
	}
	lockLoad sync.RWMutex
}

// Load calls LoadFunc.
func (mock *StoreMock) Load(fmFeed string, max int, skipJunk bool) ([]feed.Item, error) {
	if mock.LoadFunc == nil {
		panic("StoreMock.LoadFunc: method is nil but Store.Load was just called")
	}
	callInfo := struct {
		FmFeed   string
		Max      int
		SkipJunk bool
	}{
		FmFeed:   fmFeed,
		Max:      max,
		SkipJunk: skipJunk,
	}
	mock.lockLoad.Lock()
	mock.calls.Load = append(mock.calls.Load, callInfo)
	mock.lockLoad.Unlock()
	return mock.LoadFunc(fmFeed, max, skipJunk)
}

// LoadCalls gets all the calls that were made to Load.
// Check the length with:
//     len(mockedStore.LoadCalls())
func (mock *StoreMock) LoadCalls() []struct {
	FmFeed   string
	Max      int
	SkipJunk bool
} {
	var calls []struct {
		FmFeed   string
		Max      int
		SkipJunk bool
	}
	mock.lockLoad.RLock()
	calls = mock.calls.Load
	mock.lockLoad.RUnlock()
	return calls
}