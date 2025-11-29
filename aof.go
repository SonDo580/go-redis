package main

import (
	"io"
	"os"
	"sync"
	"time"
)

type Aof struct {
	file *os.File // append-only file to store RESP commands
	mu   sync.Mutex
}

func NewAof(path string) (*Aof, error) {
	// Open the file for reading and writing, create if it doesn't exist
	// (permission 0666: all users can read and write but cannot execute)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	aof := &Aof{file: file}

	// Start a goroutine to flush the AOF to disk every 1 second
	go func() {
		for {
			aof.mu.Lock() // prevent concurrent writes
			aof.file.Sync()
			aof.mu.Unlock()
			time.Sleep(time.Second)
		}
	}()

	return aof, nil
}

// Close the AOF
func (aof *Aof) Close() error {
	aof.mu.Lock()
	defer aof.mu.Unlock()

	return aof.file.Close()
}

// Serialize RESP command and write to the AOF
func (aof *Aof) Write(value Value) error {
	aof.mu.Lock()
	defer aof.mu.Unlock()

	_, err := aof.file.Write(value.Marshal())
	if err != nil {
		return err
	}

	return nil
}

// Read the AOF, deserialize RESP commands, invoke provided callback for each command
// (assume the file is valid, in real-world scenario it could be corrupted)
func (aof *Aof) Read(callback func(value Value)) error {
	aof.mu.Lock()
	defer aof.mu.Unlock()

	resp := NewResp(aof.file)

	for {
		value, err := resp.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		callback(value)
	}

	return nil
}
