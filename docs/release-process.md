# Release Process

This document describes the automated release process for jmux.

## Overview

The release system provides multiple ways to create releases:

1. **Interactive releases** - Manual process with confirmation prompts
2. **Automated releases** - Fully automated without prompts
3. **Preview releases** - Dry-run to see what would happen
4. **GitHub Actions** - Cloud-based automated releases

## Local Release Scripts

### Main Release Script

The primary release script is `scripts/release.sh`:

```bash
# Interactive release (prompts for confirmation)
./scripts/release.sh

# Automated release (no prompts)
./scripts/release.sh --auto

# Preview what would be released
./scripts/release.sh --dry-run

# Show help
./scripts/release.sh --help
```

### Quick Release Helper

For convenience, use `scripts/quick-release.sh`:

```bash
# Preview mode
./scripts/quick-release.sh preview

# Automated mode  
./scripts/quick-release.sh auto

# Interactive mode (default)
./scripts/quick-release.sh
```

## Release Process Steps

The automated release process performs these steps:

1. **Validation**
   - Checks for git repository
   - Ensures no uncommitted changes
   - Validates current version format

2. **Version Management**
   - Determines release version (removes -dev suffix)
   - Updates version.go with release info
   - Includes git commit hash and build time

3. **Binary Building**
   - Builds static binary for maximum portability
   - Generates SHA256 checksum
   - Verifies binary functionality

4. **Git Operations**
   - Commits the release version
   - Creates annotated git tag
   - Pushes tag to GitHub

5. **GitHub Release**
   - Creates GitHub release with binaries
   - Generates comprehensive release notes
   - Includes installation instructions

6. **Development Setup**
   - Increments version for next dev cycle
   - Commits development version
   - Optionally pushes to main branch

## Environment Variables

Control release behavior with these variables:

- `AUTO_RELEASE=true` - Skip confirmation prompts
- `AUTO_PUSH=true` - Automatically push development version
- `DRY_RUN=true` - Preview mode (equivalent to --dry-run)

## GitHub Actions

Cloud-based releases using GitHub Actions:

1. Go to the Actions tab in GitHub
2. Select "Automated Release" workflow
3. Click "Run workflow"
4. Choose options:
   - Release type: auto/patch/minor/major
   - Dry run: preview changes without releasing

## Version Numbering

jmux follows semantic versioning:

- `vX.Y.Z` - Release versions
- `vX.Y.Z-dev` - Development versions

The release script automatically:
- Converts `v1.4.1-dev` → `v1.4.1` (release)
- Then creates `v1.4.2-dev` (next development)

## Best Practices

### Before Releasing

1. Ensure all features are tested
2. Update documentation if needed
3. Run the release preview: `./scripts/quick-release.sh preview`
4. Verify the proposed release notes

### During Release

- Use `--auto` mode for CI/CD pipelines
- Use interactive mode for manual releases
- Always verify the GitHub release was created successfully

### After Release

1. Test the auto-update functionality: `dmux update`
2. Verify binary downloads work correctly
3. Check that the next development version is set

## Troubleshooting

### Common Issues

**"Not in a git repository"**
- Ensure you're running from the project root
- Check that `.git` directory exists

**"You have uncommitted changes"**
- Commit or stash changes before releasing
- Use `git status` to see what needs to be committed

**"Build failed"**
- Check Go installation and version
- Ensure all dependencies are available
- Verify the code compiles: `go build`

**"GitHub CLI not found"**
- Install GitHub CLI: `apt install gh` or `brew install gh`
- Authenticate: `gh auth login`

### Manual Recovery

If a release fails partway through:

1. Check git tags: `git tag -l`
2. Delete problematic tag: `git tag -d vX.Y.Z`
3. Reset version file: `git checkout HEAD -- src/jmux-go/internal/version/version.go`
4. Try release again

## Security

- Releases are signed with SHA256 checksums
- Static binaries avoid dependency vulnerabilities
- GitHub releases use authenticated API calls
- All commits include verified authorship

## Examples

### Quick Development Release

```bash
# Preview what will be released
./scripts/quick-release.sh preview

# If satisfied, create the release
./scripts/quick-release.sh auto
```

### Manual Release with Review

```bash
# Interactive mode with confirmation prompts
./scripts/release.sh

# Follow the prompts to review and confirm
```

### CI/CD Integration

```bash
# In CI pipeline
AUTO_RELEASE=true AUTO_PUSH=true ./scripts/release.sh --auto
```