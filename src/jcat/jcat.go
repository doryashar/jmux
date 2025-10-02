// Binary jcat can serve as both client and server for remote tty shells.
package main

import (
	"encoding/gob"
	"flag"
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

var (
	server  = flag.Bool("server", false, "Run as server")
	client  = flag.Bool("client", false, "Run as client")
	listen  = flag.String("listen", ":1337", "Address to listen on ([ip]:port) when running as server")
	connect = flag.String("connect", ":1337", "Address to connect to ([ip]:port) when running as client")
)

func main() {
	flag.Parse()

	if *server && *client {
		log.Fatal("cannot run as both server and client")
	}

	if !*server && !*client {
		log.Fatal("must specify either -server or -client")
	}

	if *server {
		runServer()
	} else {
		runClient()
	}
}

func runServer() {
	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("listening on %s", *listen)
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

func runClient() {
	conn, err := net.Dial("tcp", *connect)
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