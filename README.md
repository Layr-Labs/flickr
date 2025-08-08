# Flickr 🚀

**A comprehensive Docker management tool for Ethereum AVS releases with on-chain ReleaseManager integration.**

Flickr enables operators to push, pull, and run AVS releases stored on-chain, providing a decentralized and trustless way to deploy containerized applications based on Ethereum smart contract state.

## 🌟 Key Features

- **On-chain Release Management**: Push releases to and fetch from Ethereum ReleaseManager contracts
- **Context Management**: Manage multiple environments with different configurations
- **Signer Support**: Sign transactions with ECDSA private keys or keystore files
- **Metadata URI Management**: Set and verify metadata URIs for operator sets
- **Release Operations**: Push, pull, and run releases by ID or latest
- **Digest Verification**: Converts between on-chain bytes32 and Docker sha256 formats
- **AVS Context Injection**: Automatically passes AVS metadata as environment variables
- **Cross-platform**: Supports Linux and macOS (AMD64/ARM64)

## 📦 Installation

### Quick Install (Recommended)
```bash
go install github.com/yourorg/flickr/cmd/flickr@latest
```

### Install from Source
```bash
git clone https://github.com/yourorg/flickr.git
cd flickr
make install
```

### Verify Installation
```bash
flickr --version
```

## ✅ Prerequisites

