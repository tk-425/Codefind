package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LSPClient manages communication with an LSP server
type LSPClient struct {
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

// RequestTimeout is the default timeout for LSP requests
const RequestTimeout = 5 * time.Second

// JSON-RPC 2.0 message types
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type jsonRPCNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
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

// LSP Protocol types
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
	// DocumentSymbolProvider can be bool or an object, so we use interface{}
	DocumentSymbolProvider interface{} `json:"documentSymbolProvider"`
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

// DocumentSymbol represents a symbol in a document (hierarchical)
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           SymbolKind       `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

// SymbolInformation represents a symbol (flat structure, legacy)
type SymbolInformation struct {
	Name     string   `json:"name"`
	Kind     SymbolKind `json:"kind"`
	Location Location `json:"location"`
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

// SymbolKind represents the kind of a symbol
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

// String returns the string representation of a SymbolKind
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

// NewLSPClient creates a new LSP client for the given language
func NewLSPClient(language, rootPath string) (*LSPClient, error) {
	// Get LSP executable for language
	lspInfo, ok := KnownLSPs[language]
	if !ok {
		return nil, fmt.Errorf("no LSP configured for language: %s", language)
	}

	execPath, err := exec.LookPath(lspInfo.Executable)
	if err != nil {
		return nil, fmt.Errorf("LSP executable not found: %s", lspInfo.Executable)
	}

	// Start LSP process with any required arguments (e.g., --stdio)
	cmd := exec.Command(execPath, lspInfo.Args...)
	cmd.Dir = rootPath

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// Capture stderr to avoid noise
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start LSP: %w", err)
	}

	client := &LSPClient{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		reader:    bufio.NewReader(stdout),
		msgID:     0,
		responses: make(map[int]chan json.RawMessage),
		language:  language,
		rootPath:  rootPath,
	}

	// Start response reader goroutine
	go client.readResponses()

	return client, nil
}

// Initialize sends the initialize request to the LSP server
func (c *LSPClient) Initialize(ctx context.Context) error {
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

	// Send initialized notification
	if err := c.notify("initialized", struct{}{}); err != nil {
		return fmt.Errorf("initialized notification failed: %w", err)
	}

	return nil
}

// Shutdown sends the shutdown request and exit notification
func (c *LSPClient) Shutdown(ctx context.Context) error {
	// Send shutdown request
	var result interface{}
	if err := c.call(ctx, "shutdown", nil, &result); err != nil {
		// Ignore shutdown errors, just try to exit
	}

	// Send exit notification
	c.notify("exit", nil)

	// Close pipes
	c.stdin.Close()
	c.stdout.Close()

	// Kill process if still running
	if c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}

	return nil
}

// DocumentSymbols requests document symbols for a file
func (c *LSPClient) DocumentSymbols(ctx context.Context, filePath string) ([]DocumentSymbol, error) {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	uri := "file://" + filePath
	langID := c.getLanguageID()

	// Open document
	openParams := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: langID,
			Version:    1,
			Text:       string(content),
		},
	}
	if err := c.notify("textDocument/didOpen", openParams); err != nil {
		return nil, fmt.Errorf("didOpen failed: %w", err)
	}

	// Request symbols
	symbolParams := DocumentSymbolParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}

	var rawResult json.RawMessage
	if err := c.call(ctx, "textDocument/documentSymbol", symbolParams, &rawResult); err != nil {
		return nil, fmt.Errorf("documentSymbol failed: %w", err)
	}

	// Close document
	closeParams := DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}
	c.notify("textDocument/didClose", closeParams)

	// Parse result - can be DocumentSymbol[] or SymbolInformation[]
	var symbols []DocumentSymbol
	if err := json.Unmarshal(rawResult, &symbols); err != nil {
		// Try parsing as SymbolInformation[]
		var symbolInfos []SymbolInformation
		if err2 := json.Unmarshal(rawResult, &symbolInfos); err2 != nil {
			return nil, fmt.Errorf("failed to parse symbols: %w", err)
		}
		// Convert SymbolInformation to DocumentSymbol
		for _, si := range symbolInfos {
			symbols = append(symbols, DocumentSymbol{
				Name:  si.Name,
				Kind:  si.Kind,
				Range: si.Location.Range,
			})
		}
	}

	return symbols, nil
}

// getLanguageID returns the LSP language identifier
func (c *LSPClient) getLanguageID() string {
	langMap := map[string]string{
		"go":                    "go",
		"python":                "python",
		"typescript/javascript": "typescript",
		"rust":                  "rust",
		"java":                  "java",
		"swift":                 "swift",
		"ocaml":                 "ocaml",
	}
	if id, ok := langMap[c.language]; ok {
		return id
	}
	return c.language
}

// call sends a request and waits for response
func (c *LSPClient) call(ctx context.Context, method string, params interface{}, result interface{}) error {
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

// notify sends a notification (no response expected)
func (c *LSPClient) notify(method string, params interface{}) error {
	notif := jsonRPCNotification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	return c.send(notif)
}

// send writes a message to the LSP server
func (c *LSPClient) send(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	_, err = c.stdin.Write([]byte(header))
	if err != nil {
		return err
	}
	_, err = c.stdin.Write(data)
	return err
}

// readResponses continuously reads responses from the LSP server
func (c *LSPClient) readResponses() {
	for {
		// Read headers
		var contentLength int
		for {
			line, err := c.reader.ReadString('\n')
			if err != nil {
				return // Server closed
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break // End of headers
			}
			if strings.HasPrefix(line, "Content-Length:") {
				lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
				contentLength, _ = strconv.Atoi(lengthStr)
			}
		}

		if contentLength == 0 {
			continue
		}

		// Read body
		body := make([]byte, contentLength)
		_, err := io.ReadFull(c.reader, body)
		if err != nil {
			return
		}

		// Parse response
		var resp jsonRPCResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			continue // Skip malformed responses
		}

		// Route to waiting caller
		c.mu.Lock()
		if ch, ok := c.responses[resp.ID]; ok {
			if resp.Error != nil {
				// Send error as empty result
				ch <- nil
			} else {
				ch <- resp.Result
			}
		}
		c.mu.Unlock()
	}
}
