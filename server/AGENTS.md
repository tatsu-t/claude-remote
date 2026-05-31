<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-05-30 | Updated: 2026-05-30 -->

# server/

## Purpose

SSH forced command として呼び出されるサーバーデーモン。
`claude-remote-server <command> [args]` 形式で実行され、stdin/stdout で通信する。

## Key Files

| File | Description |
|------|-------------|
| `main.go` | エントリーポイント。`SSH_ORIGINAL_COMMAND` または argv を解析してハンドラーへディスパッチ |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `handler/` | `upload`, `attach`, `pull`, `list` の各コマンドハンドラー |
| `workspace/` | `instances/{id}/` ディレクトリ管理・`claude --server-mode` 起動 |

## For AI Agents

### Working In This Directory

- サーバーバイナリのビルド: `go build -o claude-remote-server ./server`
- 環境変数 `CLAUDE_REMOTE_DATA_DIR` でデータディレクトリを変更可能 (デフォルト: `/var/lib/claude-remote`)
- `workspace.Manager` はすべてのハンドラーで共有される

### Server Storage Layout

```
/var/lib/claude-remote/
└── instances/
    └── {instance-id}/
        ├── meta.json      ← instance.Instance の JSON
        ├── repo/          ← 将来の git bare clone 用 (現在未使用)
        ├── workspace/     ← 展開された作業ディレクトリ
        └── artifacts/
            └── claude.log ← Claude の stdout/stderr ログ
```

### Testing Requirements

- Docker コンテナ上で実際の SSH 接続テストを行う
- `workspace.StartClaude()` は `claude` バイナリが PATH 上にある環境でのみ動作

<!-- MANUAL: -->
