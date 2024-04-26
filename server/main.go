package main

import (
	"fmt"
	"log"
	"net"
)

type Message struct {
	Text string
	Conn net.Conn
}

type Server struct {
	Port string
	connections []net.Conn
	messages chan Message
}

func (s *Server) ListenAndBroadcast() {
	for {
		msg := <- s.messages
		for _, conn := range s.connections {
			if conn != msg.Conn {
				conn.Write([]byte(msg.Text))
			}
		}
	}
}


func (s *Server) handleClientConnection(conn net.Conn) {
	s.connections = append(s.connections, conn)

	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			conn.Write([]byte("ERROR:\n"))
			conn.Close()
		}
		text := string(buffer[0:n])
		s.messages <- Message{Text: text, Conn: conn}
	}
}

func main() {

	server := Server {
		Port: "6969",
		connections: make([]net.Conn, 0),
		messages: make(chan Message),
	}

	ln, err := net.Listen("tcp", ":"+server.Port)
	if err != nil{
		log.Fatalf("Could not initialize the server: %s", err)
	}
	fmt.Println("Ready for accepting the connections")


	go server.ListenAndBroadcast()

	// Accept the incoming connections
	for {
		conn, err := ln.Accept();
		if err != nil{
			log.Println("Could not accept the connection")
			continue;
		}

		go server.handleClientConnection(conn)
	}
}
