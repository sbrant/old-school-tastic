package pcap

import (
	"encoding/binary"
	"os"
	"sync"
	"time"
)

// PCAP global header (24 bytes)
// Link type 147 = USER0 (user-defined)
var globalHeader = []byte{
	0xd4, 0xc3, 0xb2, 0xa1, // magic
	0x02, 0x00, 0x04, 0x00, // version 2.4
	0x00, 0x00, 0x00, 0x00, // thiszone
	0x00, 0x00, 0x00, 0x00, // sigfigs
	0x00, 0x02, 0x00, 0x00, // snaplen 512
	0x93, 0x00, 0x00, 0x00, // link type 147 (USER0)
}

type Writer struct {
	mu   sync.Mutex
	file *os.File
}

func NewWriter(path string) (*Writer, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	if _, err := f.Write(globalHeader); err != nil {
		f.Close()
		return nil, err
	}

	return &Writer{file: f}, nil
}

func (w *Writer) WritePacket(data []byte, ts time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Packet header (16 bytes)
	var hdr [16]byte
	binary.LittleEndian.PutUint32(hdr[0:4], uint32(ts.Unix()))
	binary.LittleEndian.PutUint32(hdr[4:8], uint32(ts.Nanosecond()/1000))
	binary.LittleEndian.PutUint32(hdr[8:12], uint32(len(data)))
	binary.LittleEndian.PutUint32(hdr[12:16], uint32(len(data)))

	if _, err := w.file.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.file.Write(data)
	return err
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}