- **Docker** (latest) - [Install Docker](https://docs.docker.com/engine/install/)
- **Go 1.21+** (for building from source) - [Install Go](https://go.dev/doc/install/)
- **Ethereum RPC endpoint** (archive node recommended for production)

## 🚀 Quick Start

### 1. Create and Configure a Context

```bash
# Create a new context
flickr context create --name mainnet --use

# Configure the context
flickr context set \
  --avs-address 0x1234567890123456789012345678901234567890 \
  --operator-set-id 0 \
  --release-manager 0xabcdef1234567890abcdef1234567890abcdef12 \
  --rpc-url https://eth-mainnet.g.alchemy.com/v2/YOUR-API-KEY \
  --ecdsa-private-key 0xYOUR_PRIVATE_KEY
```

### 2. Set Metadata URI (Required Before Pushing)

```bash
flickr metadata set --uri "https://example.com/avs-metadata.json"
```

### 3. Push a Release

```bash
# Push a Docker image as a release
flickr push --image myregistry.io/myavs:v1.0.0

# Skip Docker push if image is already in registry
flickr push --image myregistry.io/myavs:v1.0.0 --skip-docker-push
```

### 4. Pull a Release

```bash
# Pull the latest release
flickr pull

# Pull a specific release
flickr pull --release-id 3
```

### 5. Run a Release

```bash
# Run the latest release
flickr run

# Run a specific release with custom command
flickr run --release-id 3 --cmd sh --cmd -c --cmd "echo 'Hello from AVS'"
```

## 📋 Command Reference

### Context Management

Manage multiple environments with different configurations:

```bash
# Create a new context
flickr context create --name <name> [--use]

# List all contexts
flickr context list

# Switch to a different context
flickr context use <name>

# Update context settings
flickr context set [options]

# Show current context
flickr context show
```

#### Context Settings

| Setting | Flag | Description |
|---------|------|-------------|
| AVS Address | `--avs-address` | AVS contract address |
| Operator Set | `--operator-set-id` | Operator set ID |
| Release Manager | `--release-manager` | ReleaseManager contract address |
| RPC URL | `--rpc-url` | Ethereum RPC endpoint |
| ECDSA Key | `--ecdsa-private-key` | Hex-encoded private key for signing |
| Keystore | `--keystore-path` | Path to keystore file |
| Keystore Password | `--keystore-password` | Password for keystore |

### Metadata Management

Manage metadata URIs for operator sets:

```bash
# Set metadata URI (required before pushing releases)
flickr metadata set --uri "https://example.com/metadata.json"

# Get current metadata URI
flickr metadata get
```

### Push Command

Push Docker images as on-chain releases:

```bash
flickr push [options]
```

| Flag | Description | Required |
|------|-------------|----------|
| `--image` | Docker image(s) to push | Yes |
| `--upgrade-by-time` | Unix timestamp for upgrade deadline | No (default: 30 days) |
| `--registry` | Override registry URL | No |
| `--skip-docker-push` | Skip Docker push step | No |
| `--gas-limit` | Gas limit for transaction | No (default: 500000) |

### Pull Command

Pull Docker images for releases from the chain:

```bash
flickr pull [options]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--release-id` | Specific release ID | Latest |
| `--all` | Pull all artifacts | First only |

### Run Command

Run releases as Docker containers:

```bash
flickr run [options]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--release-id` | Specific release ID | Latest |
| `--name` | Container name | Auto-generated |
| `--detach`, `-d` | Run in background | false |
| `--env`, `-e` | Additional environment variables | None |
| `--cmd` | Command to run in container | Image default |

## 🔐 Signer Configuration

Flickr supports two types of signers for pushing releases:

### ECDSA Private Key
```bash
flickr context set --ecdsa-private-key 0xYOUR_PRIVATE_KEY
```

### Keystore File
```bash
flickr context set \
  --keystore-path /path/to/keystore.json \
  --keystore-password YOUR_PASSWORD
```

**Note**: Setting one type of signer clears the other (they are mutually exclusive).

## 🏗️ Architecture

### Workflow

1. **Setup Phase**
   - Create and configure context
   - Set metadata URI for operator set
   - Configure signer for transactions

2. **Push Phase**
   - Docker image is pushed to registry (optional)
   - Image digest is extracted
   - Release is created on-chain with artifacts

3. **Pull Phase**
   - Query contract for release information
   - Convert on-chain digests to Docker format
   - Pull Docker images from registry

4. **Run Phase**
   - Fetch release from contract
   - Pull image if not cached
   - Run container with AVS environment variables

### Contract Integration

Flickr integrates with ReleaseManager contracts implementing:

```solidity
interface IReleaseManager {
    struct Artifact {
        string registry;
        bytes32 digest;
    }
    
    struct Release {
        Artifact[] artifacts;
        uint32 upgradeByTime;
    }
    
    struct OperatorSet {
        address avs;
        uint32 id;
    }
    
    function publishMetadataURI(OperatorSet calldata operatorSet, string calldata uri) external;
    function publishRelease(OperatorSet calldata operatorSet, Release calldata release) external returns (uint256);
    function getLatestRelease(OperatorSet memory operatorSet) external view returns (uint256, Release memory);
    function getRelease(OperatorSet memory operatorSet, uint256 releaseId) external view returns (Release memory);
    function getTotalReleases(OperatorSet memory operatorSet) external view returns (uint256);
    function getMetadataURI(OperatorSet memory operatorSet) external view returns (string memory);
}
```

### Environment Variables

Containers receive these AVS context variables:

| Variable | Description |
|----------|-------------|
| `AVS_ADDRESS` | AVS contract address |
| `OPERATOR_SET_ID` | Operator set ID |
| `RELEASE_ID` | Release ID being run |
| `UPGRADE_BY_TIME` | Unix timestamp for upgrade deadline |

## 🔧 Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/yourorg/flickr.git
cd flickr

# Build the binary
make build

# Run tests
make test

# Run integration tests (requires Docker)
make test-integration
```

### Available Make Commands

```bash
make help              # Show all available commands
make build             # Build the binary
make test              # Run all tests
make test-fast         # Run unit tests only
make test-integration  # Run Docker integration tests
make fmt               # Format code
make lint              # Run linter
make install           # Install to ~/bin
make release           # Build all platform binaries
make clean             # Remove built artifacts
```

### Project Structure

```
flickr/
├── cmd/flickr/          # CLI entry point
├── internal/
│   ├── commands/        # CLI commands
│   │   ├── context/     # Context management
│   │   ├── metadata/    # Metadata URI management
│   │   ├── pull/        # Pull releases
│   │   ├── push/        # Push releases
│   │   └── run/         # Run releases
│   ├── config/          # Configuration management
│   ├── controller/      # Main orchestration logic
│   ├── docker/          # Docker operations
│   ├── eth/             # Ethereum client
│   ├── middleware/      # CLI middleware
│   ├── ref/             # Digest/reference utilities
│   └── signer/          # Transaction signing
├── tests/               # Test files
├── Makefile             # Build automation
└── go.mod               # Go dependencies
```

## 🧪 Testing

### Run All Tests
```bash
make test
```

### Run Unit Tests Only
```bash
make test-fast
```

### Run Integration Tests
```bash
# Requires Docker to be running
make test-integration
```

### Run with Coverage
```bash
make coverage
# Opens coverage.html in your browser
```

## 🚢 Example Workflows

### Production Deployment

```bash
# 1. Create production context
flickr context create --name prod --use

# 2. Configure with production settings
flickr context set \
  --avs-address 0xPROD_AVS_ADDRESS \
  --operator-set-id 0 \
  --release-manager 0xPROD_RELEASE_MANAGER \
  --rpc-url https://eth-mainnet.production.com \
  --keystore-path /secure/path/keystore.json \
  --keystore-password $KEYSTORE_PASSWORD

# 3. Set metadata URI
flickr metadata set --uri "https://cdn.example.com/avs-prod-metadata.json"

# 4. Push new release
flickr push --image registry.example.com/avs:v2.0.0

# 5. Run the release
flickr run --name avs-prod --detach
```

### Development Testing

```bash
# 1. Create local context
flickr context create --name local --use

# 2. Configure for local testing
flickr context set \
  --avs-address 0x70997970C51812dc3A010C7d01b50e0d17dc79C8 \
  --operator-set-id 0 \
  --release-manager 0x59c8d715dca616e032b744a753c017c9f3e16bf4 \
  --rpc-url http://localhost:8545 \
  --ecdsa-private-key 0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d

# 3. Set test metadata
flickr metadata set --uri "https://example.com/test-metadata.json"

# 4. Push test image
flickr push --image alpine:latest --skip-docker-push

# 5. Test the release
flickr run --cmd sh --cmd -c --cmd "echo 'Test successful!'"
```

## 🔒 Security Considerations

- **Private Keys**: Use keystore files for production; never commit private keys
- **RPC Security**: Use authenticated, secure RPC endpoints
- **Digest Verification**: Always verify digests match expected images
- **Registry Trust**: Only pull from trusted registries
- **Container Isolation**: Run containers with appropriate security constraints
- **Gas Limits**: Set appropriate gas limits for transactions

## 🐛 Troubleshooting

### No Metadata URI Set
```bash
Error: no metadata URI set for this operator set

Solution:
flickr metadata set --uri "https://your-metadata-uri.json"
```

### No Releases Found
```bash
Error: no releases found for this operator set

Solution:
1. Verify metadata URI is set: flickr metadata get
2. Push a release: flickr push --image your-image:tag
```

### Permission Denied
```bash
Error: execution reverted: unauthorized

Solution:
Ensure your signer address has permission to publish for the AVS
```

### Docker Pull Failed
```bash
Error: pull access denied

Solution:
1. Verify Docker is logged in: docker login
2. Check image exists: docker pull <image>
3. Verify registry format in push command
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- EigenLayer team for the AVS architecture
- Ethereum community for smart contract standards
- Docker team for containerization technology

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/yourorg/flickr/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourorg/flickr/discussions)
- **Security**: Please report security issues to security@yourorg.com

## 🗺️ Current Features

- ✅ Context management for multiple environments
- ✅ ECDSA and keystore signer support
- ✅ Metadata URI management
- ✅ Push releases to chain
- ✅ Pull releases by ID or latest
- ✅ Run releases with custom commands
- ✅ Better error messages and guidance
- ✅ Chain ID-based default contract addresses
- ✅ Parameter inference from context
- ✅ Multi-artifact support (pull with --all flag)

---

**Built with ❤️ for the EigenLayer ecosystem**