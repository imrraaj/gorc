package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

const (
	MESSAGE = iota
	WHISPER
)

type Message struct {
	originalMessage string
	Type            int
	Nickname        string
	WhisperTo       string
	Text            string
	Conn            net.Conn
}

func ParseMessageFromBuffer(Conn net.Conn, buf string) (Message, error) {
	values := strings.Split(buf, "\r\n")
	if len(values) < 3 {
		return Message{}, fmt.Errorf("invalid message format")
	}
	switch values[0] {
	case "MESSAGE":
		return Message{Type: MESSAGE, Nickname: values[1], Text: values[2], Conn: Conn, originalMessage: buf}, nil
	case "WHISPER":
		if len(values) < 4 {
			return Message{}, fmt.Errorf("invalid whisper message format")
		}
		return Message{Type: WHISPER, WhisperTo: values[1], Nickname: values[2], Text: values[3], Conn: Conn, originalMessage: buf}, nil
	default:
		return Message{}, fmt.Errorf("invalid message type")
	}

}

type Server struct {
	Port        string
	connections map[string]net.Conn
	messages    chan Message
	mutex       sync.Mutex
}

func (s *Server) BroadcastMessage(msg string) {
	s.mutex.Lock()
	for _, conn := range s.connections {
		conn.Write([]byte(msg))
	}
	s.mutex.Unlock()
}

func (s *Server) ListenAndBroadcast() {
	for {
		msg := <-s.messages
		s.mutex.Lock()
		for _, conn := range s.connections {
			if conn != msg.Conn {
				conn.Write([]byte(msg.originalMessage))
			}
		}
		s.mutex.Unlock()
	}
}

func (s *Server) handleClientConnection(conn net.Conn) {
	initialBuf := make([]byte, 1024)
	_, err := conn.Read(initialBuf)
	if err != nil {
		log.Println("Could not read the nickname")
		conn.Close()
		return
	}
	m, err := ParseMessageFromBuffer(conn, string(initialBuf))
	if err != nil {
		log.Println("Could not parse the message")
		conn.Close()
		return
	}
	if m.Type != MESSAGE {
		log.Println("Invalid message type")
		conn.Close()
		return
	}
	nickname := m.Nickname
	log.Printf("New user connected: %s\n", nickname)

	s.mutex.Lock()
	s.connections[nickname] = conn
	s.mutex.Unlock()
	// s.BroadcastMessage(fmt.Sprintf("New user connected: %s\n", nickname))

	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil || n == 0 {
			log.Println("Someone disconnected")
			s.mutex.Lock()
			delete(s.connections, conn.RemoteAddr().String())
			s.mutex.Unlock()
			// go s.BroadcastMessage("Somebody disconnected")
			conn.Close()
			break
		}
		m, err := ParseMessageFromBuffer(conn, string(buffer))
		if err != nil {
			log.Println("Could not parse the message")
			continue
		}
		if m.Type == WHISPER {
			s.mutex.Lock()
			if c, ok := s.connections[m.WhisperTo]; ok {
				c.Write(buffer)
			}
			s.mutex.Unlock()
			continue
		}
		s.messages <- m
	}
}

func main() {

	server := Server{
		Port:        "6969",
		connections: make(map[string]net.Conn),
		messages:    make(chan Message),
	}

	ln, err := net.Listen("tcp", ":"+server.Port)
	if err != nil {
		log.Fatalf("Could not initialize the server: %s", err)
	}
	fmt.Println("Ready for accepting the connections")

	go server.ListenAndBroadcast()

	// Accept the incoming connections
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Could not accept the connection")
			continue
		}
		go server.handleClientConnection(conn)
	}
}
