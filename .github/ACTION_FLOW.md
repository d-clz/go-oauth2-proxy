# Workflow Architecture Diagram

## 🔄 Enhanced Docker Build & Scan Pipeline

```
┌─────────────────────────────────────────────────────────────────────┐
│                         GitHub Event Triggers                        │
│  • Push to main branch                                              │
│  • Create tag (v*.*.*)                                              │
│  • Publish release                                                  │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Step 1: Checkout & Setup                          │
│  ┌───────────┐  ┌──────────────┐  ┌────────────────┐               │
│  │ Checkout  │→ │ Setup Docker │→ │ Login to GHCR  │               │
│  │ Code      │  │ Buildx       │  │                │               │
│  └───────────┘  └──────────────┘  └────────────────┘               │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Step 2: Build & Push Image                        │
│  ┌──────────────┐                                                   │
│  │ Extract Tags │                                                   │
│  │ & Metadata   │                                                   │
│  └──────┬───────┘                                                   │
│         │                                                           │
│         ▼                                                           │
│  ┌──────────────────────────────────────────┐                      │
│  │ Build Docker Image                       │                      │
│  │ • Context: ./src                         │                      │
│  │ • Platform: linux/amd64                  │                      │
│  │ • Cache: GitHub Actions cache            │                      │
│  │ • Tags: version, latest, sha             │                      │
│  └──────┬───────────────────────────────────┘                      │
│         │                                                           │
│         ▼                                                           │
│  ┌──────────────────────────────────────────┐                      │
│  │ Push to GitHub Container Registry        │                      │
│  │ Output: Image digest (sha256:...)        │                      │
│  └──────────────────────────────────────────┘                      │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                Step 3: Security Scanning (NEW)                       │
│  ┌────────────────────────────────────────────────────┐             │
│  │ Get Primary Image Tag                              │             │
│  │ Extract: ghcr.io/org/repo:version                  │             │
│  └────────┬───────────────────────────────────────────┘             │
│           │                                                         │
│           ▼                                                         │
│  ┌────────────────────────────────────────────────────┐             │
│  │ Run Trivy Scan - SARIF Format                      │             │
│  │ • Severity: CRITICAL, HIGH, MEDIUM                 │             │
│  │ • Output: trivy-results.sarif                      │             │
│  └────────┬───────────────────────────────────────────┘             │
│           │                                                         │
│           ▼                                                         │
│  ┌────────────────────────────────────────────────────┐             │
│  │ Upload SARIF to GitHub Security                    │             │
│  │ → Appears in Security Tab                          │             │
│  └────────────────────────────────────────────────────┘             │
│           │                                                         │
│           ▼                                                         │
│  ┌────────────────────────────────────────────────────┐             │
│  │ Run Trivy Scan - JSON Format                       │             │
│  │ • Severity: CRITICAL, HIGH, MEDIUM, LOW            │             │
│  │ • Output: trivy-results.json                       │             │
│  │ • Purpose: Machine-readable, parsing               │             │
│  └────────┬───────────────────────────────────────────┘             │
│           │                                                         │
│           ▼                                                         │
│  ┌────────────────────────────────────────────────────┐             │
│  │ Run Trivy Scan - Table Format                      │             │
│  │ • Severity: CRITICAL, HIGH, MEDIUM, LOW            │             │
│  │ • Output: trivy-results.txt                        │             │
│  │ • Purpose: Human-readable report                   │             │
│  └────────────────────────────────────────────────────┘             │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│               Step 4: Process & Report Results (NEW)                 │
│                                                                      │
│  ┌────────────────────────────────────────────────────┐             │
│  │ Parse JSON Results                                 │             │
│  │ • Count by severity (Critical, High, Medium, Low)  │             │
│  │ • Generate summary statistics                      │             │
│  └────────┬───────────────────────────────────────────┘             │
│           │                                                         │
│           ├─────────────┬─────────────┬──────────────┐             │
│           │             │             │              │             │
│           ▼             ▼             ▼              ▼             │
│  ┌────────────┐ ┌──────────────┐ ┌─────────┐ ┌────────────┐      │
│  │ Workflow   │ │ GitHub       │ │ Warning │ │ Upload     │      │
│  │ Summary    │ │ Annotations  │ │ Checks  │ │ Artifacts  │      │
│  └────────────┘ └──────────────┘ └─────────┘ └────────────┘      │
│  • Vuln table  • ::warning::     • Fail if   • 90 day     │      │
│  • Scan details  for critical    critical    retention   │      │
│  • Status emoji  and high issues  (optional)              │      │
└──────────────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│          Step 5: Update Release (On Release Events Only)             │
│                                                                      │
│  ┌────────────────────────────────────────────────────┐             │
│  │ Read Scan Results                                  │             │
│  │ • trivy-results.json (for counts)                  │             │
│  │ • trivy-results.txt (for details)                  │             │
│  └────────┬───────────────────────────────────────────┘             │
│           │                                                         │
│           ▼                                                         │
│  ┌────────────────────────────────────────────────────┐             │
│  │ Build Release Body                                 │             │
│  │ • Docker image info & tags                         │             │
│  │ • Image digest                                     │             │
│  │ • Usage instructions                               │             │
│  │ • Security scan summary table                      │             │
│  │ • Detailed scan report (collapsible)               │             │
│  │ • Build timestamp & scanner info                   │             │
│  └────────┬───────────────────────────────────────────┘             │
│           │                                                         │
│           ▼                                                         │
│  ┌────────────────────────────────────────────────────┐             │
│  │ Append to Release Notes                            │             │
│  │ Using GitHub API                                   │             │
│  └────────────────────────────────────────────────────┘             │
└─────────────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          Final Outputs                               │
│                                                                      │
│  1. Docker Image                                                    │
│     • Published to ghcr.io                                          │
│     • Multiple tags (version, latest, sha)                          │
│     • Image digest recorded                                         │
│                                                                      │
│  2. Security Results                                                │
│     • GitHub Security Tab (SARIF)                                   │
│     • Workflow Summary (visual)                                     │
│     • Artifacts (JSON, TXT, SARIF - 90 days)                        │
│                                                                      │
│  3. Release Notes (if release)                                      │
│     • Docker info with digest                                       │
│     • Usage instructions                                            │
│     • Security scan summary                                         │
│     • Detailed vulnerability report                                 │
│                                                                      │
│  4. Notifications                                                   │
│     • GitHub Actions summary                                        │
│     • Warnings for critical/high vulnerabilities                    │
│     • Optional: Slack/Email (custom)                                │
└─────────────────────────────────────────────────────────────────────┘
```

