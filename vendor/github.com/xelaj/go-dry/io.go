// Copyright (c) 2020 Xelaj Software
//
// This file is a part of go-dry package.
// See https://github.com/xelaj/go-dry/blob/master/LICENSE for details

package dry

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	dry_ioutil "github.com/xelaj/go-dry/ioutil"
)

type ReadCounter = dry_ioutil.ReadCounter

func CountingReader(r io.Reader) ReadCounter {
	return dry_ioutil.CountingReader(r)
}

type WriteCounter = dry_ioutil.WriteCounter

func CountingWriter(w io.Writer) WriteCounter {
	return dry_ioutil.CountingWriter(w)
}

// WriteFull calls writer.Write until all of data is written,
// or an is error returned.
func WriteFull(data []byte, writer io.Writer) (n int, err error) {
	return dry_ioutil.WriteAll(data, writer)
}

// ReaderFunc implements io.Reader as function type with a Read method.
type ReaderFunc = dry_ioutil.ReaderFunc

// WriterFunc implements io.Writer as function type with a Write method.
type WriterFunc = dry_ioutil.WriterFunc

// CancelableReader позволяет читать данные с контекстом
type CancelableReader = dry_ioutil.CancelableReader

func NewCancelableReader(ctx context.Context, r io.Reader) *CancelableReader {
	return dry_ioutil.NewCancelableReader(ctx, r)
}

// methods below are deprecated

// TODO: принцип интересный, но не входит в рамки io
type CountingReadWriter struct {
	ReadWriter   io.ReadWriter
	BytesRead    int
	BytesWritten int
}

func (rw *CountingReadWriter) Read(p []byte) (n int, err error) {
	n, err = rw.ReadWriter.Read(p)
	rw.BytesRead += n
	return n, err
}

func (rw *CountingReadWriter) Write(p []byte) (n int, err error) {
	n, err = rw.ReadWriter.Write(p)
	rw.BytesWritten += n
	return n, err
}

//! DEPRECATED
// ReadBinary wraps binary.Read with a CountingReader and returns
// the acutal bytes read by it.
func ReadBinary(r io.Reader, order binary.ByteOrder, data any) (n int, err error) {
	countingReader := CountingReader(r)
	err = binary.Read(countingReader, order, data)
	return countingReader.Count(), err
}

//! DEPRECATED
// ReadLine reads unbuffered until a newline '\n' byte and removes
// an optional carriege return '\r' at the end of the line.
// In case of an error, the string up to the error is returned.
func ReadLine(reader io.Reader) (line string, err error) {
	buffer := bytes.NewBuffer(make([]byte, 0, 4096))
	p := make([]byte, 1)
	for {
		var n int
		n, err = reader.Read(p)
		if err != nil || p[0] == '\n' {
			break
		}
		if n > 0 {
			buffer.WriteByte(p[0])
		}
	}
	data := buffer.Bytes()
	if len(data) > 0 && data[len(data)-1] == '\r' {
		data = data[:len(data)-1]
	}
	return string(data), err
}

//! DEPRECATED
// WaitForStdin blocks until input is available from os.Stdin.
// The first byte from os.Stdin is returned as result.
// If there are println arguments, then fmt.Println will be
// called with those before reading from os.Stdin.
func WaitForStdin(v ...any) byte {
	if len(v) > 0 {
		fmt.Println(v...)
	}
	buffer := make([]byte, 1)
	_, _ = os.Stdin.Read(buffer)
	return buffer[0]
}
