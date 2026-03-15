package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"meshtastic-cli/internal/logger"
	"meshtastic-cli/internal/serial"
	"meshtastic-cli/internal/store"
	"meshtastic-cli/internal/tui"
)

func main() {
	baud := flag.Int("baud", 115200, "serial baud rate")
	session := flag.String("session", "default", "database session name")
	packetLimit := flag.Int("packet-limit", 1000, "max packets to store")
	skipNodes := flag.Bool("skip-nodes", false, "skip downloading node database on startup (faster connect)")
	skipConfig := flag.Bool("skip-config", false, "skip loading device config on startup (faster connect)")
	bot := flag.Bool("bot", false, "auto-reply to ping and test messages")
	bbsMode := flag.Bool("bbs", false, "enable BBS mode (auto-respond to commands)")
	fahrenheit := flag.Bool("fahrenheit", false, "display temperatures in Fahrenheit")
	pcapFile := flag.String("pcap", "", "write packets to pcap file")
	enableLogging := flag.Bool("enable-logging", false, "enable verbose logging to ~/.config/meshtastic-cli/log")
	clearSession := flag.Bool("clear", false, "clear the database for the session and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Meshtastic CLI Viewer

Usage: %s [options] <serial-port>

Arguments:
  serial-port    Serial port path (e.g. /dev/ttyUSB0, /dev/ttyACM0)

Options:
`, os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample: %s /dev/ttyUSB0\n", os.Args[0])
	}
	flag.Parse()

	// Suppress Go's default logger from writing to stderr (breaks TUI)
	log.SetOutput(io.Discard)

	// Initialize logger
	logger.Init(*enableLogging)
	defer logger.Close()

	// Handle --clear
	if *clearSession {
		path := store.DbPath(*session)
		os.Remove(path)
		os.Remove(path + "-wal")
		os.Remove(path + "-shm")
		fmt.Printf("Cleared database for session %q\n", *session)
		return
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}
	port := flag.Arg(0)

	// Open database
	db, err := store.Open(*session, *packetLimit)
	if err != nil {
		logger.LogError("failed to open database", err)
		fmt.Fprintf(os.Stderr, "error: failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// skip-config implies skip-nodes
	skip := *skipNodes || *skipConfig

	// Open serial port
	conn, err := serial.Open(port, *baud, skip)
	if err != nil {
		logger.LogError("failed to open serial port", err)
		fmt.Fprintf(os.Stderr, "error: failed to open serial port: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	logger.Info("Main", fmt.Sprintf("starting: port=%s baud=%d session=%s", port, *baud, *session))

	// Run TUI
	opts := tui.Options{
		Bot:        *bot,
		BBS:        *bbsMode,
		Fahrenheit: *fahrenheit,
		PcapFile:   *pcapFile,
	}
	m := tui.NewModel(conn, db, opts)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		logger.LogError("application error", err)
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Main", "shutdown complete")
}