## 📊 Data Flow

```
                    ┌──────────────┐
                    │  Source Code │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ Docker Build │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ Docker Image │──────┐
                    └──────┬───────┘      │
                           │              │ Push
                           │              ▼
                           │       ┌────────────┐
                           │       │ GHCR       │
                           │       │ Registry   │
                           │       └────────────┘
                           │
                           │ Scan
                           ▼
                    ┌──────────────┐
                    │ Trivy        │
                    │ Scanner      │
                    └──────┬───────┘
                           │
                ┌──────────┼──────────┐
                │          │          │
                ▼          ▼          ▼
         ┌─────────┐  ┌────────┐  ┌─────────┐
         │  SARIF  │  │  JSON  │  │  TABLE  │
         └────┬────┘  └───┬────┘  └────┬────┘
              │           │            │
              │           │            │
              ▼           ▼            ▼
      ┌──────────┐  ┌─────────┐  ┌──────────┐
      │ GitHub   │  │ Parsing │  │ Workflow │
      │ Security │  │ & Logic │  │ Summary  │
      └──────────┘  └────┬────┘  └──────────┘
                         │
                ┌────────┼────────┐
                │        │        │
                ▼        ▼        ▼
          ┌─────────┐ ┌───────┐ ┌────────┐
          │ Release │ │ Warns │ │ Stored │
          │ Notes   │ │       │ │ Files  │
          └─────────┘ └───────┘ └────────┘
```

## 🎭 Workflow States

```
┌────────────┐
│   Start    │
└─────┬──────┘
      │
      ▼
┌─────────────────┐
│ Build Image     │───✅───▶ Success
└─────┬───────────┘
      │
      ❌ Build Failed
      │
      ▼
   [End]


┌────────────┐
│ Scan Image │
└─────┬──────┘
      │
      ├──▶ 0 Critical/High ──✅──▶ Pass
      │
      ├──▶ 1-5 Critical/High ─⚠️─▶ Warning
      │
      └──▶ >5 Critical/High ──❌──▶ Fail (optional)
```

## 🔑 Key Integration Points

```
GitHub Actions Workflow
        │
        ├──▶ Docker Buildx (Build)
        │        │
        │        └──▶ GHCR (Publish)
        │
        ├──▶ Trivy Scanner (Scan)
        │        │
        │        ├──▶ SARIF → GitHub Security
        │        ├──▶ JSON → Parsing
        │        └──▶ Table → Display
        │
        ├──▶ GitHub API (Release Update)
        │
        └──▶ Artifacts Storage (Archive)
```

## 📈 Timeline View

```
Time →

0s    ┌──────────┐
      │ Checkout │
      └────┬─────┘
           │
30s   ┌────▼──────┐
      │   Build   │ ████████████████
      └────┬──────┘
           │
2m    ┌────▼──────┐
      │   Push    │ ███
      └────┬──────┘
           │
2m30s ┌────▼──────┐
      │ Scan SARIF│ ██████
      └────┬──────┘
           │
3m    ┌────▼──────┐
      │ Scan JSON │ ██████
      └────┬──────┘
           │
3m30s ┌────▼──────┐
      │ Scan Table│ ██████
      └────┬──────┘
           │
4m    ┌────▼──────┐
      │  Process  │ ████
      └────┬──────┘
           │
4m30s ┌────▼──────┐
      │  Update   │ ███
      │  Release  │
      └───────────┘

Total: ~4-5 minutes (vs ~2 minutes without scanning)
```

## 🎯 Decision Tree

```
Release Published?
     │
     ├─ Yes ─▶ Full Workflow
     │          ├─ Build
     │          ├─ Push
     │          ├─ Scan
     │          ├─ Report
     │          └─ Update Release ✓
     │
     └─ No ──▶ Build & Scan Only
                ├─ Build
                ├─ Push
                ├─ Scan
                └─ Report

Critical Vulns Found?
     │
     ├─ Yes ─▶ Fail Build? (config)
     │          ├─ Yes ─▶ ❌ Fail
     │          └─ No ──▶ ⚠️  Warn
     │
     └─ No ──▶ ✅ Pass
```

## 🔄 Continuous Improvement Loop

```
┌──────────────┐
│ Build Image  │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Scan Image   │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Find Issues  │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Fix Issues   │
└──────┬───────┘
       │
       └────────┐
                ▼
           [Repeat]
```

---

This diagram shows the complete flow of the enhanced Docker build workflow with security scanning integrated at every step.