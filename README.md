# Flickr 🚀

**A lightweight Docker runner for Ethereum AVS releases managed by on-chain ReleaseManager contracts.**

Flickr enables operators to automatically pull and run AVS releases stored on-chain, providing a decentralized and trustless way to deploy containerized applications based on Ethereum smart contract state.

## 🌟 Key Features

- **On-chain Release Management**: Fetches release information directly from Ethereum ReleaseManager contracts
- **Digest Verification**: Converts on-chain bytes32 digests to Docker-compatible sha256 format
- **AVS Context Injection**: Automatically passes AVS metadata as environment variables
- **Lightweight**: Minimal dependencies, single binary distribution
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

### Manual Binary Download
Download the appropriate binary for your platform from the [releases page](https://github.com/yourorg/flickr/releases).

```bash
# macOS (Apple Silicon)
curl -L https://github.com/yourorg/flickr/releases/download/v0.1.0/flickr-darwin-arm64.tar.gz | tar xz
mv flickr /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/yourorg/flickr/releases/download/v0.1.0/flickr-darwin-amd64.tar.gz | tar xz
mv flickr /usr/local/bin/

# Linux (AMD64)
curl -L https://github.com/yourorg/flickr/releases/download/v0.1.0/flickr-linux-amd64.tar.gz | tar xz
mv flickr /usr/local/bin/

# Linux (ARM64)
curl -L https://github.com/yourorg/flickr/releases/download/v0.1.0/flickr-linux-arm64.tar.gz | tar xz
mv flickr /usr/local/bin/
```

### Verify Installation
```bash
flickr --help
```

## ✅ Prerequisites

- **Docker** (latest) - [Install Docker](https://docs.docker.com/engine/install/)
- **Go 1.21+** (for building from source) - [Install Go](https://go.dev/doc/install/)
- **Ethereum RPC endpoint** (archive node recommended)

## 🚀 Usage

### Basic Usage

Run the latest release for an AVS:
```bash
flickr run \
  --avs 0x1234567890123456789012345678901234567890 \
  --operator-set 1 \
  --release-manager 0xabcdef1234567890abcdef1234567890abcdef12 \
  --rpc-url https://eth-mainnet.g.alchemy.com/v2/YOUR-API-KEY
```

### Run a Specific Release
```bash
flickr run \
  --avs 0x1234567890123456789012345678901234567890 \
  --operator-set 1 \
  --release-manager 0xabcdef1234567890abcdef1234567890abcdef12 \
  --rpc-url https://eth-mainnet.g.alchemy.com/v2/YOUR-API-KEY \
  --release-id 42
```

### Run in Background with Custom Name
```bash
flickr run \
  --avs 0x1234567890123456789012345678901234567890 \
  --operator-set 1 \
  --release-manager 0xabcdef1234567890abcdef1234567890abcdef12 \
  --rpc-url https://eth-mainnet.g.alchemy.com/v2/YOUR-API-KEY \
  --name my-avs-instance \
  --detach
```

### Pass Additional Environment Variables
```bash
flickr run \
  --avs 0x1234567890123456789012345678901234567890 \
  --operator-set 1 \
  --release-manager 0xabcdef1234567890abcdef1234567890abcdef12 \
  --rpc-url https://eth-mainnet.g.alchemy.com/v2/YOUR-API-KEY \
  --env API_KEY=secret \
  --env LOG_LEVEL=debug
```

## 📋 Command Reference

### Required Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--avs` | AVS contract address | `0x1234...7890` |
| `--operator-set` | Operator set ID | `1` |
| `--release-manager` | ReleaseManager contract address | `0xabcd...ef12` |
| `--rpc-url` | Ethereum RPC endpoint URL | `https://eth.example.com` |

### Optional Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--release-id` | Specific release ID to run | Latest release |
| `--name` | Container name | Auto-generated |
| `--detach`, `-d` | Run container in background | `false` |
| `--env`, `-e` | Additional environment variables (KEY=VALUE) | None |

## 🏗️ Architecture

### How It Works

1. **Query Contract**: Flickr queries the ReleaseManager contract for release information
2. **Fetch Release**: Retrieves either the latest release or a specific release ID
3. **Process Artifact**: Takes the first artifact from the release (MVP limitation)
4. **Convert Digest**: Converts the on-chain bytes32 digest to Docker's `sha256:<hex>` format
5. **Build Reference**: Constructs a pullable Docker image reference
6. **Pull Image**: Downloads the Docker image from the registry
7. **Run Container**: Starts the container with AVS context environment variables

### Contract Integration

Flickr expects a ReleaseManager contract implementing:

```solidity
interface IReleaseManager {
    struct Artifact {
        string registry;    // e.g., "ghcr.io/org/image"
        bytes32 digest32;   // sha256 digest as bytes32
    }
    
    struct Release {
        Artifact[] artifacts;
        uint32 upgradeByTime;
    }
    
    function getLatestRelease(address avs, uint32 operatorSetId) 
        external view returns (Release memory, uint64 releaseId);
    
    function getRelease(address avs, uint32 operatorSetId, uint64 releaseId) 
        external view returns (Release memory);
}
```

### Environment Variables

The following variables are automatically injected into containers:

| Variable | Description |
|----------|-------------|
| `AVS_ADDRESS` | The AVS contract address |
| `OPERATOR_SET_ID` | The operator set ID |
| `RELEASE_ID` | The release ID being run |
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
make test-fast         # Run unit tests only (skip integration)
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
│   ├── controller/      # Main orchestration logic
│   ├── docker/          # Docker operations
│   ├── eth/             # Ethereum client
│   └── ref/             # Digest/reference utilities
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

## 🚢 Deployment

### For Operators

1. Ensure Docker is installed and running
2. Install Flickr using one of the methods above
3. Configure your RPC endpoint (archive node recommended)
4. Run Flickr with your AVS parameters

### For AVS Developers

1. Deploy your ReleaseManager contract
2. Register your AVS and operator sets
3. Publish releases with container artifacts
4. Share the contract addresses with operators

## 🔒 Security Considerations

- **Digest Verification**: Always verify the digest matches the expected image
- **Registry Trust**: Only pull from trusted registries
- **RPC Security**: Use secure, authenticated RPC endpoints
- **Container Isolation**: Run containers with appropriate security constraints

## 📝 Configuration

### RPC Endpoints

Flickr requires an Ethereum RPC endpoint. For production use:
- Use an archive node for historical state access
- Consider using authenticated endpoints
- Monitor rate limits and quotas

### Docker Configuration

Ensure Docker daemon is configured with:
- Sufficient disk space for images
- Appropriate memory limits
- Network policies if required

## 🐛 Troubleshooting

### Docker Not Found
```bash
# Verify Docker installation
docker --version

# Start Docker daemon if not running
sudo systemctl start docker  # Linux
open -a Docker               # macOS
```

### RPC Connection Issues
```bash
# Test RPC endpoint
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  YOUR_RPC_URL
```

### Container Fails to Start
```bash
# Check Docker logs
docker logs <container-name>

# Verify image exists
docker images | grep <image-name>
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

## 🗺️ Roadmap

### Current (MVP)
- ✅ Basic contract integration
- ✅ Single artifact support
- ✅ Docker pull and run

### Future Enhancements
- [ ] Multi-artifact support
- [ ] ORAS authentication
- [ ] Signature verification
- [ ] Complex Docker arguments
- [ ] Network presets
- [ ] Configuration files
- [ ] Kubernetes support
- [ ] Metrics and monitoring

---

**Built with ❤️ for the EigenLayer ecosystem**