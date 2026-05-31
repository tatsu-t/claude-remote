<!-- Generated: 2026-05-30 | Updated: 2026-05-30 -->

# Claude Code Remote Handoff CLI

## Purpose

ローカルの Claude Code CLI 作業を SSH 経由でリモートサーバーへ引き継ぎ、サーバー側で Claude Code セッションを継続実行するツール。
`/upload`, `/attach`, `/pull` の3コマンドを Claude Code slash command として提供し、ロジックは Go CLI バイナリに集約する。

## Key Files

| File | Description |
|------|-------------|
| `go.mod` | Go モジュール定義 (module: github.com/tatsu-t/claude-remote) |
| `仕様書.md` | 設計仕様・対話フロー・設計決定の記録 |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `cmd/claude-remote/` | クライアント CLI バイナリのエントリーポイント (see `cmd/claude-remote/AGENTS.md`) |
| `internal/` | クライアント側の内部パッケージ群 (see `internal/AGENTS.md`) |
| `server/` | サーバーデーモンのエントリーポイントとハンドラー (see `server/AGENTS.md`) |
| `deploy/` | Docker 配布ファイル (Dockerfile, docker-compose.yml, sshd_config) |
| `.claude/commands/` | `/upload`, `/attach`, `/pull` slash command 定義 |

## For AI Agents

### Architecture Overview

```
Local (Claude Code CLI)
  ├── /upload  ─── SSH ──→  claude-remote-server upload
  │                           └── extract tar.gz → start claude --server-mode → stream logs
  ├── /attach  ─── SSH ──→  claude-remote-server attach <id>
  │                           └── stream stored log or live output
  └── /pull    ─── SSH ──→  claude-remote-server pull <id>
                              └── JSON header + tar.gz of workspace
```

### Wire Protocol

**upload**: クライアント → サーバー方向
```
[JSON header line: UploadMeta]\n
[raw tar.gz bytes]
```
サーバー → クライアント方向: JSON Lines (MsgLog / MsgURL / MsgError / MsgDone)

**pull**: サーバー → クライアント方向
```
[JSON header line: PullMeta]\n
[raw tar.gz bytes]
```

**attach**: サーバー → クライアント方向: JSON Lines

### Key Design Decisions

- 通信: SSH のみ (`golang.org/x/crypto/ssh`)
- ファイル選定: `git status --porcelain` → Claude Haiku で絞り込み
- サーバーストレージ: `/var/lib/claude-remote/instances/{id}/`
- Claude 起動: `claude --server-mode --project-dir <workspace>`
- Remote Control URL: stdout の正規表現マッチで取得

### Instance States

```
creating → running → attached → completed
                              → failed
                              → archived
```

### Working In This Directory

- `go build ./...` でビルド確認
- `go test ./...` でテスト実行
- サーバーバイナリは `./server/` から、クライアントは `./cmd/claude-remote/` からビルド

### Testing Requirements

- `go test ./...` でユニットテスト
- Docker で `docker compose up --build` → サーバー起動確認
- SSH 接続テストは実機 or Docker コンテナを使う

### Common Patterns

- エラーは `fmt.Errorf("operation: %w", err)` でラップ
- 対話プロンプトは行動語ベース ("Continue once" / "Exclude large files" など)
- サーバー側エラーは `remote.WriteMessage(w, remote.MsgError, ...)` で返す

## Dependencies

### External
| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI コマンド構造 |
| `golang.org/x/crypto/ssh` | SSH クライアント接続・公開鍵認証 |
| `github.com/anthropics/anthropic-sdk-go` | Claude Haiku によるファイル選定 |
| `github.com/BurntSushi/toml` | 設定ファイル (`~/.claude-remote/config.toml`) |
| `github.com/schollz/progressbar/v3` | アップロードプログレスバー |

<!-- MANUAL: -->
