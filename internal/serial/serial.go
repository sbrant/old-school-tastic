package serial

import (
	"fmt"
	"math/rand"
	"time"

	"go.bug.st/serial"
	"meshtastic-cli/internal/logger"
	"meshtastic-cli/internal/proto"
)

const defaultBaud = 115200

const specialNonceOnlyConfig = 69420

type ConnStatus int

const (
	StatusConnected ConnStatus = iota
	StatusDisconnected
	StatusReconnecting
)

type Conn struct {
	port      serial.Port
	path      string
	baud      int
	skipNodes bool
	framer    Framer
	Packets   chan []byte
	Status    chan ConnStatus
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
		Status:    make(chan ConnStatus, 16),
		done:      make(chan struct{}),
	}

	go c.readLoop()
	c.requestConfig()

	return c, nil
}

func (c *Conn) RequestConfig() {
	c.requestConfig()
}

func (c *Conn) requestConfig() {
	go func() {
		c.port.Write([]byte{start1, start2, 0x00, 0x00})
		time.Sleep(100 * time.Millisecond)

		nonce := rand.Uint32()
		if c.skipNodes {
			nonce = specialNonceOnlyConfig
		}
		data, err := proto.EncodeWantConfig(nonce)
		if err != nil {
			logger.Error("Serial", fmt.Sprintf("failed to encode config request: %v", err))
			return
		}
		c.Send(data)

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

			logger.Warn("Serial", fmt.Sprintf("read error: %v, reconnecting...", err))
			c.emitStatus(StatusReconnecting)

			if c.reconnect() {
				c.emitStatus(StatusConnected)
				continue
			}
			c.emitStatus(StatusDisconnected)
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
			logger.Info("Serial", fmt.Sprintf("reconnect attempt %d failed: %v", attempt+1, err))
			continue
		}

		c.port = port
		c.framer = Framer{}
		logger.Info("Serial", fmt.Sprintf("reconnected to %s", c.path))

		c.requestConfig()
		return true
	}

	logger.Error("Serial", "gave up reconnecting after 30 attempts")
	return false
}

func (c *Conn) emitStatus(s ConnStatus) {
	select {
	case c.Status <- s:
	default:
	}
}

func (c *Conn) Send(payload []byte) error {
	_, err := c.port.Write(Encode(payload))
	return err
}

func (c *Conn) Close() error {
	close(c.done)
	return c.port.Close()
}
