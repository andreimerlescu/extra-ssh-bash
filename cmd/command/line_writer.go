package command

import (
	"fmt"
	"strings"
	"sync"
)

type LineWriter struct {
	lines  chan string
	buffer []byte
	done   chan bool
	errCh  chan error
	err    error
	wg     sync.WaitGroup
}

func NewLineWriter(size int) *LineWriter {
	return &LineWriter{
		lines: make(chan string, size),
		done:  make(chan bool),
		errCh: make(chan error, 1),
	}
}

func (lw *LineWriter) Write(p []byte) (n int, err error) {
	lw.buffer = append(lw.buffer, p...)
	start := 0
	for i := 0; i < len(lw.buffer); i++ {
		if lw.buffer[i] == '\n' {
			line := string(lw.buffer[start:i])
			if strings.Contains(line, "was forcibly closed by the remote host") {
				lw.errCh <- fmt.Errorf("error encountered: %s", line)
			} else {
				lw.lines <- line
			}
			start = i + 1
		}
	}
	lw.buffer = lw.buffer[start:]
	return len(p), nil
}

func (lw *LineWriter) Close() {
	lw.wg.Wait()
	close(lw.lines)
	close(lw.errCh)
	lw.done <- true
	close(lw.done)
}

func (lw *LineWriter) Lines() <-chan string {
	return lw.lines
}

func (lw *LineWriter) ReadLine() (string, error) {
	select {
	case line, ok := <-lw.lines:
		if !ok {
			return "", fmt.Errorf("LineWriter is closed")
		}
		return line, nil
	case <-lw.done:
		return "", fmt.Errorf("LineWriter is closed")
	}
}

func (lw *LineWriter) Err() error {
	if lw.err == nil {
		select {
		case err, ok := <-lw.errCh:
			if ok {
				lw.err = err
			}
		default:
		}
	}
	return lw.err
}
