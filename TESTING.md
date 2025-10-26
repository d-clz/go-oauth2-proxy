# Local Testing Guide

Complete guide for testing Token Gateway locally before deploying to Kubernetes.

## Prerequisites

✅ Go 1.21+ installed  
✅ Google Cloud service account JSON key  
✅ Service account has Cloud Run Invoker role  
✅ curl installed (for testing)  

## Quick Start

### 1. Navigate to Project

```bash
cd /tmp/token-gateway

# Or copy to your workspace
cp -r /tmp/token-gateway ~/workspace/token-gateway
cd ~/workspace/token-gateway
```

### 2. Update Configuration

Edit `config.yaml` with your actual values:

```yaml
upstreams:
  - name: my-service
    url: https://your-proxy-endpoint.com
    audience: https://your-actual-cloudrun-service.run.app
    timeout: 30
```

### 3. Run Quick Start Script

```bash
./start.sh
```

Or manually:

```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/your-key.json
go run cmd/gateway/main.go -config config.yaml -log-level debug
```

## Test Scenarios

### Scenario 1: Basic Functionality

**Test health check:**
```bash
curl http://localhost:8080/healthz
# Expected: OK
```

**Test metrics (before any requests):**
```bash
curl http://localhost:8080/metrics
```

Expected output:
```json
{
  "tokens_cached": 0,
  "tokens_refreshed": 0,
  "tokens_rejected": 0,
  "tokens_errors": 0,
  "upstreams_count": 1
}
```

### Scenario 2: Token Creation (First Request)

**Make first proxy request:**
```bash
curl -v http://localhost:8080/api/test
```

**Check logs - you should see:**
```
[INFO] Refreshing token audience=https://... state=NEW refresh_count=0
[DEBUG] Token source created audience=https://...
[INFO] New token created audience=https://... expires_at=... valid_for=59m59s
[DEBUG] Proxying request method=GET path=/api/test
[DEBUG] Upstream request method=GET url=https://...
[DEBUG] Upstream response upstream=my-service status=200
[INFO] Request method=GET path=/api/test status=200 duration_ms=...
```

**Check token info:**
```bash
curl http://localhost:8080/token-info | jq
```

Expected:
```json
{
  "total_tokens": 1,
  "upstreams_configured": 1,
  "tokens": [
    {
      "audience": "https://your-service.run.app",
      "state": "CACHED",
      "issued_at": "2025-01-24T12:00:00Z",
      "expires_at": "2025-01-24T13:00:00Z",
      "expires_in": "59m30s",
      "last_used": "2025-01-24T12:00:30Z",
      "refresh_count": 0,
      "rejected_count": 0,
      "error_count": 0
    }
  ]
}
```

### Scenario 3: Token Caching (Subsequent Requests)

**Make multiple requests:**
```bash
for i in {1..5}; do
  curl -s http://localhost:8080/api/test > /dev/null
  echo "Request $i completed"
done
```

**Check logs - should see:**
```
[DEBUG] Token retrieved audience=... state=CACHED expires_in=59m... refresh_count=0
[DEBUG] Proxying request method=GET path=/api/test
```

Note: Token is NOT refreshed - using cached token.

**Check metrics:**
```bash
curl http://localhost:8080/metrics | jq
```

Expected:
```json
{
  "tokens_cached": 1,
  "tokens_refreshed": 0,
  "tokens_rejected": 0,
  "tokens_errors": 0,
  "upstreams_count": 1,
  "oldest_token_age": "2m30s",
  "newest_token_age": "2m30s"
}
```

### Scenario 4: Multiple Upstreams

**Add another upstream to config.yaml:**
```yaml
upstreams:
  - name: service-a
    url: https://proxy-a.com
    audience: https://service-a.run.app
  
  - name: service-b
    url: https://proxy-b.com
    audience: https://service-b.run.app
```

**Restart gateway and test:**
```bash
# Default upstream (first one)
curl http://localhost:8080/api/test

# Specific upstream
curl -H "X-Target-Upstream: service-b" http://localhost:8080/api/test
```

**Check token info - should see 2 tokens:**
```bash
curl http://localhost:8080/token-info | jq '.total_tokens'
# Expected: 2
```

### Scenario 5: Token Refresh

To test token refresh, you have two options:

**Option A: Wait for expiry (55 minutes)**
```bash
# Initial request
curl http://localhost:8080/api/test

# Check token expires_in
curl http://localhost:8080/token-info | jq '.tokens[0].expires_in'
# Output: "59m30s"

# Wait 55 minutes...
# Token will be in EXPIRING state and auto-refresh
```

**Option B: Force refresh (modify code temporarily)**

Edit `internal/token/manager.go`, line where `refreshBeforeExpiry` is set:
```go
// Change from:
refreshBeforeExpiry: time.Duration(refreshBeforeMinutes) * time.Minute,

// To (for testing):
refreshBeforeExpiry: time.Duration(1) * time.Minute,  // Refresh 1 minute before expiry
```

Then tokens refresh after 59 minutes instead of 55.

### Scenario 6: Error Handling

**Test with invalid credentials:**
```bash
export GOOGLE_APPLICATION_CREDENTIALS=/invalid/path.json
go run cmd/gateway/main.go
# Should fail immediately with clear error
```

**Test with wrong audience:**

