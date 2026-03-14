package bbs

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxFileSize     = 50 * 1024 // 50KB max file size
	chunkDataSize   = 160       // raw bytes per chunk (leaves room for header in 228 byte limit)
	textChunkSize   = 200       // chars per plain text chunk
	filePrefix      = "FILE"    // protocol prefix for binary chunks
)

func filesDir() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "meshtastic-cli", "bbs-files")
	os.MkdirAll(dir, 0755)
	return dir
}

func (b *BBS) cmdFiles(args []string) string {
	if len(args) == 0 {
		return "Usage: files list | files get <name>"
	}

	switch strings.ToLower(args[0]) {
	case "list":
		return b.filesList()
	case "get":
		if len(args) < 2 {
			return "Usage: files get <filename>"
		}
		return "" // handled by filesSend which returns multiple messages
	default:
		return "Usage: files list | files get <name>"
	}
}

func (b *BBS) filesList() string {
	dir := filesDir()
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		return "No files available"
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Files (%d):", len(entries)))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		size := info.Size()
		sizeStr := fmt.Sprintf("%dB", size)
		if size > 1024 {
			sizeStr = fmt.Sprintf("%.1fKB", float64(size)/1024)
		}
		lines = append(lines, fmt.Sprintf("  %s (%s)", e.Name(), sizeStr))
	}

	result := strings.Join(lines, "\n")
	if len(result) > 220 {
		result = result[:220] + "..."
	}
	return result
}

// FilesGet returns the chunks to send for a file request.
// Text files (.txt, .md) are sent as plain text chunks.
// Other files are sent as base64-encoded chunks with FILE: protocol headers.
func (b *BBS) FilesGet(filename string) ([]string, error) {
	dir := filesDir()

	// Sanitize filename — no path traversal
	filename = filepath.Base(filename)
	path := filepath.Join(dir, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s", filename)
	}

	if len(data) > maxFileSize {
		return nil, fmt.Errorf("file too large (%d bytes, max %d)", len(data), maxFileSize)
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".txt" || ext == ".md" || ext == ".text" {
		return chunkText(filename, string(data)), nil
	}

	return chunkBinary(filename, data), nil
}

func chunkText(filename string, text string) []string {
	var chunks []string

	// Split into lines first, then group into chunks
	lines := strings.Split(text, "\n")
	var current string

	for _, line := range lines {
		if len(current)+len(line)+1 > textChunkSize && current != "" {
			chunks = append(chunks, current)
			current = ""
		}
		if current != "" {
			current += "\n"
		}
		current += line
	}
	if current != "" {
		chunks = append(chunks, current)
	}

	// Add header to first chunk and footer to last
	if len(chunks) > 0 {
		header := fmt.Sprintf("--- %s [1/%d] ---\n", filename, len(chunks))
		chunks[0] = header + chunks[0]
	}
	if len(chunks) > 1 {
		for i := 1; i < len(chunks); i++ {
			header := fmt.Sprintf("--- %s [%d/%d] ---\n", filename, i+1, len(chunks))
			chunks[i] = header + chunks[i]
		}
	}

	return chunks
}

func chunkBinary(filename string, data []byte) []string {
	encoded := base64.StdEncoding.EncodeToString(data)

	// Calculate chunk count
	// Each chunk payload: FILE:<filename>:<n>/<total>:<base64>
	// We need to figure out how much base64 fits per chunk
	totalChunks := (len(encoded) + chunkDataSize - 1) / chunkDataSize

	var chunks []string
	for i := 0; i < totalChunks; i++ {
		start := i * chunkDataSize
		end := start + chunkDataSize
		if end > len(encoded) {
			end = len(encoded)
		}

		chunk := fmt.Sprintf("%s:%s:%d/%d:%s",
			filePrefix, filename, i+1, totalChunks, encoded[start:end])
		chunks = append(chunks, chunk)
	}

	return chunks
}

// ParseFileChunk checks if a message is a binary file chunk and extracts its parts.
// Returns filename, chunkNum, totalChunks, base64Data, ok.
func ParseFileChunk(msg string) (filename string, chunkNum, totalChunks int, data string, ok bool) {
	if !strings.HasPrefix(msg, filePrefix+":") {
		return "", 0, 0, "", false
	}

	parts := strings.SplitN(msg, ":", 4)
	if len(parts) != 4 {
		return "", 0, 0, "", false
	}

	filename = parts[1]

	var n, total int
	_, err := fmt.Sscanf(parts[2], "%d/%d", &n, &total)
	if err != nil || n < 1 || total < 1 {
		return "", 0, 0, "", false
	}

	return filename, n, total, parts[3], true
}
