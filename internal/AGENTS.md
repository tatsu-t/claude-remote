<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-05-30 | Updated: 2026-05-30 -->

# internal/

## Purpose

クライアント側の内部パッケージ。外部からは直接インポートされない。
サーバー側 (`server/`) と共有するパッケージは `internal/files` と `internal/instance` のみ。

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `cli/` | `/upload`, `/attach`, `/pull` の対話フローと CLI ロジック |
| `config/` | `~/.claude-remote/config.toml` の読み書き |
| `files/` | ファイル選定 (Claude API)・tar.gz パック/アンパック・プロトコルヘッダー |
| `instance/` | インスタンス状態定義・ローカルストア (`~/.claude-remote/instances.json`) |
| `remote/` | SSH クライアント接続・JSON Lines ストリーミング |

## For AI Agents

### Working In This Directory

- `internal/files` と `internal/instance` はサーバー側からも import される
- `internal/remote`, `internal/cli`, `internal/config` はクライアント専用
- `internal/cli/util.go` に共通ヘルパー (readLine, generateID, getRepoInfo など)

### Common Patterns

- エラーは呼び出し元で `fmt.Errorf("context: %w", err)` でラップ
- ユーザー向けの出力は `fmt.Printf`、エラーは `fmt.Fprintf(os.Stderr, ...)`

<!-- MANUAL: -->
