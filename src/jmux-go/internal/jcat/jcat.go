package jcat

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/creack/pty"
	"github.com/hashicorp/yamux"
	"golang.org/x/term"
)

const (
	JcatVersion  = "2.0.0"
	HandshakeMsg = "JCAT/" + JcatVersion + "\n"
)

// Server represents a jcat server
type Server struct {
	listenAddr string
	rcfile     string
}

// Client represents a jcat client
type Client struct {
	connectAddr string
	mode        string // "pair", "view", or "rogue"
}

// NewServer creates a new jcat server
func NewServer(listenAddr, rcfile string) *Server {
	return &Server{
		listenAddr: listenAddr,
		rcfile:     rcfile,
	}
}

// NewClient creates a new jcat client
func NewClient(connectAddr string) *Client {
	return &Client{
		connectAddr: connectAddr,
		mode:        "pair", // default mode
	}
}

// NewClientWithMode creates a new jcat client with specified mode
func NewClientWithMode(connectAddr, mode string) *Client {
	return &Client{
		connectAddr: connectAddr,
		mode:        mode,
	}
}

// Start starts the jcat server
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	log.Printf("jcat server listening on %s", s.listenAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go s.handle(conn)
	}
}

// Connect connects the jcat client
func (c *Client) Connect() error {
	conn, err := net.Dial("tcp", c.connectAddr)
	if err != nil {
		return err
	}

	// Read handshake message
	handshake := make([]byte, len(HandshakeMsg))
	_, err = io.ReadFull(conn, handshake)
	if err != nil {
		return fmt.Errorf("handshake error: %v", err)
	}

	if string(handshake) == HandshakeMsg {
		log.Printf("Connected to jcat server version %s", JcatVersion)
	} else {
		return fmt.Errorf("invalid handshake: %s", string(handshake))
	}

	// Send mode information to server
	modeMsg := fmt.Sprintf("MODE:%s\n", c.mode)
	_, err = conn.Write([]byte(modeMsg))
	if err != nil {
		return fmt.Errorf("failed to send mode: %v", err)
	}

	// Configure yamux client
	config := yamux.DefaultConfig()
	config.EnableKeepAlive = true
	config.KeepAliveInterval = 30 * time.Second
	config.ConnectionWriteTimeout = 30 * time.Second

	session, err := yamux.Client(conn, config)
	if err != nil {
		return err
	}

	stdin := int(os.Stdin.Fd())
	if !term.IsTerminal(stdin) {
		return fmt.Errorf("not on a terminal")
	}

	oldState, err := term.MakeRaw(stdin)
	if err != nil {
		return err
	}
	defer term.Restore(stdin, oldState)

	done := make(chan struct{})

	// Control channel for window size
	controlChannel, err := session.Open()
	if err != nil {
		return err
	}

	go func() {
		w := gob.NewEncoder(controlChannel)
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGWINCH)
		for {
			cols, rows, err := term.GetSize(stdin)
			if err != nil {
				log.Printf("getsize error: %v", err)
				break
			}
			win := struct {
				Rows, Cols int
			}{Rows: rows, Cols: cols}
			if err := w.Encode(win); err != nil {
				break
			}
			<-c
		}
		done <- struct{}{}
	}()

	// Data channel for I/O
	dataChannel, err := session.Open()
	if err != nil {
		return err
	}

	cp := func(dst io.Writer, src io.Reader) {
		io.Copy(dst, src)
		done <- struct{}{}
	}
	go cp(dataChannel, os.Stdin)
	go cp(os.Stdout, dataChannel)

	<-done
	session.Close()
	return nil
}

