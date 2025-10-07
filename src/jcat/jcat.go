// Binary jcat can serve as both client and server for remote tty shells.
package main

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

	"github.com/hashicorp/yamux"
	"github.com/creack/pty"
	"golang.org/x/crypto/ssh/terminal"
)
const (
	JcatVersion = "1.0.0"
	HandshakeMsg = "JCAT/" + JcatVersion + "\n"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	
	switch command {
	case "listen":
		address := ":1337" // default
		if len(os.Args) > 2 {
			address = os.Args[2]
		}
		runServer(address)
	case "connect":
		if len(os.Args) < 3 {
			log.Fatal("connect command requires host:port argument")
		}
		address := os.Args[2]
		runClient(address)
	case "reverse-listen":
		address := ":1337" // default
		if len(os.Args) > 2 {
			address = os.Args[2]
		}
		runReverseServer(address)
	case "reverse-connect":
		if len(os.Args) < 3 {
			log.Fatal("reverse-connect command requires host:port argument")
		}
		address := os.Args[2]
		runReverseClient(address)
	case "version":
		fmt.Printf("jcat version %s\n", JcatVersion)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`jcat - TCP tunnel for terminal sharing

Usage:
  jcat listen [port]              Listen on port (default :1337) and share your shell
  jcat connect <host:port>        Connect to remote host and receive their shell
  jcat reverse-listen [port]      Listen on port and wait to receive shell from client
  jcat reverse-connect <host:port> Connect to remote host and share your shell with them
  jcat version                   Show version
  jcat help                      Show this help

Examples:
  # Normal mode (server shares shell with client):
  jcat listen                    # Listen on default port :1337
  jcat connect localhost:1337    # Connect and receive server's shell
  
  # Reverse mode (client shares shell with server):
  jcat reverse-listen :8080      # Listen and wait for client to share their shell
  jcat reverse-connect localhost:8080 # Connect and share your shell with server
  
  # Equivalent to socat reverse shell:
  # socat file:tty,raw,echo=0 tcp-listen:12345
  jcat reverse-listen :12345
  
  # socat tcp-connect:$RHOST:12345 exec:/bin/bash,pty,stderr,setsid,sigint,sane  
  jcat reverse-connect $RHOST:12345
`)
}

func runServer(address string) {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("listening on %s", address)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[%s] accept error: %v", conn.RemoteAddr().String(), err)
			continue
		}
		go handle(conn)
	}
}

