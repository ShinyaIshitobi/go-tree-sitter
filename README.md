# go-tree-sitter

A Go binding for Tree-sitter with Brotli compression support for WASM output.

## Overview

This project provides Tree-sitter bindings for Go with integrated Brotli compression capabilities. It's designed to generate compressed WASM modules and static libraries optimized for web deployment.

## Features

- 🌳 **Tree-sitter Integration**: Full Tree-sitter parsing library support
- 🗜️ **Brotli Compression**: Automatic compression of build outputs
- 📦 **WASM Ready**: Optimized for WebAssembly deployment
- 🔧 **Bazel Build System**: Modern, scalable build configuration
- 📂 **Go Vendor Structure**: Standard Go dependency management

## Quick Start

### Prerequisites

- [Bazel](https://bazel.build/)
- Go 1.24.5+
- Clang/GCC for C compilation

### Build Commands

```bash
# Test Brotli compression with sample file
bazel build //:test_brotli_compression

# Build compressed Tree-sitter static library
bazel build //:tree_sitter_compressed
```

## Build Targets

| Target                    | Output                          | Description                                    |
| ------------------------- | ------------------------------- | ---------------------------------------------- |
| `test_brotli_compression` | `bazel-bin/test.txt.br`         | Test file for Brotli compression (~30B)        |
| `tree_sitter_compressed`  | `bazel-bin/libtree_sitter.a.br` | Compressed Tree-sitter static library (~5.5KB) |

## Output Files

After running the build commands, compressed files will be available:

```
bazel-bin/
├── test.txt.br              # Sample compressed file
└── libtree_sitter.a.br      # Compressed Tree-sitter library
```

### How to Generate WASM.br Files

To create compressed WASM files:

1. **Build the static library** (current):

   ```bash
   bazel build //:tree_sitter_compressed
   # Generates: bazel-bin/libtree_sitter.a.br
   ```

2. **For WASM output** (requires Emscripten):

   - Add Emscripten toolchain to your MODULE.bazel
   - Create WASM build target using `emcc_binary` rule
   - Apply Brotli compression to the WASM output

   Example WASM target (to be implemented):

   ```starlark
   # Future WASM target
   genrule(
       name = "tree_sitter_wasm_compressed",
       srcs = [":tree_sitter_wasm"],
       outs = ["tree_sitter.wasm.br"],
       cmd = "$(location @com_google_brotli//:brotli) -o $@ $(SRCS)",
       tools = ["@com_google_brotli//:brotli"],
   )
   ```

## Project Structure

```
├── BUILD.bazel                              # Main build configuration
├── MODULE.bazel                            # Bazel module dependencies
├── vendor/                                 # Go-style vendor directory
│   └── github.com/
│       └── tree-sitter/
│           └── tree-sitter/
│               ├── BUILD.bazel             # Tree-sitter build config
│               └── lib/                    # Tree-sitter C library
└── bazel-bin/                              # Build outputs (ignored)
```

## Dependencies

- **Tree-sitter**: v0.25.8 (parsing library)
- **Brotli**: v1.1.0 (compression library)
- **rules_cc**: v0.0.9 (C/C++ build rules)

## Configuration

### MODULE.bazel

- Configures external dependencies via Bzlmod
- Downloads Tree-sitter and Brotli sources
- Sets up C/C++ compilation rules

### BUILD.bazel

- Defines build targets for compression
- Configures Tree-sitter static library generation
- Sets up Brotli compression pipeline

## Development

### Adding New Compression Targets

1. Define your source target in BUILD.bazel
2. Create a corresponding compression rule:
   ```starlark
   genrule(
       name = "your_target_compressed",
       srcs = [":your_target"],
       outs = ["your_file.br"],
       cmd = "$(location @com_google_brotli//:brotli) -o $@ $(SRCS)",
       tools = ["@com_google_brotli//:brotli"],
   )
   ```

### Vendor Management

Tree-sitter sources are managed in Go-standard vendor structure:

- Sources: `vendor/github.com/tree-sitter/tree-sitter/`
- Build config: `vendor/github.com/tree-sitter/tree-sitter/BUILD.bazel`
- Git tracking: Only BUILD.bazel is tracked, sources are ignored

## License

This project is licensed under the MIT License. Tree-sitter is licensed under the MIT License.

## Contributing

1. Fork the repository
2. Create your feature branch
3. Add appropriate build targets and tests
4. Submit a pull request

---

**Note**: This project is optimized for WASM deployment with Brotli compression. For production use, ensure your web server is configured to serve `.br` files with appropriate `Content-Encoding: br` headers.
