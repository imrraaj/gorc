package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	// "net"
)

type (
	errMsg error
)

type Client struct {
	Nickname string
	conn     net.Conn
}

type model struct {
	viewport viewport.Model
	textarea textarea.Model
	err      error
	client   Client
}

var senderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

func handleServerMessages(conn net.Conn) {
	// Read messages from server
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil || n == 0 {
			log.Println("Someone disconnected")
			conn.Close()
			break
		}
		values := strings.Split(string(buf), "\r\n")
		if len(values) < 3 {
			log.Println("invalid message format")
			continue
		}
		switch values[0] {
		case "MESSAGE":
			msg := senderStyle.Render(values[1]+": ") + values[2]
			messages = append(messages, msg)
		case "WHISPER":
			msg := senderStyle.Render(values[2]+" whispers: ") + values[3]
			messages = append(messages, msg)
		default:
			log.Fatalf("invalid message type")
		}
	}
}

var messages []string

func main() {
	address := flag.String("address", ":6969", "Address to connect to")
	flag.Parse()
	fmt.Printf("Connecting to %s\n", *address)

	conn, err := net.Dial("tcp", *address)
	if err != nil {
		log.Fatalf("ERROR: %s\n", err.Error())
	}
	defer conn.Close()
	reader := bufio.NewReader(os.Stdin)

	// ask for user name
	fmt.Print("Enter your nickname: ")
	nickname, err := reader.ReadString('\n')
	// strip newline character
	nickname = strings.TrimSuffix(nickname, "\n")
	if err != nil {
		log.Fatalf("Error reading from stdin: %s", err)
	}

	// Send message to server
	msg := "MESSAGE\r\n" + nickname + "\r\n" + ""
	_, err = conn.Write([]byte(msg))
	if err != nil {
		log.Fatalf("Error sending message to server: %s", err)
	}

	client := Client{
		conn:     conn,
		Nickname: nickname,
	}

	go handleServerMessages(conn)
	p := tea.NewProgram(initialModel(client), tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func initialModel(client Client) model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 256

	ta.SetWidth(100)
	ta.SetHeight(2)

	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.ShowLineNumbers = false

	vp := viewport.New(100, 20)
	vp.SetContent(`Welcome to the chat room!
					Type a message and press Enter to send.
				`)

	ta.KeyMap.InsertNewline.SetEnabled(true)

	megaModel := model{
		textarea: ta,
		viewport: vp,
		err:      nil,
		client:   client,
	}
	return megaModel
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case tea.KeyEnter:
			value := m.textarea.Value()
			// check if value starts with /whisper username msg
			if strings.HasPrefix(value, "/whisper") {
				values := strings.Split(value, " ")
				if len(values) < 3 {
					m.err = errMsg(fmt.Errorf("invalid whisper format"))
					return m, nil
				}
				msg := "WHISPER\r\n" + values[1] + "\r\n" + m.client.Nickname + "\r\n" + values[2]
				_, err := m.client.conn.Write([]byte(msg))
				if err != nil {
					log.Fatalf("Error sending message to server: %s", err)
				}
				messages = append(messages, senderStyle.Render("You whispered to "+values[1]+": ")+values[2])
				m.textarea.Reset()
				m.viewport.GotoBottom()
				break
			}

			msg := "MESSAGE\r\n" + m.client.Nickname + "\r\n" + m.textarea.Value()
			_, err := m.client.conn.Write([]byte(msg))
			if err != nil {
				log.Fatalf("Error sending message to server: %s", err)
			}
			messages = append(messages, senderStyle.Render("You: ")+m.textarea.Value())
			m.textarea.Reset()
			m.viewport.GotoBottom()
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.viewport.SetContent(strings.Join(messages, ""))
	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		m.viewport.View(),
		m.textarea.View(),
	)
}
