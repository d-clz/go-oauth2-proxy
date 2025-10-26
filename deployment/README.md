# Deployment Guide

This directory contains deployment configurations for the Go OAuth2 Proxy.

## Files

- `Dockerfile` - Multi-stage Docker build configuration
- `config.yaml` - Runtime configuration (create from config.example.yaml)
- `credentials.json` - GCP service account credentials (you need to provide this)
- `.env.example` - Environment variables template
- `k8s/` - Kubernetes deployment manifests

## Docker Compose Setup

### Prerequisites

1. **GCP Service Account Credentials**
   - Create a service account in GCP with appropriate permissions
   - Download the JSON key file
   - Save it as `deployment/credentials.json`

2. **Configuration File**
   - Copy the example configuration:
     ```bash
     cp ../src/config.example.yaml config.yaml
     ```
   - Update `config.yaml` with your upstream services

### Running with Docker Compose

#### Production (using published image)

```bash
# From repository root
docker-compose up -d

# View logs
docker-compose logs -f oauth2-proxy

# Stop
docker-compose down
```

#### Development (local build)

```bash
# From repository root
docker-compose -f docker-compose.dev.yml up --build

# View logs
docker-compose -f docker-compose.dev.yml logs -f

# Stop
docker-compose -f docker-compose.dev.yml down
```

### Quick Start

1. **Setup credentials and config:**
   ```bash
   # Create config from example
   cp src/config.example.yaml deployment/config.yaml

   # Edit config with your settings
   vim deployment/config.yaml

   # Add your GCP credentials
   cp /path/to/your/credentials.json deployment/credentials.json
   ```

2. **Start the service:**
   ```bash
   docker-compose up -d
   ```

3. **Test the proxy:**
   ```bash
   # Health check
   curl http://localhost:8080/healthz

   # Token info
   curl http://localhost:8080/token-info

   # Proxy request (example)
   curl http://localhost:8080/run_sse
   ```

### Environment Variables

You can override settings using environment variables:

```bash
# Create .env file
cat > .env << EOF
GOOGLE_APPLICATION_CREDENTIALS=/app/credentials.json
LOG_LEVEL=debug
HOST_PORT=8080
EOF

# Start with .env
docker-compose up -d
```

### Volume Mounts

The docker-compose configuration mounts:
- `./deployment/config.yaml` → `/app/config.yaml` (read-only)
- `./deployment/credentials.json` → `/app/credentials.json` (read-only)

### Health Checks

The service includes health checks:
- Endpoint: `http://localhost:8080/healthz`
- Interval: 30s
- Timeout: 10s
- Retries: 3

Check health status:
```bash
docker-compose ps
# or
curl http://localhost:8080/healthz
```

## Docker Run (without compose)

If you prefer to use `docker run` directly:

```bash
docker run -d \
  --name go-oauth2-proxy \
  -p 8080:8080 \
  -v $(pwd)/deployment/config.yaml:/app/config.yaml:ro \
  -v $(pwd)/deployment/credentials.json:/app/credentials.json:ro \
  -e GOOGLE_APPLICATION_CREDENTIALS=/app/credentials.json \
  ghcr.io/d-clz/go-oauth2-proxy:latest
```

## Troubleshooting

### Container won't start
```bash
# Check logs
docker-compose logs oauth2-proxy

# Common issues:
# - Missing credentials.json
# - Invalid config.yaml syntax
# - Wrong file permissions
```

### Health check failing
```bash
# Check if service is listening
docker-compose exec oauth2-proxy wget -O- http://localhost:8080/healthz

# Check configuration
docker-compose exec oauth2-proxy cat /app/config.yaml
```

### Permission issues
```bash
# Ensure files are readable
chmod 644 deployment/config.yaml
chmod 644 deployment/credentials.json
```

## Updating the Image

```bash
# Pull latest image
docker-compose pull

# Restart with new image
docker-compose up -d

# Or in one command
docker-compose pull && docker-compose up -d
```

## Security Notes

- **Never commit `credentials.json` to git** - it's in `.gitignore`
- **Never commit `config.yaml` with sensitive data** - use config.example.yaml as template
- Run containers as non-root user (already configured in Dockerfile)
- Keep credentials files read-only (`:ro` mount flag)
- Regularly update the Docker image to get security patches
