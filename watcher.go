package watcher

import (
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

type R struct {
	Out  chan ROut
	Err  chan error
	Done chan bool
}

type ROut struct {
	Data  byte
	First bool
}

type W struct {
	Out  chan bool
	Err  chan error
	Done chan bool
}

func Watch(filepath string) (W, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return W{}, err
	}

	w := W{
		make(chan bool, CHANSIZE),
		make(chan error),
		make(chan bool),
	}

	go WatchSubscribe(file, w, SLEEP)
	return w, nil
}

func WatchSubscribe(file *os.File, w W, sleep int) {
	var lastChange time.Time
	var changed bool
	var err error

	defer file.Close()
	defer close(w.Out)

	for {
		select {
		case <-w.Done:
			fmt.Printf("stop watching [%s]\n", file.Name())
			return
		case <-time.After(time.Duration(sleep) * time.Millisecond):
		}

		changed, lastChange, err = change(file, lastChange)
		if err != nil {
			w.Err <- err
			return
		}
		if changed {
			w.Out <- changed
		}
	}
}

func Read(filepath string) (R, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return R{}, err
	}

	r := R{
		make(chan ROut, CHANSIZE),
		make(chan error),
		make(chan bool),
	}

	go ReadSubscribe(file, r, SLEEP)
	return r, nil
}

func ReadSubscribe(file *os.File, r R, sleep int) {
	var lastChange time.Time
	var changed bool
	var err error

	defer file.Close()
	defer close(r.Out)

	for {
		select {
		case <-r.Done:
			fmt.Printf("stop watching [%s]\n", file.Name())
			return
		case <-time.After(time.Duration(sleep) * time.Millisecond):
		}

		changed, lastChange, err = change(file, lastChange)
		if err != nil {
			r.Err <- err
			return
		}
		if changed {
			if err = send(file, r.Out, BSIZE); err != nil {
				r.Err <- err
				return
			}
		}
	}
}

func send(r io.Reader, rout chan<- ROut, bsize int) error {
	data := make([]byte, bsize)
	first := true

	for {
		// read from file
		n, err := r.Read(data)
		if err != nil && err != io.EOF {
			return err
		}
		// send info
		for _, b := range data[:n] {
			rout <- ROut{
				b,
				first,
			}
		}
		if err == io.EOF {
			return nil
		}
		first = false
	}
}

func change(file *os.File, lastChange time.Time) (bool, time.Time, error) {
	info, err := file.Stat()
	if err != nil {
		return false, time.Now(), err
	}

	return lastChange.Before(info.ModTime()), info.ModTime(), nil
}
