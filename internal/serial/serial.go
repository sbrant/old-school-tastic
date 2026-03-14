package serial

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"go.bug.st/serial"
	"meshtastic-cli/internal/proto"
)

const defaultBaud = 115200

const specialNonceOnlyConfig = 69420

type Conn struct {
	port      serial.Port
	path      string
	baud      int
	skipNodes bool
	framer    Framer
	Packets   chan []byte
	done      chan struct{}
}

func Open(path string, baud int, skipNodes bool) (*Conn, error) {
	if baud <= 0 {
		baud = defaultBaud
	}

	port, err := serial.Open(path, &serial.Mode{BaudRate: baud})
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}

	c := &Conn{
		port:      port,
		path:      path,
		baud:      baud,
		skipNodes: skipNodes,
		Packets:   make(chan []byte, 512),
		done:      make(chan struct{}),
	}

	go c.readLoop()

	// Request config immediately — packets will buffer in the channel
	// until the TUI starts consuming them
	c.requestConfig()

	return c, nil
}

// RequestConfig sends a want_config_id to the device. Call this after
// the consumer is ready to read from the Packets channel.
func (c *Conn) RequestConfig() {
	c.requestConfig()
}

func (c *Conn) requestConfig() {
	go func() {
		// Send START bytes to wake serial API, then config request
		// Some devices need a nudge before they'll respond
		c.port.Write([]byte{start1, start2, 0x00, 0x00})
		time.Sleep(100 * time.Millisecond)

		nonce := rand.Uint32()
		if c.skipNodes {
			nonce = specialNonceOnlyConfig
		}
		data, err := proto.EncodeWantConfig(nonce)
		if err != nil {
			log.Printf("warning: failed to encode config request: %v", err)
			return
		}
		if err := c.Send(data); err != nil {
			log.Printf("warning: config request failed: %v", err)
		}

		// Send again after a delay in case the first was missed
		time.Sleep(1 * time.Second)
		c.Send(data)
	}()
}

func (c *Conn) readLoop() {
	defer close(c.Packets)
	buf := make([]byte, 1024)

	for {
		n, err := c.port.Read(buf)
		if err != nil {
			select {
			case <-c.done:
				return
			default:
			}

			// Try to reconnect
			log.Printf("serial read error: %v, reconnecting...", err)
			if c.reconnect() {
				continue
			}
			return
		}

		for _, frame := range c.framer.Feed(buf[:n]) {
			select {
			case c.Packets <- frame:
			case <-c.done:
				return
			}
		}
	}
}

func (c *Conn) reconnect() bool {
	c.port.Close()
	backoff := time.Second

	for attempt := 0; attempt < 30; attempt++ {
		select {
		case <-c.done:
			return false
		default:
		}

		time.Sleep(backoff)
		if backoff < 10*time.Second {
			backoff = backoff * 3 / 2
		}

		port, err := serial.Open(c.path, &serial.Mode{BaudRate: c.baud})
		if err != nil {
			log.Printf("reconnect attempt %d failed: %v", attempt+1, err)
			continue
		}

		c.port = port
		c.framer = Framer{} // reset parser state
		log.Printf("reconnected to %s", c.path)

		// Re-request config after reconnect
		c.requestConfig()
		return true
	}

	log.Printf("gave up reconnecting after 30 attempts")
	return false
}

func (c *Conn) Send(payload []byte) error {
	_, err := c.port.Write(Encode(payload))
	return err
}

func (c *Conn) Close() error {
	close(c.done)
	return c.port.Close()
}
