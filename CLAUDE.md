# apkgo

CLI tool for uploading APK files to Chinese Android app stores. All output is structured JSON on stdout; logs go to stderr.

## Install

```bash
go install github.com/KevinGong2013/apkgo@latest
# or download binary from https://github.com/KevinGong2013/apkgo/releases
```

## Commands

```bash
apkgo init [-s store1,store2] [-c config.yaml]   # Generate config file
apkgo upload -f <apk> [flags]                     # Upload APK to stores
apkgo doctor [-s stores] [-f apk | -p package]    # Diagnose store credentials/permissions
apkgo audit [-f apk | -p package] [-s stores] [--watch]  # Query review (审核) status
apkgo stores                                      # List stores and config schema (JSON)
apkgo version                                     # Version info (JSON)
```

### Review status (`apkgo audit`)

Upload finishes at **submitted (审核中)** — it does not block waiting for the
review outcome (tencent's old in-upload audit poll was removed). Poll review
progress separately with `apkgo audit -p <package>` (or `-f <apk>`), which runs
on its own context like `doctor`. `--watch [--interval 30s]` loops until every
store reaches a terminal state (approved / approved_first / rejected /
withdrawn) or the global `-t` timeout. Each store's status is normalised to a
unified `state` set:
`reviewing`, `approved`, `approved_first`, `needs_fix`, `rejected`,
`withdrawn`, `unknown`.
`approved_first` means the review passed and that store can be judged to have
first-time shelf / no existing on-shelf version signals, so it is terminal
like `approved`; `needs_fix` means the store explicitly exposed a "整改"
style state and remains non-terminal for `--watch`. Supported: **tencent,
huawei, honor, vivo, oppo, samsung, xiaomi** (stores with a review-status API
or Xiaomi version inference; others report "audit not supported").

`apkgo audit` also reports a separate `listing` dimension for whether the app
is on shelf. `listing` is orthogonal to `state`: `on_shelf` (在架),
`off_shelf` (下架), `not_listed` (未上架), `unknown`. The text renderer shows
this as a leading column before the review state; JSON output already includes
the `listing` field through `AuditStoreResult`.
Listing precision varies by store: Huawei reports all three states directly,
except `releaseState=9` (下架审核不通过, a rejected takedown request), which is
inferred from the on-shelf version signal and degrades to unknown when that
signal is absent; vivo maps `saleStatus` 0/1/2 to not-listed/on-shelf/off-shelf;
OPPO prefers `audit_status` 0/111/222 and falls back to conservative label
matching; Samsung maps direct content states and uses a SALE probe for
approved versions; Honor weakly distinguishes not-listed/on-shelf from
empty/non-empty `releaseInfo` (a missing `releaseInfo` key degrades to
unknown) but cannot identify off-shelf. Xiaomi only distinguishes on-shelf vs
not-listed, and Tencent uses its public detail page as a best-effort
three-state signal.
Missing fields, unexpected values, and probe/scrape failures degrade to
`unknown` rather than inventing a business state.

`needs_fix` is currently produced by OPPO (整改/冻结 labels) and Samsung
(`SUSPENDED` / `*_SUSPENDED`) and remains non-terminal for `--watch`.
`approved_first` is currently supported by Huawei, Tencent, Honor, vivo, and
Samsung. Honor only emits it for a releaseId-scoped review result; without a
releaseId it reports live version/listing only and leaves review `state` empty.

## Upload flags

```
-f, --file         APK or AAB file path (required; .aab is googleplay-only)
    --file64       64-bit APK for split-arch uploads
-s, --store        Comma-separated store names (default: all configured)
-n, --notes        Release notes text
    --notes-file   Read release notes from file (overrides --notes)
    --release-time Schedule a timed release (定时发布) at an RFC3339 time, e.g. 2026-06-20T10:00:00+08:00
    --dry-run      Validate without uploading
-t, --timeout      Global timeout (default: 10m)
-c, --config       Config file path (default: apkgo.yaml)
-o, --output       Output format: json or text (default: json)
```

### Scheduled release (`--release-time`)

Schedules a timed release instead of going live immediately after review.
Value is RFC3339 **with a timezone offset** and must be in the future.
Supported stores: **huawei, honor, xiaomi, oppo, vivo, samsung, tencent**
(see `supports_scheduled_release` in `apkgo stores`). Stores that can't
schedule (googleplay, pgyer, fir, script) log a warning and release
immediately. Each store maps the instant to its own field/format
internally — epoch-based stores use the absolute instant; oppo/vivo/samsung
render it in Beijing time (UTC+8).

### Download mode (URL pass-through)

When `-f` (or `--file64`) is a **public** http(s) URL, stores that support
it pull the APK straight from your OSS instead of apkgo re-uploading the
bytes — faster, especially for large APKs or cloud runs. Supported stores:
**huawei, honor, vivo** (see `supports_url_push` in `apkgo stores`); the
others always upload. apkgo still fetches the APK once locally for metadata.

- The URL must be reachable **without auth** (the store GETs it directly).
  Passing `--fetch-header` (auth) makes apkgo upload instead of passing the
  URL through.
- These flows are **asynchronous**: the store downloads in the background
  and apkgo polls until it finishes. Each store has its own download
  interface (huawei `app-package-file/by-url`, honor `upload-by-url`, vivo
  `app.update.app` + `app.query.task.status`).
- **honor** throttles its status poll to ~once/3min, so it only URL-pushes
  when the APK is at least `url_push_min_mb` MB (default 100); smaller APKs
  upload directly. huawei and vivo URL-push whenever the source is a URL.

## Supported stores

huawei, xiaomi, oppo, vivo, honor, tencent, googleplay, samsung, pgyer, fir, script

## Configuration

YAML file (`apkgo.yaml`) or environment variables (`APKGO_<STORE>_<KEY>`):

```yaml
stores:
  huawei:
    service_account: ""        # recommended; raw JSON or base64
    service_account_file: ""   # alternative; path to JSON credential file
    client_id: ""              # legacy API key (deprecated by Huawei)
    client_secret: ""          # legacy API key
    app_id: ""                 # optional, auto-detected from package name
  xiaomi:
    email: ""          # required, developer account email
    private_key: ""    # required, the value Xiaomi's SDK calls "password"
    cert: ""           # required (one of cert / cert_file); raw PEM or base64
    cert_file: ""      # path to public-key certificate downloaded from dev.mi.com
  oppo:
    client_id: ""      # required
    client_secret: ""  # required
  vivo:
    access_key: ""     # required
    access_secret: ""  # required
  honor:
    client_id: ""      # required
    client_secret: ""  # required
    app_id: ""         # required
  tencent:
    user_id: ""        # required, from open.qq.com
    access_secret: ""  # required, API access secret
    app_id: ""         # single-app default; required if app_id_map is empty
    app_id_map: ""     # multi-app: JSON string like '{"com.foo":"111","com.bar":"222"}'; map wins over app_id when key matches
    package_name: ""   # optional; auto-detected from APK if omitted
  script:
    command: "./deploy.sh"  # required, shell command or script path

  # Multiple script instances via "script.<name>" prefix:
  script.cdn-upload:
    command: "./upload-cdn.sh"
  script.dingtalk:
    command: "./notify-dingtalk.sh"
```

Env var example: `APKGO_HUAWEI_SERVICE_ACCOUNT=$(base64 -w0 huawei-sa.json) apkgo upload -f app.apk --store huawei`

## Hooks

Shell commands executed before/after uploads. Receive context as JSON on stdin.

### Configuration

```yaml
hooks:
  before: "./scripts/before-all.sh"   # runs before any upload
  after: "./scripts/after-all.sh"     # runs after all uploads

stores:
  huawei:
    client_id: "..."
    before: "./scripts/before-huawei.sh"  # runs before this store
    after: "./scripts/after-huawei.sh"    # runs after this store
```

### Protocol

**Exit codes:**
- `0` — success (continue)
- non-zero — failure (`before` hooks abort the upload; `after` hooks log warning only)

**Environment variables** (set automatically):
- `APKGO_STORE` — store name (empty for global hooks)
- `APKGO_PACKAGE` — package name (e.g. `com.example.app`)
- `APKGO_VERSION` — version name (e.g. `1.2.0`)

**Errors:** stderr is captured as the error message.

### Stdin JSON schemas

**Global before** (`hooks.before`):
```json
{
  "file_path": "/path/to/app.apk",
  "apk": {"package": "com.example.app", "version_name": "1.0.0", "version_code": 1, "app_name": "MyApp"},
  "stores": ["huawei", "xiaomi"]
}
```

**Global after** (`hooks.after`):
```json
{
  "file_path": "/path/to/app.apk",
  "apk": {"package": "com.example.app", "version_name": "1.0.0", "version_code": 1, "app_name": "MyApp"},
  "results": [
    {"store": "huawei", "success": true, "duration_ms": 12300},
    {"store": "xiaomi", "success": false, "error": "auth failed", "duration_ms": 400}
  ]
}
```

**Per-store before** (`stores.<name>.before`):
```json
{
  "file_path": "/path/to/app.apk",
  "apk": {"package": "com.example.app", "version_name": "1.0.0", "version_code": 1, "app_name": "MyApp"},
  "store": "huawei"
}
```

**Per-store after** (`stores.<name>.after`):
```json
{
  "file_path": "/path/to/app.apk",
  "apk": {"package": "com.example.app", "version_name": "1.0.0", "version_code": 1, "app_name": "MyApp"},
  "store": "huawei",
  "result": {"store": "huawei", "success": true, "duration_ms": 12300}
}
```

## Output format

stdout is always parseable JSON:

```json
{
  "apk": {"package": "com.example", "version_name": "1.0.0", "version_code": 1, "app_name": "MyApp"},
  "results": [
    {"store": "huawei", "success": true, "duration_ms": 12300},
    {"store": "xiaomi", "success": false, "error": "auth: invalid private key", "duration_ms": 400}
  ]
}
```

## Exit codes

- **0**: All uploads succeeded
- **1**: Some uploads failed (partial success)
- **2**: All uploads failed
- **3**: Input/config error

## Typical agent workflow

```bash
# 1. Check if apkgo is installed
which apkgo

# 2. Generate config for needed stores
apkgo init --store huawei,xiaomi -c apkgo.yaml

# 3. Discover required config fields
apkgo stores

# 4. Dry-run to validate
apkgo upload -f app.apk --dry-run

# 5. Upload
apkgo upload -f app.apk --notes "v1.0.0 release" --timeout 15m

# 6. Parse JSON result from stdout, check exit code
```

## Project structure

```
cmd/           CLI commands (cobra)
pkg/store/     Store interface + implementations (self-registering via init())
pkg/config/    YAML config + env var loading
pkg/apk/       APK metadata parser
pkg/uploader/  Concurrent upload orchestrator
```

Adding a new store: create `pkg/store/<name>/<name>.go`, implement `store.Store` interface, call `store.Register()` in `init()`. Zero changes to existing code.

## 团队规范使用时机

本仓库已引入 `context/team` 下的团队规范，执行相关任务时按需查阅：

- 修改 Git 工作流、提交信息、Tag、更新日志或 PR 内容前，先阅读 `context/team/engineering/`。
- 编写或评审 Java、MySQL、前端代码时，先阅读 `context/team/coding/` 下对应语言或技术栈规范。
- 当前 apkgo 主体是 Go CLI 项目；未覆盖 Go 的团队规范不应强行套用，仍以本文件的项目说明、现有 Go 代码风格和仓库实际构建测试方式为准。