Edit config.yaml:
```yaml
upstreams:
  - name: test
    url: https://proxy.com
    audience: https://wrong-audience.com  # Invalid
```

```bash
curl http://localhost:8080/api/test
```

**Check logs - you should see:**
```
[ERROR] Failed to get token error=... 
[ERROR] Proxy error upstream=test
```

**Check token state:**
```bash
curl http://localhost:8080/token-info | jq '.tokens[0]'
```

Expected:
```json
{
  "audience": "https://wrong-audience.com",
  "state": "ERROR",
  "error_count": 1,
  "last_error": "oauth2: cannot fetch token: ..."
}
```

### Scenario 7: Token Rejection (401/403)

If upstream rejects the token:

**Check logs:**
```
[WARN] Upstream rejected token audience=... status=401
[WARN] Token rejected by upstream audience=... rejected_count=1
```

**Check token state:**
```bash
curl http://localhost:8080/token-info | jq '.tokens[0]'
```

Expected:
```json
{
  "state": "REJECTED",
  "rejected_count": 1
}
```

**Next request will force token refresh.**

### Scenario 8: Load Testing

**Simple load test:**
```bash
# Install apache bench if needed: apt-get install apache2-utils

# 1000 requests, 10 concurrent
ab -n 1000 -c 10 http://localhost:8080/api/test
```

**Check metrics during/after:**
```bash
watch -n 1 'curl -s http://localhost:8080/metrics | jq'
```

### Scenario 9: Logging Levels

**Test different log levels:**

```bash
# Debug - very verbose
go run cmd/gateway/main.go -log-level debug

# Info - normal (default)
go run cmd/gateway/main.go -log-level info

# Warn - only warnings/errors
go run cmd/gateway/main.go -log-level warn

# Error - only errors
go run cmd/gateway/main.go -log-level error
```

## Monitoring Dashboard (Terminal)

Create a simple monitoring script `monitor.sh`:

```bash
#!/bin/bash

while true; do
    clear
    echo "======================================"
    echo "Token Gateway - Live Monitor"
    echo "======================================"
    echo ""
    
    echo "Health:"
    curl -s http://localhost:8080/healthz
    echo ""
    echo ""
    
    echo "Metrics:"
    curl -s http://localhost:8080/metrics | jq
    echo ""
    
    echo "Token Info:"
    curl -s http://localhost:8080/token-info | jq '.tokens[] | {audience: .audience, state: .state, expires_in: .expires_in, refresh_count: .refresh_count}'
    echo ""
    
    sleep 5
done
```

```bash
chmod +x monitor.sh
./monitor.sh
```

## Troubleshooting

### Gateway won't start

**Check logs for specific error:**
```bash
go run cmd/gateway/main.go -log-level debug 2>&1 | tee gateway.log
```

**Common issues:**

1. **"GOOGLE_APPLICATION_CREDENTIALS environment variable not set"**
   ```bash
   export GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json
   ```

2. **"failed to read config file"**
   ```bash
   # Check file exists
   ls -la config.yaml
   
   # Check YAML syntax
   go run cmd/gateway/main.go -config config.yaml
   ```

3. **"invalid port: 0"**
   - Config is missing `server.port`
   - Add: `server: { port: 8080 }`

### Token creation fails

**Check service account permissions:**
```bash
# Test manually
gcloud auth activate-service-account --key-file=/path/to/key.json
gcloud auth print-identity-token --audiences=https://your-service.run.app

# If this works but gateway doesn't, check file path
```

**Verify audience URL:**
```bash
# Get correct Cloud Run URL
gcloud run services describe YOUR_SERVICE \
    --region=YOUR_REGION \
    --format='value(status.url)'

# Use this EXACT URL in config.yaml
```

### Requests fail with 502

**Check upstream is reachable:**
```bash
curl -v https://your-proxy-endpoint.com/api/test
```

**Check token manually:**
```bash
TOKEN=$(curl -s http://localhost:8080/token-info | jq -r '.tokens[0].token')
curl -H "Authorization: Bearer $TOKEN" https://your-service.run.app
```

## Success Criteria

After testing, you should verify:

- ✅ Gateway starts without errors
- ✅ Health checks return 200
- ✅ First request creates token (state: NEW → CACHED)
- ✅ Subsequent requests use cached token
- ✅ Token auto-refreshes before expiry
- ✅ Multiple upstreams work correctly
- ✅ Errors are logged clearly
- ✅ Metrics show correct counts
- ✅ Token info shows detailed state

## Next Steps

Once local testing passes:

1. Build Docker image: `make docker-build`
2. Test Docker locally: `make docker-run`
3. Deploy to Kubernetes: `kubectl apply -f deployments/k8s/`
4. Monitor in production: `kubectl logs -f -l app=token-gateway`

## Quick Reference

```bash
# Start gateway
go run cmd/gateway/main.go -config config.yaml -log-level debug

# Health check
curl http://localhost:8080/healthz

# Metrics
curl http://localhost:8080/metrics | jq

# Token info
curl http://localhost:8080/token-info | jq

# Test proxy
curl http://localhost:8080/api/test

# Test specific upstream
curl -H "X-Target-Upstream: my-service" http://localhost:8080/api/test

# Build binary
make build

# Run binary
./token-gateway -config config.yaml
```