func handle(conn net.Conn) {
	remote := conn.RemoteAddr().String()
	local := conn.LocalAddr().String()
	
	// Send handshake message first
	_, err := conn.Write([]byte(HandshakeMsg))
	if err != nil {
		log.Printf("[%s] handshake error: %v", remote, err)
		return
	}
	
	// Configure yamux with longer keepalive to avoid timeout issues
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

	// Extract ports and addresses for SOCAT environment variables
	var remoteHost, remotePort, localPort string
	if host, port, err := net.SplitHostPort(remote); err == nil {
		remoteHost = host
		remotePort = port
	}
	if _, port, err := net.SplitHostPort(local); err == nil {
		localPort = port
	}
	
	var cmd *exec.Cmd
	if rcfile := os.Getenv("JCAT_SETSIZE_SCRIPT"); rcfile != "" {
		// Create a wrapper script that exports SOCAT variables and then sources the rcfile
		wrapperScript := fmt.Sprintf(`
export SOCAT_SOCKPORT=%s
export SOCAT_PEERADDR=%s
export SOCAT_PEERPORT=%s
source %s
`, localPort, remoteHost, remotePort, rcfile)
		cmd = exec.Command("/bin/bash", "-c", wrapperScript)
	} else {
		// Create a wrapper script that exports SOCAT variables and starts interactive bash
		wrapperScript := fmt.Sprintf(`
export SOCAT_SOCKPORT=%s
export SOCAT_PEERADDR=%s
export SOCAT_PEERPORT=%s
exec /bin/bash -i
`, localPort, remoteHost, remotePort)
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
			if err := Setsize(shellPty, win.Rows, win.Cols); err != nil {
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

func runClient(address string) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Fatalf("connection error: %v", err)
	}

	// Read handshake message
	handshake := make([]byte, len(HandshakeMsg))
	_, err = io.ReadFull(conn, handshake)
	if err != nil {
		log.Fatalf("handshake error: %v", err)
	}
	
	if string(handshake) == HandshakeMsg {
		log.Printf("Connected to jcat server version %s", JcatVersion)
	} else {
		log.Fatalf("invalid handshake: %s", string(handshake))
	}

	// Configure yamux client with same settings as server
	config := yamux.DefaultConfig()
	config.EnableKeepAlive = true
	config.KeepAliveInterval = 30 * time.Second
	config.ConnectionWriteTimeout = 30 * time.Second
	
	session, err := yamux.Client(conn, config)
	if err != nil {
		log.Fatalf("session error: %v", err)
	}

	stdin := int(os.Stdin.Fd())
	if !terminal.IsTerminal(stdin) {
		log.Fatal("not on a terminal")
	}
	oldState, err := terminal.MakeRaw(stdin)
	if err != nil {
		log.Fatalf("unable to make terminal raw: %v", err)
	}
	defer terminal.Restore(stdin, oldState)

	done := make(chan struct{})

	controlChannel, err := session.Open()
	if err != nil {
		log.Fatalf("control channel open error: %v", err)
	}
	go func() {
		w := gob.NewEncoder(controlChannel)
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGWINCH)
		for {
			cols, rows, err := terminal.GetSize(stdin)
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

	dataChannel, err := session.Open()
	if err != nil {
		log.Fatalf("data channel open error: %v", err)
	}
	cp := func(dst io.Writer, src io.Reader) {
		io.Copy(dst, src)
		done <- struct{}{}
	}
	go cp(dataChannel, os.Stdin)
	go cp(os.Stdout, dataChannel)

	<-done
	session.Close()
}

type winsize struct {
	ws_row    uint16
	ws_col    uint16
	ws_xpixel uint16
	ws_ypixel uint16
}

func Setsize(f *os.File, rows, cols int) error {
	ws := winsize{ws_row: uint16(rows), ws_col: uint16(cols)}
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

// runReverseServer listens for connections and receives a shell from the client (ask-share equivalent)
func runReverseServer(address string) {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("🔗 Reverse listening on %s (waiting for client shell)", address)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[%s] accept error: %v", conn.RemoteAddr().String(), err)
			continue
		}

		// Close the listener to prevent new connections
		ln.Close()
		
		handleReverse(conn)
		break // Exit after handling one connection
	}
}

// handleReverse handles a reverse connection where the client shares their shell with us
func handleReverse(conn net.Conn) {
	remote := conn.RemoteAddr().String()
	
	// Send handshake message first
	_, err := conn.Write([]byte(HandshakeMsg))
	if err != nil {
		log.Printf("[%s] handshake error: %v", remote, err)
		return
	}
	
	// Configure yamux with longer keepalive to avoid timeout issues
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
	
	// Check if we're on a terminal first
	stdin := int(os.Stdin.Fd())
	isTerminal := terminal.IsTerminal(stdin)
	
	// Accept control channel from client
	controlChannel, err := session.Accept()
	if err != nil {
		log.Printf("[%s] control channel accept error: %v", remote, err)
		return
	}
	
	// Send our terminal size changes to client when we resize (reverse of normal flow)
	go func() {
		if !isTerminal {
			return
		}
		
		w := gob.NewEncoder(controlChannel)
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGWINCH)
		
		// Send initial terminal size
		cols, rows, err := terminal.GetSize(stdin)
		if err == nil {
			win := struct {
				Rows, Cols int
			}{Rows: rows, Cols: cols}
			w.Encode(win)
		}
		
		// Send updates when server terminal resizes
		for {
			<-c // Wait for SIGWINCH on server terminal
			cols, rows, err := terminal.GetSize(stdin)
			if err != nil {
				log.Printf("[%s] getsize error: %v", remote, err)
				break
			}
			win := struct {
				Rows, Cols int
			}{Rows: rows, Cols: cols}
			if err := w.Encode(win); err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	// Accept data channel from client  
	dataChannel, err := session.Accept()
	if err != nil {
		log.Printf("[%s] data channel accept error: %v", remote, err)
		return
	}
	if !isTerminal {
		log.Printf("[%s] not on a terminal - copying data only", remote)
		// Just copy data without terminal handling
		cp := func(dst io.Writer, src io.Reader) {
			io.Copy(dst, src)
			done <- struct{}{}
		}
		go cp(dataChannel, os.Stdin)
		go cp(os.Stdout, dataChannel)
		<-done
		return
	}
	
	// We're on a terminal, enable raw mode
	oldState, err := terminal.MakeRaw(stdin)
	if err != nil {
		log.Printf("[%s] unable to make terminal raw: %v", remote, err)
		return
	}
	defer terminal.Restore(stdin, oldState)
	
	// Clear the terminal and show minimal connection message
	fmt.Print("\033[2J\033[H") // Clear screen and move cursor to top
	fmt.Printf("🔗 Connected to shared session from %s\r\n", remote)
	
	cp := func(dst io.Writer, src io.Reader) {
		io.Copy(dst, src)
		done <- struct{}{}
	}
	go cp(dataChannel, os.Stdin)
	go cp(os.Stdout, dataChannel)

	<-done
	session.Close()
	
	// Restore terminal and show disconnection message
	fmt.Print("\r\n\r\nConnection to client shell closed.\r\n")
}

// runReverseClient connects to a server and shares our shell with them (join-share equivalent)
func runReverseClient(address string) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			fmt.Printf("❌ Connection refused: Remote server is not listening on %s\n", address)
			fmt.Printf("💡 The user may not be sharing or may have stopped sharing\n")
			os.Exit(1)
		} else if strings.Contains(err.Error(), "no route to host") {
			fmt.Printf("❌ Cannot reach host: %s\n", address)
			fmt.Printf("💡 Check network connectivity or hostname\n")
			os.Exit(1)
		} else {
			fmt.Printf("❌ Connection failed to %s: %v\n", address, err)
			os.Exit(1)
		}
	}

	// Read handshake message
	handshake := make([]byte, len(HandshakeMsg))
	_, err = io.ReadFull(conn, handshake)
	if err != nil {
		log.Fatalf("handshake error: %v", err)
	}
	
	if string(handshake) == HandshakeMsg {
		log.Printf("Connected to reverse jcat server version %s", JcatVersion)
	} else {
		log.Fatalf("invalid handshake: %s", string(handshake))
	}

	// Configure yamux client with same settings as server
	config := yamux.DefaultConfig()
	config.EnableKeepAlive = true
	config.KeepAliveInterval = 30 * time.Second
	config.ConnectionWriteTimeout = 30 * time.Second
	
	session, err := yamux.Client(conn, config)
	if err != nil {
		log.Fatalf("session error: %v", err)
	}

	stdin := int(os.Stdin.Fd())
	var oldState *terminal.State
	isTerminal := terminal.IsTerminal(stdin)
	
	if isTerminal {
		oldState, err = terminal.MakeRaw(stdin)
		if err != nil {
			log.Printf("unable to make terminal raw: %v", err)
			isTerminal = false
		} else {
			defer terminal.Restore(stdin, oldState)
		}
	}

	done := make(chan struct{})

	// Create control channel to receive window size updates from server
	controlChannel, err := session.Open()
	if err != nil {
		log.Fatalf("control channel open error: %v", err)
	}

	// Create data channel for terminal I/O
	dataChannel, err := session.Open()
	if err != nil {
		log.Fatalf("data channel open error: %v", err)
	}
	
	// Create bash shell to share with the server
	var cmd *exec.Cmd
	var shellPty *os.File
	
	if rcfile := os.Getenv("JCAT_SETSIZE_SCRIPT"); rcfile != "" {
		// Create a wrapper script that sources the rcfile
		fmt.Println("Starting JCAT_SETSIZE_SCRIPT interactive bash shell...")
		wrapperScript := fmt.Sprintf(`source %s`, rcfile)
		cmd = exec.Command("/bin/bash", "-c", wrapperScript)
	} else {
		// Start interactive bash
		fmt.Println("Starting interactive bash shell...")
		cmd = exec.Command("/bin/bash", "-i")
	}
	
	// // Try to start with PTY first
	// shellPty, err = pty.Start(cmd)
	
	f, tty, err := pty.Open()
	if err != nil {
		log.Fatalf("failed to open pty: %v", err)
	}
	defer tty.Close()

	cmd.Stdin = tty
	cmd.Stdout = tty
	cmd.Stderr = tty

	if err := cmd.Start(); err != nil {
		log.Fatalf("cmd start error: %v", err)
	}
	shellPty = f

	// Handle window size updates from server and apply to our shell PTY
	go func() {
		r := gob.NewDecoder(controlChannel)
		for {
			var win struct {
				Rows, Cols int
			}
			if err := r.Decode(&win); err != nil {
				break
			}
			// Apply server's terminal size to our shell PTY and send SIGWINCH to shell process
			if cmd.Process != nil {
				cols := uint16(win.Cols)
				rows := uint16(win.Rows)
				// Resize the PTY
				_, _, err := syscall.Syscall(syscall.SYS_IOCTL, shellPty.Fd(), uintptr(syscall.TIOCSWINSZ), 
					uintptr(unsafe.Pointer(&struct {
						Row    uint16
						Col    uint16
						Xpixel uint16
						Ypixel uint16
					}{rows, cols, 0, 0})))
				if err != 0 {
					log.Printf("failed to set shell PTY size: %v", err)
				} else {
					// Send SIGWINCH to the shell process
					if err := syscall.Kill(cmd.Process.Pid, syscall.SIGWINCH); err != nil {
						log.Printf("sigwinch error: %v", err)
					}
				}
			}
		}
		done <- struct{}{}
	}()

	if err != nil {
		log.Printf("PTY start failed (%v), falling back to standard exec", err)
		
		// Create a new command for non-PTY mode since the previous one might be in invalid state
		if rcfile := os.Getenv("JCAT_SETSIZE_SCRIPT"); rcfile != "" {
			wrapperScript := fmt.Sprintf(`source %s`, rcfile)
			cmd = exec.Command("/bin/bash", "-c", wrapperScript)
		} else {
			cmd = exec.Command("/bin/bash", "-i")
		}
		
		// Fallback: start without PTY
		var stdin io.WriteCloser
		var stdout io.ReadCloser
		
		stdin, err = cmd.StdinPipe()
		if err != nil {
			log.Fatalf("stdin pipe error: %v", err)
		}
		
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			log.Fatalf("stdout pipe error: %v", err)
		}
		
		cmd.Stderr = cmd.Stdout // Combine stderr with stdout
		
		if err = cmd.Start(); err != nil {
			log.Fatalf("command start error: %v", err)
		}
		
		// Create a combined reader/writer for non-PTY mode
		go func() {
			if err := cmd.Wait(); err != nil {
				log.Printf("shell wait error: %v", err)
			}
			done <- struct{}{}
		}()
		
		log.Printf("🔀 Sharing shell with server (non-PTY mode)...")
		
		cp := func(dst io.Writer, src io.Reader) {
			io.Copy(dst, src)
			done <- struct{}{}
		}
		go cp(dataChannel, stdout)
		go cp(stdin, dataChannel)
		
		<-done
		session.Close()
		return
	}
	
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("shell wait error: %v", err)
		}
		done <- struct{}{}
	}()
	
	log.Printf("🔀 Sharing shell with server...")
	
	cp := func(dst io.Writer, src io.Reader) {
		io.Copy(dst, src)
		done <- struct{}{}
	}
	go cp(dataChannel, shellPty)
	go cp(shellPty, dataChannel)

	<-done
	session.Close()
}