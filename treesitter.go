package main

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero/api"
)

// TreeSitter is a wrapper around Tree-sitter WASM instance
type TreeSitter struct {
	ctx      context.Context
	instance api.Module
	memory   api.Memory
}

// NewTreeSitter creates a new TreeSitter instance
func NewTreeSitter(ctx context.Context, instance api.Module) (*TreeSitter, error) {
	memory := instance.ExportedMemory("memory")
	if memory == nil {
		return nil, fmt.Errorf("WASM module does not export memory")
	}

	return &TreeSitter{
		ctx:      ctx,
		instance: instance,
		memory:   memory,
	}, nil
}

// Parser represents a Tree-sitter parser
type Parser struct {
	ts      *TreeSitter
	pointer uint32
}

// NewParser creates a new parser
func (ts *TreeSitter) NewParser() (*Parser, error) {
	newParserFn := ts.instance.ExportedFunction("ts_parser_new_wasm")
	if newParserFn == nil {
		return nil, fmt.Errorf("ts_parser_new_wasm function not found")
	}

	results, err := newParserFn.Call(ts.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to call ts_parser_new_wasm: %w", err)
	}

	pointer := uint32(results[0])
	if pointer == 0 {
		return nil, fmt.Errorf("failed to create parser: null pointer returned")
	}

	return &Parser{
		ts:      ts,
		pointer: pointer,
	}, nil
}

// Delete frees the parser's memory
func (p *Parser) Delete() error {
	deleteFn := p.ts.instance.ExportedFunction("ts_parser_delete")
	if deleteFn == nil {
		return fmt.Errorf("ts_parser_delete function not found")
	}

	_, err := deleteFn.Call(p.ts.ctx, api.EncodeU32(p.pointer))
	if err != nil {
		return fmt.Errorf("failed to call ts_parser_delete: %w", err)
	}

	p.pointer = 0
	return nil
}

// SetLanguage sets the parser's language
func (p *Parser) SetLanguage(languagePointer uint32) error {
	setLanguageFn := p.ts.instance.ExportedFunction("ts_parser_set_language")
	if setLanguageFn == nil {
		return fmt.Errorf("ts_parser_set_language function not found")
	}

	results, err := setLanguageFn.Call(p.ts.ctx, api.EncodeU32(p.pointer), api.EncodeU32(languagePointer))
	if err != nil {
		return fmt.Errorf("failed to call ts_parser_set_language: %w", err)
	}

	success := results[0] != 0
	if !success {
		return fmt.Errorf("failed to set language")
	}

	return nil
}

// ParseString parses a string and returns a syntax tree
func (p *Parser) ParseString(text string) (*Tree, error) {
	// Write string to WASM memory
	textPtr, err := p.ts.allocateString(text)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate string: %w", err)
	}
	defer p.ts.free(textPtr)

	parseStringFn := p.ts.instance.ExportedFunction("ts_parser_parse_wasm")
	if parseStringFn == nil {
		return nil, fmt.Errorf("ts_parser_parse_wasm function not found")
	}

	results, err := parseStringFn.Call(
		p.ts.ctx,
		api.EncodeU32(p.pointer),         // parser
		api.EncodeU32(0),                 // old_tree (null)
		api.EncodeU32(textPtr),           // string
		api.EncodeU32(uint32(len(text))), // length
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call ts_parser_parse_string: %w", err)
	}

	treePointer := uint32(results[0])
	if treePointer == 0 {
		return nil, fmt.Errorf("failed to parse string: null tree returned")
	}

	return &Tree{
		ts:      p.ts,
		pointer: treePointer,
	}, nil
}

// Tree represents a syntax tree
type Tree struct {
	ts      *TreeSitter
	pointer uint32
}

// Delete frees the syntax tree's memory
func (t *Tree) Delete() error {
	deleteFn := t.ts.instance.ExportedFunction("ts_tree_delete")
	if deleteFn == nil {
		return fmt.Errorf("ts_tree_delete function not found")
	}

	_, err := deleteFn.Call(t.ts.ctx, api.EncodeU32(t.pointer))
	if err != nil {
		return fmt.Errorf("failed to call ts_tree_delete: %w", err)
	}

	t.pointer = 0
	return nil
}

