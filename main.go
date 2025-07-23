package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/andybalholm/brotli"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	// Load and decompress Brotli-compressed WASM file
	fmt.Println("Loading Brotli-compressed WASM file...")
	wasmBytes, err := loadAndDecompressWasm("lib/treesitter.wasm.br")
	if err != nil {
		return fmt.Errorf("failed to load and decompress WASM: %w", err)
	}

	fmt.Printf("WASM file size: %d bytes\n", len(wasmBytes))

	// Create wazero runtime
	runtime := wazero.NewRuntime(ctx)
	defer runtime.Close(ctx)

	// Instantiate WASI (if needed)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		return fmt.Errorf("failed to instantiate WASI: %w", err)
	}

	// Define env module (required by Tree-sitter)
	envModuleBuilder := runtime.NewHostModuleBuilder("env")

	// Add commonly required WASM functions
	envModuleBuilder.NewFunctionBuilder().
		WithName("abort").
		WithFunc(func(ctx context.Context) {
			// abort function - usually does nothing
			fmt.Println("WASM abort called")
		}).
		Export("abort")

	envModuleBuilder.NewFunctionBuilder().
		WithName("__assert_fail").
		WithFunc(func(ctx context.Context, assertion, file, line, function uint32) {
			// Assertion failure - usually does nothing
			fmt.Printf("WASM assert failed: assertion=%d, file=%d, line=%d, function=%d\n", assertion, file, line, function)
		}).
		Export("__assert_fail")

	envModuleBuilder.NewFunctionBuilder().
		WithName("tree_sitter_log_callback").
		WithFunc(func(ctx context.Context, logType uint32, message uint32) {
			// Tree-sitter log callback - usually does nothing
			fmt.Printf("Tree-sitter log: type=%d, message=%d\n", logType, message)
		}).
		Export("tree_sitter_log_callback")

	envModuleBuilder.NewFunctionBuilder().
		WithName("tree_sitter_parse_callback").
		WithFunc(func(ctx context.Context, payload uint32, bytes uint32, offset uint32, position uint32, length uint32) {
			// Tree-sitter parse callback - usually does nothing
			fmt.Printf("Tree-sitter parse: payload=%d, bytes=%d, offset=%d, position=%d, length=%d\n", payload, bytes, offset, position, length)
		}).
		Export("tree_sitter_parse_callback")

	envModuleBuilder.NewFunctionBuilder().
		WithName("tree_sitter_progress_callback").
		WithFunc(func(ctx context.Context, payload uint32, progress uint32) uint32 {
			// Tree-sitter progress callback - usually returns 0
			fmt.Printf("Tree-sitter progress: payload=%d, progress=%d\n", payload, progress)
			return 0
		}).
		Export("tree_sitter_progress_callback")

	envModuleBuilder.NewFunctionBuilder().
		WithName("tree_sitter_query_progress_callback").
		WithFunc(func(ctx context.Context, payload uint32) uint32 {
			// Tree-sitter query progress callback - usually returns 0
			fmt.Printf("Tree-sitter query progress: payload=%d\n", payload)
			return 0
		}).
		Export("tree_sitter_query_progress_callback")

	// Add other common C standard library functions
	envModuleBuilder.NewFunctionBuilder().
		WithName("emscripten_resize_heap").
		WithFunc(func(ctx context.Context, size uint32) uint32 {
			// Memory resize - usually does nothing
			fmt.Printf("Emscripten resize heap: size=%d\n", size)
			return 0
		}).
		Export("emscripten_resize_heap")

	// Add Emscripten-required functions to env module
	envModuleBuilder.NewFunctionBuilder().
		WithName("__heap_base").
		WithFunc(func(ctx context.Context) uint32 {
			return 1048576 // 1MB heap base
		}).
		Export("__heap_base")

	envModuleBuilder.NewFunctionBuilder().
		WithName("__data_end").
		WithFunc(func(ctx context.Context) uint32 {
			return 1048576 // Data section end
		}).
		Export("__data_end")

	// Instantiate env module
	if _, err := envModuleBuilder.Instantiate(ctx); err != nil {
		return fmt.Errorf("failed to instantiate env module: %w", err)
	}

	// Compile WASM module
	fmt.Println("Compiling WASM module...")
	module, err := runtime.CompileModule(ctx, wasmBytes)
	if err != nil {
		return fmt.Errorf("failed to compile WASM module: %w", err)
	}
	defer module.Close(ctx)

	// Display module information
	fmt.Println("WASM module compiled successfully!")

	// Display list of exported functions
	fmt.Println("Exported functions:")
	for name, def := range module.ExportedFunctions() {
		fmt.Printf("  - %s: %v\n", name, def)
	}

	// Display list of exported memories
	fmt.Println("Exported memories:")
	for name, def := range module.ExportedMemories() {
		fmt.Printf("  - %s: %v\n", name, def)
	}

	// Instantiate module
	fmt.Println("Instantiating WASM module...")
	instance, err := runtime.InstantiateModule(ctx, module, wazero.NewModuleConfig())
	if err != nil {
		return fmt.Errorf("failed to instantiate WASM module: %w", err)
	}
	defer instance.Close(ctx)

	fmt.Println("WASM module instantiation completed!")

	// Check if basic tree-sitter functions are available
	checkTreeSitterFunctions(instance)

	// Create and test Tree-sitter wrapper
	fmt.Println("\n=== Tree-sitter Wrapper Test ===")
	if err := testTreeSitterWrapper(ctx, instance); err != nil {
		fmt.Printf("Tree-sitter wrapper test failed: %v\n", err)
		// Continue even on error (some WASM features may not be available)
	}

	return nil
}

