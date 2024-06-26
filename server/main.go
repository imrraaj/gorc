package main

import (
	"fmt"
	"log"
	"net"
	"sync"

	common "github.com/imrraaj/gorc/common"
)

type Server struct {
	Port        string
	connections map[string]net.Conn
	messages    chan common.Message
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
	n, err := conn.Read(initialBuf)
	if err != nil || n == 0 {
		log.Println("Could not read the nickname")
		conn.Close()
		return
	}
	m, err := common.ParseMessageFromBuffer(conn, string(initialBuf[:n]))
	if err != nil {
		log.Println("Could not parse the message")
		conn.Close()
		return
	}
	if m.Type != common.MESSAGE {
		log.Println("Invalid message type")
		conn.Close()
		return
	}
	nickname := m.Nickname
	log.Printf("New user connected: %s\n", nickname)

	s.mutex.Lock()
	s.connections[nickname] = conn
	s.mutex.Unlock()
	s.BroadcastMessage(fmt.Sprintf("New user connected: %s\n", nickname))

	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil || n == 0 {
			log.Println("Someone disconnected")
			s.mutex.Lock()
			delete(s.connections, nickname)
			s.mutex.Unlock()
			go s.BroadcastMessage(fmt.Sprintf("User disconnected: %s\n", nickname))
			conn.Close()
			break
		}
		m, err := common.ParseMessageFromBuffer(conn, string(buffer[:n]))
		if err != nil {
			log.Println("Could not parse the message")
			continue
		}
		if m.Type == common.WHISPER {
			s.mutex.Lock()
			if c, ok := s.connections[m.WhisperTo]; ok {
				c.Write([]byte(m.originalMessage))
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
		messages:    make(chan common.Message),
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
