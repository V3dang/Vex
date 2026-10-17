package wal

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"sync"
)

// Each record: [type 1 byte][length 4 bytes][payload N bytes]
const (
	OpInsert byte = 0x01
	OpDelete byte = 0x02
)

type Record struct {
	Op      byte
	ID      string
	Vector  []float32
	Payload []byte
}

type WAL struct {
	f  *os.File
	mu sync.Mutex
}

func Open(path string) (*WAL, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	return &WAL{f: f}, nil
}

func (w *WAL) Write(r Record) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := json.Marshal(r)
	if err != nil {
		return err
	}

	// [op][len uint32][data]
	buf := make([]byte, 5+len(data))
	buf[0] = r.Op
	binary.LittleEndian.PutUint32(buf[1:5], uint32(len(data)))
	copy(buf[5:], data)

	_, err = w.f.Write(buf)
	return err
}

// Replay reads all records from the WAL.
// Call this on startup to reconstruct the index.
func (w *WAL) Replay() ([]Record, error) {
	if _, err := w.f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	reader := bufio.NewReader(w.f)
	var records []Record

	for {
		header := make([]byte, 5)
		if _, err := io.ReadFull(reader, header); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		op := header[0]
		n := binary.LittleEndian.Uint32(header[1:5])
		data := make([]byte, n)
		if _, err := io.ReadFull(reader, data); err != nil {
			return nil, err
		}

		var r Record
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, err
		}
		r.Op = op
		records = append(records, r)
	}
	return records, nil
}

func (w *WAL) Close() error { return w.f.Close() }