// handle handles a server connection
func (s *Server) handle(conn net.Conn) {
	remote := conn.RemoteAddr().String()
	local := conn.LocalAddr().String()

	// Send handshake message first
	_, err := conn.Write([]byte(HandshakeMsg))
	if err != nil {
		log.Printf("[%s] handshake error: %v", remote, err)
		return
	}

	// Read mode information from client
	modeBuffer := make([]byte, 256)
	n, err := conn.Read(modeBuffer)
	if err != nil {
		log.Printf("[%s] mode read error: %v", remote, err)
		return
	}
	
	// Parse mode from "MODE:value\n" format
	modeStr := string(modeBuffer[:n])
	clientMode := "pair" // default
	if strings.HasPrefix(modeStr, "MODE:") && strings.Contains(modeStr, "\n") {
		clientMode = strings.TrimSpace(strings.Split(modeStr, ":")[1])
		clientMode = strings.TrimSpace(strings.Split(clientMode, "\n")[0])
	}
	log.Printf("[%s] client joining in %s mode", remote, clientMode)

	// Configure yamux server
	config := yamux.DefaultConfig()
	config.EnableKeepAlive = true
	config.KeepAliveInterval = 30 * time.Second
	config.ConnectionWriteTimeout = 30 * time.Second

	session, err := yamux.Server(conn, config)
	if err != nil {
		log.Printf("[%s] session error: %v", remote, err)
		return
	}

	done := make(chan struct{})

	// Extract ports and addresses for environment variables
	var remoteHost, remotePort, localPort string
	if host, port, err := net.SplitHostPort(remote); err == nil {
		remoteHost = host
		remotePort = port
	}
	if _, port, err := net.SplitHostPort(local); err == nil {
		localPort = port
	}

	// Create bash command with environment variables
	var cmd *exec.Cmd
	if s.rcfile != "" {
		// Check if the rcfile exists before trying to source it
		wrapperScript := fmt.Sprintf(`
export SOCAT_SOCKPORT=%s
export SOCAT_PEERADDR=%s
export SOCAT_PEERPORT=%s
export JCAT_MODE=%s
if [[ -f "%s" ]]; then
    source %s
else
    echo "Warning: setsize script not found at %s" >&2
fi
exec /bin/bash -i
`, localPort, remoteHost, remotePort, clientMode, s.rcfile, s.rcfile, s.rcfile)
		cmd = exec.Command("/bin/bash", "-c", wrapperScript)
	} else {
		wrapperScript := fmt.Sprintf(`
export SOCAT_SOCKPORT=%s
export SOCAT_PEERADDR=%s
export SOCAT_PEERPORT=%s
export JCAT_MODE=%s
exec /bin/bash -i
`, localPort, remoteHost, remotePort, clientMode)
		cmd = exec.Command("/bin/bash", "-c", wrapperScript)
	}

	shellPty, err := pty.Start(cmd)
	if err != nil {
		log.Printf("[%s] pty error: %v", remote, err)
		return
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("[%s] wait error: %v", remote, err)
		}
		done <- struct{}{}
	}()

	// Control channel for window size updates
	controlChannel, err := session.Accept()
	if err != nil {
		log.Printf("[%s] control channel accept error: %v", remote, err)
		return
	}

	go func() {
		r := gob.NewDecoder(controlChannel)
		for {
			var win struct {
				Rows, Cols int
			}
			if err := r.Decode(&win); err != nil {
				break
			}
			if err := setSize(shellPty, win.Rows, win.Cols); err != nil {
				log.Printf("[%s] setsize error: %v", remote, err)
				break
			}
			if err := syscall.Kill(cmd.Process.Pid, syscall.SIGWINCH); err != nil {
				log.Printf("[%s] sigwinch error: %v", remote, err)
				break
			}
		}
		done <- struct{}{}
	}()

	// Data channel for I/O
	dataChannel, err := session.Accept()
	if err != nil {
		log.Printf("[%s] data channel accept error: %v", remote, err)
		return
	}

	cp := func(dst io.Writer, src io.Reader) {
		io.Copy(dst, src)
		done <- struct{}{}
	}
	go cp(dataChannel, shellPty)
	go cp(shellPty, dataChannel)

	<-done
	shellPty.Close()
	session.Close()
	log.Printf("[%s] done", remote)
}

// setSize sets the terminal size
func setSize(f *os.File, rows, cols int) error {
	ws := struct {
		ws_row    uint16
		ws_col    uint16
		ws_xpixel uint16
		ws_ypixel uint16
	}{
		ws_row: uint16(rows),
		ws_col: uint16(cols),
	}

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		syscall.TIOCSWINSZ,
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno != 0 {
		return syscall.Errno(errno)
	}
	return nil
}