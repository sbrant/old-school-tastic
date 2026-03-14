package serial

const (
	start1       = 0x94
	start2       = 0xc3
	maxFrameSize = 512
)

type parseState int

const (
	waitStart1 parseState = iota
	waitStart2
	waitMSB
	waitLSB
	waitPayload
)

type Framer struct {
	state  parseState
	msb    byte
	length int
	buf    [maxFrameSize]byte
	pos    int
}

// Feed processes raw serial bytes and returns any complete frames found.
func (f *Framer) Feed(data []byte) [][]byte {
	var frames [][]byte
	for _, b := range data {
		switch f.state {
		case waitStart1:
			if b == start1 {
				f.state = waitStart2
			}
		case waitStart2:
			if b == start2 {
				f.state = waitMSB
			} else {
				f.state = waitStart1
			}
		case waitMSB:
			f.msb = b
			f.state = waitLSB
		case waitLSB:
			f.length = int(f.msb)<<8 | int(b)
			if f.length == 0 || f.length > maxFrameSize {
				f.state = waitStart1
			} else {
				f.pos = 0
				f.state = waitPayload
			}
		case waitPayload:
			f.buf[f.pos] = b
			f.pos++
			if f.pos >= f.length {
				frame := make([]byte, f.length)
				copy(frame, f.buf[:f.length])
				frames = append(frames, frame)
				f.state = waitStart1
			}
		}
	}
	return frames
}

// Encode wraps a protobuf payload in the Meshtastic serial frame format.
func Encode(payload []byte) []byte {
	frame := make([]byte, 4+len(payload))
	frame[0] = start1
	frame[1] = start2
	frame[2] = byte(len(payload) >> 8)
	frame[3] = byte(len(payload))
	copy(frame[4:], payload)
	return frame
}
