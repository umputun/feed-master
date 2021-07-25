package ioutil

import (
	"context"
	"fmt"
	"io"
	"strconv"
)

// CancelableReader позволяет читать данные с контекстом
type CancelableReader struct {
	ctx  context.Context
	data chan []byte

	// размер сообщения, которое мы хотим получить. пока в sizeWant не пошлется длина, ридер не будет читать
	sizeWant chan int

	err error
	r   io.Reader
}

func (c *CancelableReader) begin() {
	for {
		select {
		case sizeWant := <-c.sizeWant:
			buf := make([]byte, sizeWant)
			// readFull, cause some readers like net.TCPConn returns size smaller than buf size
			n, err := io.ReadFull(c.r, buf)
			if err != nil {
				c.err = err
				close(c.data)
				return
			}
			if n != sizeWant {
				panic("read " + strconv.Itoa(n) + ", want " + strconv.Itoa(sizeWant))
			}
			c.data <- buf
		case <-c.ctx.Done():
			close(c.data)
			close(c.sizeWant)
			return
		}
	}
}

func (c *CancelableReader) Read(p []byte) (int, error) {
	c.sizeWant <- len(p)
	select {
	case <-c.ctx.Done():
		return 0, c.ctx.Err()
	case d, ok := <-c.data:
		if !ok {
			return 0, c.err
		}
		copy(p, d)
		return len(d), nil
	}
}

func (c *CancelableReader) ReadByte() (byte, error) {
	b := make([]byte, 1)

	n, err := c.Read(b)
	if err != nil {
		return 0x0, err
	}
	if n != 1 {
		panic(fmt.Errorf("read more than 1 byte, got %v", n))
	}

	return b[0], nil
}

func NewCancelableReader(ctx context.Context, r io.Reader) *CancelableReader {
	c := &CancelableReader{
		r:        r,
		ctx:      ctx,
		data:     make(chan []byte),
		sizeWant: make(chan int),
	}
	go c.begin()
	return c
}
