package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/tk-425/Codefind/internal/pathutil"
)

type Client struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	reader    *bufio.Reader
	msgID     int
	mu        sync.Mutex
	responses map[int]chan json.RawMessage
	language  string
	rootPath  string
}

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonRPCNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type InitializeParams struct {
	ProcessID    int                `json:"processId"`
	RootPath     string             `json:"rootPath"`
	RootURI      string             `json:"rootUri"`
	Capabilities ClientCapabilities `json:"capabilities"`
}

type ClientCapabilities struct {
	TextDocument TextDocumentClientCapabilities `json:"textDocument"`
}

type TextDocumentClientCapabilities struct {
	DocumentSymbol DocumentSymbolCapabilities `json:"documentSymbol"`
}

type DocumentSymbolCapabilities struct {
	HierarchicalDocumentSymbolSupport bool `json:"hierarchicalDocumentSymbolSupport"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

type ServerCapabilities struct {
	DocumentSymbolProvider any `json:"documentSymbolProvider"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           SymbolKind       `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

type SymbolInformation struct {
	Name     string     `json:"name"`
	Kind     SymbolKind `json:"kind"`
	Location Location   `json:"location"`
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type SymbolKind int

const (
	SymbolKindFile          SymbolKind = 1
	SymbolKindModule        SymbolKind = 2
	SymbolKindNamespace     SymbolKind = 3
	SymbolKindPackage       SymbolKind = 4
	SymbolKindClass         SymbolKind = 5
	SymbolKindMethod        SymbolKind = 6
	SymbolKindProperty      SymbolKind = 7
	SymbolKindField         SymbolKind = 8
	SymbolKindConstructor   SymbolKind = 9
	SymbolKindEnum          SymbolKind = 10
	SymbolKindInterface     SymbolKind = 11
	SymbolKindFunction      SymbolKind = 12
	SymbolKindVariable      SymbolKind = 13
	SymbolKindConstant      SymbolKind = 14
	SymbolKindString        SymbolKind = 15
	SymbolKindNumber        SymbolKind = 16
	SymbolKindBoolean       SymbolKind = 17
	SymbolKindArray         SymbolKind = 18
	SymbolKindObject        SymbolKind = 19
	SymbolKindKey           SymbolKind = 20
	SymbolKindNull          SymbolKind = 21
	SymbolKindEnumMember    SymbolKind = 22
	SymbolKindStruct        SymbolKind = 23
	SymbolKindEvent         SymbolKind = 24
	SymbolKindOperator      SymbolKind = 25
	SymbolKindTypeParameter SymbolKind = 26
)

func (sk SymbolKind) String() string {
	kinds := map[SymbolKind]string{
		SymbolKindFile:          "file",
		SymbolKindModule:        "module",
		SymbolKindNamespace:     "namespace",
		SymbolKindPackage:       "package",
		SymbolKindClass:         "class",
		SymbolKindMethod:        "method",
		SymbolKindProperty:      "property",
		SymbolKindField:         "field",
		SymbolKindConstructor:   "constructor",
		SymbolKindEnum:          "enum",
		SymbolKindInterface:     "interface",
		SymbolKindFunction:      "function",
		SymbolKindVariable:      "variable",
		SymbolKindConstant:      "constant",
		SymbolKindString:        "string",
		SymbolKindNumber:        "number",
		SymbolKindBoolean:       "boolean",
		SymbolKindArray:         "array",
		SymbolKindObject:        "object",
		SymbolKindKey:           "key",
		SymbolKindNull:          "null",
		SymbolKindEnumMember:    "enum_member",
		SymbolKindStruct:        "struct",
		SymbolKindEvent:         "event",
		SymbolKindOperator:      "operator",
		SymbolKindTypeParameter: "type_parameter",
	}
	if s, ok := kinds[sk]; ok {
		return s
	}
	return "unknown"
}

func NewClient(language, rootPath string) (*Client, error) {
	server, ok := KnownLSPs[language]
	if !ok {
		return nil, fmt.Errorf("no LSP configured for language: %s", language)
	}

	execPath, err := exec.LookPath(server.Executable)
	if err != nil {
		return nil, fmt.Errorf("LSP executable not found: %s", server.Executable)
	}

	cmd := exec.Command(execPath, server.Args...)
	cmd.Dir = rootPath

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start LSP: %w", err)
	}

	client := &Client{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		reader:    bufio.NewReader(stdout),
		responses: make(map[int]chan json.RawMessage),
		language:  language,
		rootPath:  rootPath,
	}

	go client.readResponses()

	return client, nil
}

func (c *Client) Initialize(ctx context.Context) error {
	params := InitializeParams{
		ProcessID: os.Getpid(),
		RootPath:  c.rootPath,
		RootURI:   "file://" + c.rootPath,
		Capabilities: ClientCapabilities{
			TextDocument: TextDocumentClientCapabilities{
				DocumentSymbol: DocumentSymbolCapabilities{
					HierarchicalDocumentSymbolSupport: true,
				},
			},
		},
	}

	var result InitializeResult
	if err := c.call(ctx, "initialize", params, &result); err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}
	if err := c.notify("initialized", struct{}{}); err != nil {
		return fmt.Errorf("initialized notification failed: %w", err)
	}
	return nil
}

func (c *Client) Shutdown(ctx context.Context) error {
	if c == nil {
		return nil
	}

	readyForRPC := c.stdin != nil && c.stdout != nil && c.reader != nil && c.responses != nil
	var result any
	if readyForRPC {
		_ = c.call(ctx, "shutdown", nil, &result)
		_ = c.notify("exit", nil)
	}
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.stdout != nil {
		_ = c.stdout.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	return nil
}

func (c *Client) IsAlive() bool {
	if c == nil || c.cmd == nil || c.cmd.Process == nil {
		return false
	}
	return c.cmd.ProcessState == nil
}

func (c *Client) DocumentSymbols(ctx context.Context, filePath string) ([]DocumentSymbol, error) {
	if !pathutil.IsWithinDir(c.rootPath, filePath) {
		return nil, fmt.Errorf("file path outside repo root")
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	uri := "file://" + absPath
	openParams := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: c.getLanguageID(),
			Version:    1,
			Text:       string(content),
		},
	}
	if err := c.notify("textDocument/didOpen", openParams); err != nil {
		return nil, fmt.Errorf("didOpen failed: %w", err)
	}

	var rawResult json.RawMessage
	if err := c.call(ctx, "textDocument/documentSymbol", DocumentSymbolParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}, &rawResult); err != nil {
		return nil, fmt.Errorf("documentSymbol failed: %w", err)
	}

	_ = c.notify("textDocument/didClose", DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	})

	var symbols []DocumentSymbol
	if err := json.Unmarshal(rawResult, &symbols); err != nil {
		var infos []SymbolInformation
		if err2 := json.Unmarshal(rawResult, &infos); err2 != nil {
			return nil, fmt.Errorf("failed to parse symbols: %w", err)
		}
		for _, info := range infos {
			symbols = append(symbols, DocumentSymbol{
				Name:  info.Name,
				Kind:  info.Kind,
				Range: info.Location.Range,
			})
		}
	}

	return symbols, nil
}

func (c *Client) getLanguageID() string {
	switch c.language {
	case "typescript/javascript":
		return "typescript"
	default:
		return c.language
	}
}

func (c *Client) call(ctx context.Context, method string, params any, result any) error {
	c.mu.Lock()
	c.msgID++
	id := c.msgID
	respChan := make(chan json.RawMessage, 1)
	c.responses[id] = respChan
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.responses, id)
		c.mu.Unlock()
	}()

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	if err := c.send(req); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case rawResult := <-respChan:
		if result != nil && len(rawResult) > 0 {
			return json.Unmarshal(rawResult, result)
		}
		return nil
	}
}

func (c *Client) notify(method string, params any) error {
	return c.send(jsonRPCNotification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	})
}

func (c *Client) send(msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := c.stdin.Write([]byte(header)); err != nil {
		return err
	}
	_, err = c.stdin.Write(data)
	return err
}

func (c *Client) readResponses() {
	for {
		contentLength, ok := c.readContentLength()
		if !ok || contentLength == 0 {
			return
		}

		body := make([]byte, contentLength)
		if _, err := io.ReadFull(c.reader, body); err != nil {
			return
		}

		var resp jsonRPCResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			continue
		}

		c.mu.Lock()
		if ch, ok := c.responses[resp.ID]; ok {
			if resp.Error != nil {
				ch <- nil
			} else {
				ch <- resp.Result
			}
		}
		c.mu.Unlock()
	}
}

func (c *Client) readContentLength() (int, bool) {
	contentLength := 0
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return 0, false
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return contentLength, true
		}
		if lengthStr, ok := strings.CutPrefix(line, "Content-Length:"); ok {
			lengthStr = strings.TrimSpace(lengthStr)
			length, err := strconv.Atoi(lengthStr)
			if err == nil {
				contentLength = length
			}
		}
	}
}
