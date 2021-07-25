// Since in Go 1.16, stdlib ioutil is deprecated, this package named exactly as replacer to this package

package ioutil

import (
	"io"
)

// ReaderFunc implements io.Reader as function type with a Read method.
type ReaderFunc func(p []byte) (int, error)

func (f ReaderFunc) Read(p []byte) (int, error) {
	return f(p)
}

// WriterFunc implements io.Writer as function type with a Write method.
type WriterFunc func(p []byte) (int, error)

func (f WriterFunc) Write(p []byte) (int, error) {
	return f(p)
}

// WriteFull calls writer.Write until all of data is written,
// or an is error returned.
func WriteAll(data []byte, writer io.Writer) (n int, err error) {
	dataSize := len(data)
	for n = 0; n < dataSize; {
		m, err := writer.Write(data[n:])
		n += m
		if err != nil {
			return n, err
		}
	}
	return dataSize, nil
}
