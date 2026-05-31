package remote

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

type MessageType string

const (
	MsgProgress MessageType = "progress"
	MsgURL      MessageType = "url"
	MsgPort     MessageType = "port"
	MsgLog      MessageType = "log"
	MsgError    MessageType = "error"
	MsgDone     MessageType = "done"
)

// Message is the JSON Lines envelope exchanged over SSH stdio.
type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type ProgressPayload struct {
	Current int64 `json:"current"`
	Total   int64 `json:"total"`
	Percent int   `json:"percent"`
}

type URLPayload struct {
	URL        string `json:"url"`
	InstanceID string `json:"instance_id"`
}

// PortPayload carries the remote local port for SSH port forwarding.
// RemotePort is the port on the server; LocalPort is filled in by the client
// after it sets up the tunnel.
type PortPayload struct {
	RemotePort int `json:"remote_port"`
	LocalPort  int `json:"local_port,omitempty"`
}

type LogPayload struct {
	Text string `json:"text"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}

// WriteMessage encodes msg as a JSON line to w.
func WriteMessage(w io.Writer, msgType MessageType, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg := Message{Type: msgType, Payload: raw}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}

// ReadMessages scans r for JSON Lines and calls handler for each message.
// Returns on EOF or when handler returns a non-nil error.
func ReadMessages(r io.Reader, handler func(Message) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg Message
		if err := json.Unmarshal(line, &msg); err != nil {
			continue // skip malformed lines
		}
		if err := handler(msg); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// DecodePayload unmarshals msg.Payload into v.
func DecodePayload(msg Message, v any) error {
	return json.Unmarshal(msg.Payload, v)
}
