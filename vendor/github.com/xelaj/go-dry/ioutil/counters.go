package ioutil

import "io"

type Counter interface {
	Count() int
}

type ReadCounter interface {
	io.Reader
	Counter
}

type countingReader struct {
	reader    io.Reader
	bytesRead int
}

func CountingReader(r io.Reader) ReadCounter {
	return &countingReader{reader: r}
}

func (r *countingReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	r.bytesRead += n
	return n, err
}

func (r *countingReader) Count() int {
	return r.bytesRead
}

type WriteCounter interface {
	io.Writer
	Counter
}

type countingWriter struct {
	writer       io.Writer
	bytesWritten int
}

func CountingWriter(w io.Writer) WriteCounter {
	return &countingWriter{writer: w}
}

func (w *countingWriter) Write(p []byte) (n int, err error) {
	n, err = w.writer.Write(p)
	w.bytesWritten += n
	return n, err
}

func (w *countingWriter) Count() int {
	return w.bytesWritten
}
