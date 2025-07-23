# go-tree-sitter

A Tree-sitter WASM binding with Brotli compression support.

## Overview

This project provides a **one-command solution** to generate Brotli-compressed Tree-sitter WASM files for web deployment. It automatically downloads the official web-tree-sitter NPM package and compresses the pre-built WASM file using Brotli.

## Features

- 🌳 **Tree-sitter WASM**: Official pre-built Tree-sitter WebAssembly bindings
- 🗜️ **Brotli Compression**: Automatic Brotli compression (67% size reduction)
- ⚡ **One Command**: Single command generates and deploys compressed WASM
- 📦 **NPM Integration**: Downloads from official web-tree-sitter NPM package
- 🔧 **Bazel Native**: Pure Bazel build with http_archive - no vendor directories
- 🚀 **Production Ready**: Optimized for web deployment

## Quick Start

### Prerequisites

- [Bazel](https://bazel.build/) 6.0+

### Build & Generate

**One command to rule them all:**

```bash
bazel build //:generate
```

This command will:

1. 📥 Download `web-tree-sitter-0.25.8.tgz` from NPM
2. 📦 Extract the pre-built `tree-sitter.wasm` file (201KB)
3. 🗜️ Compress it with Brotli to `treesitter.wasm.br` (65KB)
4. 📂 Generate to `lib/treesitter.wasm.br`

## Output

After running the command, you'll have:

```
lib/treesitter.wasm.br    # 65KB compressed WASM file (67% size reduction)
```

### File Size Comparison

| File                 | Size  | Description                       |
| -------------------- | ----- | --------------------------------- |
| `tree-sitter.wasm`   | 201KB | Original WASM from NPM package    |
| `treesitter.wasm.br` | 65KB  | Brotli compressed (67% reduction) |

## Project Structure

```
├── BUILD.bazel                 # Single genrule for WASM compression & generation
├── MODULE.bazel               # External dependencies (web-tree-sitter, brotli)
├── lib/
│   └── treesitter.wasm.br     # 🎯 Final compressed WASM output
├── bazel-bin/                 # Build artifacts (auto-generated)
└── bazel-*                    # Bazel symlinks (auto-generated)
```

## Build Configuration

### Key Dependencies

| Dependency        | Version | Source                 | Purpose              |
| ----------------- | ------- | ---------------------- | -------------------- |
| `web-tree-sitter` | 0.25.8  | NPM Registry           | Pre-built WASM files |
| `brotli`          | 1.1.0   | GitHub                 | Compression          |
| `rules_cc`        | 0.0.9   | Bazel Central Registry | C++ build rules      |

### BUILD.bazel Overview

```starlark
# Single genrule that handles everything:
genrule(
    name = "tree_sitter_wasm_compressed",
    srcs = ["@web_tree_sitter//:wasm_files"],      # NPM package WASM
    outs = ["lib/treesitter.wasm.br"],            # Compressed output
    cmd = "brotli compression command",
    tools = ["@com_google_brotli//:brotli"],
)

# Generate target that copies to source tree
genrule(
    name = "generate",
    srcs = [":tree_sitter_wasm_compressed"],
    outs = ["generate.stamp"],
    cmd = "copy to lib/ directory",
    local = 1,  # Run outside sandbox
)
```

## Performance

### Build Times

- **First run**: ~3 seconds (downloads dependencies)
- **Subsequent runs**: ~0.1 seconds (cached)

### Compression Efficiency

- **Original size**: 201KB
- **Compressed size**: 65KB
- **Compression ratio**: 67% reduction
- **Web performance**: Faster loading, less bandwidth

## Advanced Usage

### Manual Build Steps

If you want to build without auto-generation:

```bash
# Just compress (output to bazel-bin/)
bazel build //:tree_sitter_wasm_compressed

# Then manually copy
cp bazel-bin/lib/treesitter.wasm.br lib/
```

### Customization

To use a different version of web-tree-sitter, update `MODULE.bazel`:

```starlark
http_archive(
    name = "web_tree_sitter",
    urls = ["https://registry.npmjs.org/web-tree-sitter/-/web-tree-sitter-X.Y.Z.tgz"],
    strip_prefix = "package",
    # ... rest of config
)
```

## Comparison with Other Approaches

| Approach              | Pros                                                              | Cons                                                         |
| --------------------- | ----------------------------------------------------------------- | ------------------------------------------------------------ |
| **This Project**      | ✅ One command<br>✅ No toolchain setup<br>✅ Reproducible builds | ❌ Requires Bazel                                            |
| **Manual Download**   | ✅ Simple                                                         | ❌ Manual steps<br>❌ No compression<br>❌ Not automated     |
| **Build from Source** | ✅ Full control                                                   | ❌ Requires Emscripten<br>❌ Complex setup<br>❌ Slow builds |

## Troubleshooting

### Common Issues

**Q: `bazel command not found`**

- Install Bazel: `brew install bazel` (macOS) or see [Bazel installation guide](https://bazel.build/install)

**Q: Build fails with network error**

- Check internet connection
- Try: `bazel clean && bazel build //:generate`

**Q: File not appearing in `lib/`**

- Ensure you're running `bazel build //:generate` (not just `:tree_sitter_wasm_compressed`)
- Check build output for errors

## License

MIT License. Tree-sitter is also licensed under MIT License.

## Contributing

1. Fork the repository
2. Make your changes
3. Test with `bazel build //:generate`
4. Submit a pull request

---

**🎯 Goal**: Provide the easiest way to get a production-ready, compressed Tree-sitter WASM file for web applications.
