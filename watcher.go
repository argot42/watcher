package watcher

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	BSIZE    = 2048
	CHANSIZE = 100
	SLEEP    = 5000
)

var Truncated error = errors.New("file truncated")

type Sub struct {
	Out  chan byte
	Err  chan error
	Done chan struct{}
}

func Watch(filepath string) (Sub, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return Sub{}, err
	}

	s := Sub{
		make(chan byte, CHANSIZE),
		make(chan error),
		make(chan struct{}),
	}

	go Subscribe(file, s)
	return s, nil
}

func Subscribe(file *os.File, sub Sub) {
	var lastChange time.Time

	defer file.Close()
	defer close(sub.Out)

	for {
		// check if done channel is closed
		select {
		case <-sub.Done:
			fmt.Printf("stop watching [%s]\n", file.Name())
			return
		case <-time.After(SLEEP * time.Millisecond):
		}

		// check if file changed
		var changed bool
		var err error

		changed, lastChange, err = change(file, lastChange)
		if err != nil {
			sub.Err <- err
			return
		}
		if changed {
			err = write(file, sub.Out)
			if err != nil {
				sub.Err <- err
				return
			}
		}
	}
}

func change(file *os.File, lastChange time.Time) (bool, time.Time, error) {
	info, err := file.Stat()
	if err != nil {
		return false, time.Now(), err
	}
	return lastChange.Before(info.ModTime()), info.ModTime(), nil
}

func write(r io.Reader, out chan<- byte) error {
	data := make([]byte, BSIZE)

	for {
		// read from file
		n, err := r.Read(data)
		if err != nil && err != io.EOF {
			return err
		}
		// send info
		for _, b := range data[:n] {
			out <- b
		}
		if err == io.EOF {
			return nil
		}
	}
}
