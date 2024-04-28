package main

import (
	"fmt"
	"net"
	"strings"
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

func ParseMessageFromBuffer(conn net.Conn, buf string) (Message, error) {
	values := strings.Split(buf, "\r\n")
	if len(values) < 3 {
		return Message{}, fmt.Errorf("invalid message format")
	}
	switch values[0] {
	case "MESSAGE":
		return Message{Type: MESSAGE, Nickname: values[1], Text: values[2], Conn: conn, originalMessage: buf}, nil
	case "WHISPER":
		if len(values) < 4 {
			return Message{}, fmt.Errorf("invalid whisper message format")
		}
		return Message{Type: WHISPER, WhisperTo: values[1], Nickname: values[2], Text: values[3], Conn: conn, originalMessage: buf}, nil
	default:
		return Message{}, fmt.Errorf("invalid message type")
	}
}