// loadAndDecompressWasm loads and decompresses a Brotli-compressed WASM file
func loadAndDecompressWasm(filename string) ([]byte, error) {
	// Open file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	// Create Brotli reader
	reader := brotli.NewReader(file)

	// Read decompressed data
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress Brotli file: %w", err)
	}

	return decompressed, nil
}

// checkTreeSitterFunctions checks if the main tree-sitter functions are available
func checkTreeSitterFunctions(instance api.Module) {
	fmt.Println("\nTree-sitter function check:")

	// Check commonly exported tree-sitter functions
	commonFunctions := []string{
		"ts_parser_new_wasm",
		"ts_parser_delete",
		"ts_parser_parse_wasm",
		"ts_tree_delete",
		"ts_tree_root_node_wasm",
		"ts_node_to_string_wasm",
		"malloc",
		"free",
	}

	for _, funcName := range commonFunctions {
		if fn := instance.ExportedFunction(funcName); fn != nil {
			fmt.Printf("  ✓ %s - available\n", funcName)
		} else {
			fmt.Printf("  ✗ %s - not found\n", funcName)
		}
	}

	// Check memory
	if memory := instance.ExportedMemory("memory"); memory != nil {
		fmt.Printf("  ✓ memory - available (size: %d bytes)\n", memory.Size())
	} else {
		fmt.Printf("  ✗ memory - not found\n")
	}
}

// testTreeSitterWrapper tests the basic functionality of the Tree-sitter wrapper
func testTreeSitterWrapper(ctx context.Context, instance api.Module) error {
	// Create Tree-sitter wrapper
	ts, err := NewTreeSitter(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to create TreeSitter wrapper: %w", err)
	}

	fmt.Println("Tree-sitter wrapper created successfully")

	// Try to create a parser
	parser, err := ts.NewParser()
	if err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}
	defer func() {
		if deleteErr := parser.Delete(); deleteErr != nil {
			fmt.Printf("Failed to delete parser: %v\n", deleteErr)
		}
	}()

	fmt.Println("Parser created successfully")

	// Try to parse simple text (without setting a language)
	// Note: In real Tree-sitter usage, a language parser is required, but we're testing basic functionality first
	testText := "hello world"
	fmt.Printf("Attempting to parse text '%s'...\n", testText)

	tree, err := parser.ParseString(testText)
	if err != nil {
		// Error is expected when no language is set
		fmt.Printf("Parse failed (this is expected - no language set): %v\n", err)
		return nil
	}
	defer func() {
		if deleteErr := tree.Delete(); deleteErr != nil {
			fmt.Printf("Failed to delete syntax tree: %v\n", deleteErr)
		}
	}()

	fmt.Println("Parse succeeded!")

	// Try to get the root node
	rootNode, err := tree.RootNode()
	if err != nil {
		fmt.Printf("Failed to get root node: %v\n", err)
		return nil
	}
	defer func() {
		if deleteErr := rootNode.Delete(); deleteErr != nil {
			fmt.Printf("Failed to delete node: %v\n", deleteErr)
		}
	}()

	fmt.Println("Root node retrieved successfully")

	// Try to get the string representation of the node
	nodeString, err := rootNode.String()
	if err != nil {
		fmt.Printf("Failed to get node string representation: %v\n", err)
		return nil
	}

	fmt.Printf("Root node string representation: %s\n", nodeString)

	return nil
}
