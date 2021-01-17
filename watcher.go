package watcher

import (
	"fmt"
	"io"
	"os"
	"time"
    "strconv"
)

var (
	BSIZE    = 2048
	CHANSIZE = 100
	SLEEP    = 5000
)

func init() {
    bsizeStr := os.Getenv("BSIZE")
    if len(bsizeStr) > 0 {
        bsize, err := strconv.Atoi(bsizeStr)
        if err != nil {
            fmt.Fprintf(os.Stderr, "BSIZE env var err: %s\n", err)
            os.Exit(1)
        }

        BSIZE = bsize
    }

    chansizeStr := os.Getenv("CHANSIZE")
    if len(chansizeStr) > 0 {
        chansize, err := strconv.Atoi(chansizeStr)
        if err != nil {
            fmt.Fprintf(os.Stderr, "CHANSIZE env var err: %s\n", err)
            os.Exit(1)
        }

        CHANSIZE = chansize
    }

    sleepStr := os.Getenv("SLEEP")
    if len(sleepStr) > 0 {
        sleep, err := strconv.Atoi(sleepStr)
        if err != nil {
            fmt.Fprintf(os.Stderr, "SLEEP env var err: %s\n", err)
            os.Exit(1)
        }

        SLEEP = sleep
    }
}

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

func Watch(filepath string) (W) {
	w := W{
		make(chan bool, CHANSIZE),
		make(chan error),
		make(chan bool),
	}

	go WatchSubscribe(filepath, w, SLEEP)
	return w
}

func WatchSubscribe(filepath string, w W, sleep int) {
	var lastChange time.Time
	var changed bool
	var err error

	defer close(w.Out)

    for {
		select {
		case <-w.Done:
			fmt.Printf("stop watching [%s]\n", filepath)
			return
		case <-time.After(time.Duration(sleep) * time.Millisecond):
		}

        changed, lastChange, err = change(filepath, lastChange)
		if err != nil {
			w.Err <- err
			return
		}
		if changed {
			w.Out <- changed
		}
    }
}

func Read(filepath string) (R) {
	r := R{
		make(chan ROut, CHANSIZE),
		make(chan error),
		make(chan bool),
	}

	go ReadSubscribe(filepath, r, SLEEP)
	return r
}

func ReadSubscribe(filepath string, r R, sleep int) {
	var lastChange time.Time
	var changed bool
	var err error

	defer close(r.Out)

	for {
		select {
		case <-r.Done:
			fmt.Printf("stop watching [%s]\n", filepath)
			return
		case <-time.After(time.Duration(sleep) * time.Millisecond):
		}

		changed, lastChange, err = change(filepath, lastChange)
		if err != nil {
			r.Err <- err
			return
		}
		if changed {
			if err = send(filepath, r.Out, BSIZE); err != nil {
				r.Err <- err
				return
			}
		}
	}
}

func send(filepath string, rout chan<- ROut, bsize int) error {
	data := make([]byte, bsize)
	first := true

    r, err := os.Open(filepath)
    if err != nil {
        return err
    }
    defer r.Close()

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
		    first = false
		}
		if err == io.EOF {
			return nil
		}
	}
}

func change(filepath string, lastChange time.Time) (bool, time.Time, error) {
	info, err := os.Stat(filepath)
	if err != nil {
		return false, time.Now(), err
	}

	return lastChange.Before(info.ModTime()), info.ModTime(), nil
}
