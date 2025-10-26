# How to Create Your First GitHub Release

## Problem
You're seeing packages but no releases. The security scan report only appears on **release pages**, not on regular pushes.

## Solution: Create a Release

### Method 1: Using GitHub UI (Easiest)

1. **Go to your repository on GitHub**
   ```
   https://github.com/DUONGHT/go-oauth2-proxy
   ```

2. **Click on "Releases"** (right side of the page)
    - You'll see "No releases published"
    - Click **"Create a new release"**

3. **Create a new tag**
    - Click on "Choose a tag"
    - Type: `v1.0.0` (or your desired version)
    - Click "Create new tag: v1.0.0 on publish"

4. **Fill in release details**
    - **Release title:** `v1.0.0 - Initial Release`
    - **Description:**
      ```markdown
      ## What's New
      - Initial release of OAuth2 Token Gateway
      - Docker image with security scanning
      
      ## Features
      - OAuth2 token management
      - Secure gateway implementation
      ```

5. **Publish release**
    - Check "Set as the latest release"
    - Click **"Publish release"**

6. **Wait for workflow**
    - Go to Actions tab
    - Watch the workflow run (~5 minutes)
    - Security scan will be appended to your release page automatically!

---

### Method 2: Using Git Command Line

```bash
# 1. Create and push a tag
git tag -a v1.0.0 -m "Initial release"
git push origin v1.0.0

# 2. Create release using GitHub CLI (if installed)
gh release create v1.0.0 \
  --title "v1.0.0 - Initial Release" \
  --notes "Initial release with security scanning"

# OR manually create the release on GitHub UI after pushing the tag
```

---

### Method 3: Using GitHub CLI

If you have GitHub CLI installed:

```bash
# Install gh if you don't have it
# Windows: winget install GitHub.cli
# Mac: brew install gh
# Linux: https://github.com/cli/cli/releases

# Login
gh auth login

# Create a release (will also create the tag)
gh release create v1.0.0 \
  --title "v1.0.0 - Initial Release" \
  --notes "## Initial Release

- OAuth2 Token Gateway
- Docker image with automated security scanning
- CI/CD pipeline with vulnerability detection

This release includes full security scanning with Trivy. Check below for the security report."

# The workflow will automatically run and append the security scan!
```

---

## What Happens After Creating a Release

1. **Tag is created:** `v1.0.0`
2. **Workflow triggers:** GitHub Actions starts running
3. **Image is built:** `ghcr.io/duonght/go-oauth2-proxy:v1.0.0`
4. **Security scan runs:** Trivy scans the image
5. **Release is updated:** Security report is automatically appended

### Your release page will look like this:

```markdown
v1.0.0 - Initial Release

## What's New
- Initial release of OAuth2 Token Gateway
- Docker image with security scanning

---

## 🐳 Docker Image

This release is available as a Docker image:

docker pull ghcr.io/duonght/go-oauth2-proxy:v1.0.0

**Image Digest:** `sha256:abc123...`

### 📦 All available tags:
- `ghcr.io/duonght/go-oauth2-proxy:v1.0.0`
- `ghcr.io/duonght/go-oauth2-proxy:1.0`
- `ghcr.io/duonght/go-oauth2-proxy:1`
- `ghcr.io/duonght/go-oauth2-proxy:latest`

### 🚀 Usage:
[Usage instructions]

### 🔒 Security Scan Results

| Severity | Count |
|----------|-------|
| 🔴 Critical | 0 |
| 🟠 High | 0 |
| 🟡 Medium | 2 |
| 🔵 Low | 5 |

✅ No Critical or High severity vulnerabilities detected

<details>
<summary>📋 Click to view detailed vulnerability scan report</summary>
[Full scan results]
</details>
```

---

## Current Workflow Behavior

Your workflow **already runs on push to main**:
- ✅ Builds Docker image
- ✅ Pushes to ghcr.io
- ✅ Scans for vulnerabilities
- ✅ Shows results in workflow summary
- ✅ Uploads to GitHub Security tab

But the security report **only appears on the release page** when you create a release.

---

## Quick Test

Want to test without creating a real release? Just push to main:

```bash
git add .
git commit -m "test: trigger workflow"
git push origin main
```

Check:
1. **Actions tab** - See the security scan in workflow summary
2. **Security tab** - See vulnerabilities as alerts
3. **Artifacts** - Download scan results

Then when you're ready, create a release to see the full report on the release page!

---

## Version Naming Convention

Use semantic versioning:
- `v1.0.0` - Major release
- `v1.1.0` - Minor release (new features)
- `v1.0.1` - Patch release (bug fixes)

Examples:
```bash
git tag v1.0.0  # First stable release
git tag v1.1.0  # Added new features
git tag v1.0.1  # Fixed bugs
git tag v2.0.0  # Breaking changes
```

---

## Troubleshooting

### "Workflow didn't run after creating release"
→ Wait 1-2 minutes, refresh the Actions page

### "Security report not appearing on release"
→ Make sure you **published** the release (not just created a draft)

### "No Docker image tagged with version"
→ Check that your tag starts with `v` (e.g., `v1.0.0`)

### "Release created but workflow failed"
→ Check Actions tab for error logs

---

## Next Steps

1. ✅ Create your first release using Method 1 above
2. ✅ Wait for workflow to complete (~5 minutes)
3. ✅ Check the release page for the security report
4. ✅ Pull your Docker image:
   ```bash
   docker pull ghcr.io/duonght/go-oauth2-proxy:v1.0.0
   ```

---

## Pro Tip

You can also set up **automated releases** using:
- GitHub Actions workflow to auto-release on tag push
- Semantic versioning automation
- Changelog generation

But for now, manual releases are perfect for getting started!