# kickstart-diff

A CLI tool to compare your Neovim plugin configuration with [kickstart.nvim](https://github.com/nvim-lua/kickstart.nvim), helping you discover plugins you might be missing or understand the differences between your setup and the kickstart baseline.

## Features

- Compares local Neovim plugins with kickstart.nvim plugins
- Uses tree-sitter parsing for accurate Lua analysis
- Color-coded diff output showing:
  - Plugins only in kickstart (missing from your config)
  - Plugins only in your config (extras you've added)
  - Shared plugins

## Installation

### Using Go

```bash
go install github.com/WTanardi/kickstart-diff@latest
```

### From Source

```bash
git clone https://github.com/WTanardi/kickstart-diff.git
cd kickstart-diff
go build -o kickstart-diff
```

### Download Binary

Download the latest binary for your platform from the [releases page](https://github.com/WTanardi/kickstart-diff/releases).

## Usage

Compare your Neovim configuration with kickstart.nvim:

```bash
kickstart-diff ksync
```

### Options

```bash
# Specify custom Neovim config directory
kickstart-diff ksync -R /path/to/your/nvim/config

# Enable debug output
kickstart-diff ksync -d

# Show help
kickstart-diff ksync --help
```

### Example Output

```
✗ only in kickstart (3)
  ✗  telescope.nvim/telescope
  ✗  neovim/nvim-lspconfig
  ✗  hrsh7th/nvim-cmp

✓ only in yours (2)
  ✓  tpope/vim-fugitive
  ✓  github/copilot.vim

· shared (15)
  ·  folke/which-key.nvim
  ·  nvim-treesitter/nvim-treesitter
  ...

Summary:  ✗ 3 missing  ✓ 2 extra  · 15 shared
```

## Requirements

- Go 1.26.1 or later (for building from source)
- Neovim configuration directory (defaults to `~/.config/nvim`)

## License

[Add your license here]

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
