# Publishing Guide for kickstart-diff

This guide walks you through publishing your Go CLI tool so others can easily install it.

## Step 1: Prepare Your Repository

### 1.1 Update Module Path

In `go.mod`, change the module name to match your GitHub repository:

```go
module github.com/yourusername/kickstart-diff
```

Then update the import in `main.go`:

```go
import "github.com/yourusername/kickstart-diff/cmd"
```

### 1.2 Update README

Replace `yourname` with your actual GitHub username in README.md.

### 1.3 Update Copyright

Update the copyright notice in:
- `cmd/ksync.go` (line 2)
- `main.go` (line 2)
- `LICENSE`

## Step 2: Push to GitHub

```bash
# Initialize git (if not already done)
git init
git add .
git commit -m "Initial release"

# Create a new repository on GitHub, then:
git remote add origin https://github.com/yourusername/kickstart-diff.git
git branch -M main
git push -u origin main
```

## Step 3: Create a Release

### Option A: Using GoReleaser (Recommended)

1. Install GoReleaser:
   ```bash
   # macOS
   brew install goreleaser
   
   # Linux
   go install github.com/goreleaser/goreleaser/v2@latest
   ```

2. Create a git tag:
   ```bash
   git tag -a v0.1.0 -m "First release"
   git push origin v0.1.0
   ```

3. Create a GitHub token:
   - Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
   - Generate new token with `repo` scope
   - Export it: `export GITHUB_TOKEN=your_token_here`

4. Release:
   ```bash
   goreleaser release --clean
   ```

This will:
- Build binaries for multiple platforms (Linux, macOS, Windows)
- Create a GitHub release
- Upload all binaries
- Generate checksums

### Option B: Manual Release

1. Build for different platforms:
   ```bash
   # Linux
   GOOS=linux GOARCH=amd64 go build -o kickstart-diff-linux-amd64
   
   # macOS (Intel)
   GOOS=darwin GOARCH=amd64 go build -o kickstart-diff-darwin-amd64
   
   # macOS (Apple Silicon)
   GOOS=darwin GOARCH=arm64 go build -o kickstart-diff-darwin-arm64
   
   # Windows
   GOOS=windows GOARCH=amd64 go build -o kickstart-diff-windows-amd64.exe
   ```

2. Create a release on GitHub:
   - Go to your repository
   - Click "Releases" → "Create a new release"
   - Create a tag (e.g., v0.1.0)
   - Upload the binaries
   - Write release notes

## Step 4: Enable `go install`

Once your code is on GitHub with the correct module path, users can install with:

```bash
go install github.com/yourusername/kickstart-diff@latest
```

No additional setup required!

## Step 5: Set Up Homebrew (Optional)

For macOS/Linux users, Homebrew is very popular:

1. Create a tap repository:
   ```bash
   # Create a new repo named homebrew-tap on GitHub
   ```

2. Uncomment the `brews` section in `.goreleaser.yaml` and update the values

3. Next release, GoReleaser will automatically create/update the Homebrew formula

Users can then install with:
```bash
brew tap yourusername/tap
brew install kickstart-diff
```

## Step 6: Promote Your Tool

- Add topics to your GitHub repo: `neovim`, `cli`, `go`, `nvim`, `kickstart`
- Share on Reddit: r/neovim, r/golang
- Share on Twitter/X with #neovim #golang
- Consider adding it to awesome lists

## Distribution Methods Summary

### For Go Developers (Easy)
```bash
go install github.com/yourusername/kickstart-diff@latest
```

### For Everyone (Binary Downloads)
Download from GitHub Releases page

### For Homebrew Users (macOS/Linux)
```bash
brew install yourusername/tap/kickstart-diff
```

### For Scoop Users (Windows)
Create a scoop manifest (see scoop.sh)

## Testing the Installation

After publishing, test the installation:

```bash
# Remove local version
rm -f $(which kickstart-diff)

# Install from GitHub
go install github.com/yourusername/kickstart-diff@latest

# Test
kickstart-diff ksync --help
```

## Versioning

Use semantic versioning (SemVer):
- v0.1.0 - Initial release
- v0.1.1 - Bug fixes
- v0.2.0 - New features (backwards compatible)
- v1.0.0 - Stable API

Create tags for each version:
```bash
git tag -a v0.1.0 -m "Initial release"
git push origin v0.1.0
```

## Continuous Integration (Optional)

Add GitHub Actions for automatic releases:

Create `.github/workflows/release.yml`:
```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

Now just push a tag and GitHub will automatically build and release!

## Next Steps

1. Update module path in `go.mod` and `main.go`
2. Update README with your username
3. Push to GitHub
4. Create your first release
5. Share it with the Neovim community!