// RootNode gets the syntax tree's root node
func (t *Tree) RootNode() (*Node, error) {
	rootNodeFn := t.ts.instance.ExportedFunction("ts_tree_root_node_wasm")
	if rootNodeFn == nil {
		return nil, fmt.Errorf("ts_tree_root_node_wasm function not found")
	}

	// TSNode is typically a 32-byte struct (4 pointers + 2 uint32s)
	nodeStructPtr, err := t.ts.malloc(32)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate node struct: %w", err)
	}

	_, err = rootNodeFn.Call(t.ts.ctx, api.EncodeU32(t.pointer), api.EncodeU32(nodeStructPtr))
	if err != nil {
		t.ts.free(nodeStructPtr)
		return nil, fmt.Errorf("failed to call ts_tree_root_node: %w", err)
	}

	return &Node{
		ts:      t.ts,
		pointer: nodeStructPtr,
	}, nil
}

// Node represents a syntax tree node
type Node struct {
	ts      *TreeSitter
	pointer uint32
}

// Delete frees the node's memory
func (n *Node) Delete() error {
	return n.ts.free(n.pointer)
}

// String gets the string representation of the node
func (n *Node) String() (string, error) {
	nodeStringFn := n.ts.instance.ExportedFunction("ts_node_to_string_wasm")
	if nodeStringFn == nil {
		return "", fmt.Errorf("ts_node_to_string_wasm function not found")
	}

	results, err := nodeStringFn.Call(n.ts.ctx, api.EncodeU32(n.pointer))
	if err != nil {
		return "", fmt.Errorf("failed to call ts_node_string: %w", err)
	}

	stringPtr := uint32(results[0])
	if stringPtr == 0 {
		return "", nil
	}

	// Read string (simple implementation here)
	str, err := n.ts.readCString(stringPtr)
	if err != nil {
		return "", fmt.Errorf("failed to read string: %w", err)
	}

	// Free the returned string's memory
	n.ts.free(stringPtr)

	return str, nil
}

// allocateString allocates a string in WASM memory
func (ts *TreeSitter) allocateString(s string) (uint32, error) {
	bytes := []byte(s)
	ptr, err := ts.malloc(uint32(len(bytes) + 1)) // +1 for null terminator
	if err != nil {
		return 0, err
	}

	if !ts.memory.Write(ptr, bytes) {
		ts.free(ptr)
		return 0, fmt.Errorf("failed to write string to memory")
	}

	// Null terminator
	if !ts.memory.WriteByte(ptr+uint32(len(bytes)), 0) {
		ts.free(ptr)
		return 0, fmt.Errorf("failed to write null terminator")
	}

	return ptr, nil
}

// readCString reads a null-terminated string from WASM memory
func (ts *TreeSitter) readCString(ptr uint32) (string, error) {
	if ptr == 0 {
		return "", nil
	}

	// Check memory bounds
	size := ts.memory.Size()
	if ptr >= size {
		return "", fmt.Errorf("pointer out of bounds")
	}

	// Find null terminator to determine string length
	var length uint32
	for i := ptr; i < size; i++ {
		b, ok := ts.memory.ReadByte(i)
		if !ok {
			return "", fmt.Errorf("failed to read byte at %d", i)
		}
		if b == 0 {
			break
		}
		length++
	}

	if length == 0 {
		return "", nil
	}

	// Read string
	bytes, ok := ts.memory.Read(ptr, length)
	if !ok {
		return "", fmt.Errorf("failed to read string")
	}

	return string(bytes), nil
}

// malloc allocates memory of the specified size from WASM memory
func (ts *TreeSitter) malloc(size uint32) (uint32, error) {
	mallocFn := ts.instance.ExportedFunction("malloc")
	if mallocFn == nil {
		return 0, fmt.Errorf("malloc function not found")
	}

	results, err := mallocFn.Call(ts.ctx, api.EncodeU32(size))
	if err != nil {
		return 0, fmt.Errorf("failed to call malloc: %w", err)
	}

	ptr := uint32(results[0])
	if ptr == 0 {
		return 0, fmt.Errorf("malloc returned null pointer")
	}

	return ptr, nil
}

// free frees WASM memory
func (ts *TreeSitter) free(ptr uint32) error {
	if ptr == 0 {
		return nil
	}

	freeFn := ts.instance.ExportedFunction("free")
	if freeFn == nil {
		return fmt.Errorf("free function not found")
	}

	_, err := freeFn.Call(ts.ctx, api.EncodeU32(ptr))
	if err != nil {
		return fmt.Errorf("failed to call free: %w", err)
	}

	return nil
}
