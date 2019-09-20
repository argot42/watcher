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
	/* attempt to replicate a part of tail.c shipped with plan9 */

	defer file.Close()
	defer close(sub.Out)

	var fi0, fi1 *os.FileInfo

	for {
		// check if done channel is closed
		select {
		case <-sub.Done:
			fmt.Printf("stop watching [%s]\n", file.Name())
			return
		default:
		}

		err := trunc(fi0, &fi1, file)
		if err != nil {
			sub.Err <- err
			return
		}
		err = write(file, sub.Out)
		if err != nil {
			sub.Err <- err
			return
		}
		err = trunc(fi1, &fi0, file)
		if err != nil {
			sub.Err <- err
			return
		}
		time.Sleep(SLEEP * time.Millisecond)
	}
}

func trunc(old *os.FileInfo, new **os.FileInfo, file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}

	var length int64 = 0

	if old != nil {
		length = (*old).Size()
	}
	if info.Size() < length {
		_, err := file.Seek(0, 0)
		if err != nil {
			return err
		}
	}
	newInfo, err := file.Stat()
	if err != nil {
		return err
	}
	*new = &newInfo

	return nil
}

func write(r io.Reader, out chan<- byte) error {
	data := make([]byte, BSIZE)

	for {
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
