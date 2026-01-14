package main

import (
	"bufio"
	"os"
	"fmt"
	"encoding/json"
)

type Server struct {
	in *bufio.Reader
	out *bufio.Writer
}

type Request struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      *int        `json:"id"`
	Result  any `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func main() {
	server := NewServer()

	server.Run()
}

func NewServer() *Server {
	return &Server{
		in: bufio.NewReader(os.Stdin),
		out: bufio.NewWriter(os.Stdout),
	}
}

func (s *Server) Run() {
	for {
		msg, err := readMessage(s.in)
		if err != nil {
			return
		}

		s.handleMessage(msg)
	}
}

func (s *Server) handleMessage(req *Request) {
	switch req.Method {
	case "initalize":
		s.handleIntialize(req)
	case "shutdown":
		s.sendResponse(req.ID, nil)
	case "exit":
		os.Exit(0)
	}
}

func (s *Server) handleIntialize(req *Request) {
	result := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"textDocumentSync": 1,
		},
	}

	s.sendResponse(req.ID, result)
}

func (s *Server) sendResponse(id *int, result interface{}) {
	resp := Response{
		Jsonrpc: "2.0",
		ID: id,
		Result: result,
	}

	data, _ := json.Marshal(resp)
	writeMessage(s.out, data)
}

func readMessage(r *bufio.Reader) (*Request, error) {
	// read headers
	var contentLength int
	for {
		line, _ := r.ReadString('\n')
		if line == "\r\n" {
			break
		}
		fmt.Sscanf(line, "Content-Length: %d\r\n", &contentLength)
	}

	body := make([]byte, contentLength)
	r.Read(body)

	var req Request
	json.Unmarshal(body, &req)
	return &req, nil
}

func writeMessage(w *bufio.Writer, data []byte) {
	fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(data))
	w.Write(data)
	w.Flush()
}