package bbs

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type pendingFile struct {
	filename    string
	totalChunks int
	chunks      map[int]string
	lastUpdate  time.Time
}

type FileClient struct {
	pending  map[string]*pendingFile // keyed by filename
	saveDir  string
	received []FileReceived
	maxLog   int
}

type FileReceived struct {
	Timestamp time.Time
	Filename  string
	Size      int
	FromNode  uint32
	Complete  bool
	Progress  string // e.g. "3/5"
}

func NewFileClient(saveDir string) *FileClient {
	os.MkdirAll(saveDir, 0755)
	return &FileClient{
		pending: make(map[string]*pendingFile),
		saveDir: saveDir,
		maxLog:  100,
	}
}

func DefaultSaveDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "meshtastic-cli", "received-files")
}

func (fc *FileClient) Log() []FileReceived {
	return fc.received
}

// HandleMessage checks if an incoming message is a file chunk.
// Returns true if the message was a file chunk (so it can be suppressed from chat).
func (fc *FileClient) HandleMessage(fromNode uint32, text string) bool {
	filename, chunkNum, totalChunks, data, ok := ParseFileChunk(text)
	if !ok {
		return false
	}

	pf, exists := fc.pending[filename]
	if !exists {
		pf = &pendingFile{
			filename:    filename,
			totalChunks: totalChunks,
			chunks:      make(map[int]string),
		}
		fc.pending[filename] = pf
	}

	pf.chunks[chunkNum] = data
	pf.lastUpdate = time.Now()

	progress := fmt.Sprintf("%d/%d", len(pf.chunks), pf.totalChunks)

	fc.addLog(FileReceived{
		Timestamp: time.Now(),
		Filename:  filename,
		FromNode:  fromNode,
		Complete:  false,
		Progress:  progress,
	})

	// Check if complete
	if len(pf.chunks) >= pf.totalChunks {
		fc.assembleFile(pf, fromNode)
		delete(fc.pending, filename)
	}

	return true
}

func (fc *FileClient) assembleFile(pf *pendingFile, fromNode uint32) {
	// Concatenate base64 chunks in order
	var encoded string
	for i := 1; i <= pf.totalChunks; i++ {
		encoded += pf.chunks[i]
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		fc.addLog(FileReceived{
			Timestamp: time.Now(),
			Filename:  pf.filename,
			FromNode:  fromNode,
			Complete:  false,
			Progress:  "decode error",
		})
		return
	}

	path := filepath.Join(fc.saveDir, pf.filename)
	if err := os.WriteFile(path, data, 0644); err != nil {
		fc.addLog(FileReceived{
			Timestamp: time.Now(),
			Filename:  pf.filename,
			FromNode:  fromNode,
			Complete:  false,
			Progress:  "write error",
		})
		return
	}

	fc.addLog(FileReceived{
		Timestamp: time.Now(),
		Filename:  pf.filename,
		Size:      len(data),
		FromNode:  fromNode,
		Complete:  true,
		Progress:  fmt.Sprintf("saved (%d bytes)", len(data)),
	})
}

// CleanStale removes pending files that haven't received a chunk in 5 minutes
func (fc *FileClient) CleanStale() {
	cutoff := time.Now().Add(-5 * time.Minute)
	for name, pf := range fc.pending {
		if pf.lastUpdate.Before(cutoff) {
			delete(fc.pending, name)
		}
	}
}

func (fc *FileClient) addLog(entry FileReceived) {
	// Update existing entry for same filename or append
	for i := len(fc.received) - 1; i >= 0; i-- {
		if fc.received[i].Filename == entry.Filename && !fc.received[i].Complete {
			fc.received[i] = entry
			return
		}
	}
	fc.received = append(fc.received, entry)
	if len(fc.received) > fc.maxLog {
		fc.received = fc.received[len(fc.received)-fc.maxLog:]
	}
}
