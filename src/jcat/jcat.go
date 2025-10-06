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
  jcat listen [port]          Listen on port (default :1337)
  jcat connect <host:port>    Connect to remote host
  jcat version               Show version
  jcat help                  Show this help

Examples:
  jcat listen                # Listen on default port :1337
  jcat listen :8080          # Listen on port 8080
  jcat listen 0.0.0.0:1337   # Listen on all interfaces, port 1337
  jcat connect localhost:1337 # Connect to localhost:1337
  jcat connect 192.168.1.100:8080 # Connect to remote host
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