# memo

日付ベースのメモ管理CLIツール

## required

- [fzf](https://github.com/junegunn/fzf)

## optional

- [bat](https://github.com/sharkdp/bat)
  - preview の syntax highlighting で使用

## インストール

```bash
go install github.com/watarukura/memo@latest
```

## 使い方

```bash
memo            # 今日のメモを作成・編集
memo list       # メモ一覧を表示（fzfで選択）
memo cd         # メモディレクトリでサブシェルを起動
memo YYYY-MM-DD # 指定日のメモを開く
memo help       # ヘルプを表示
```

## 環境変数

| 変数 | 説明 | デフォルト |
|------|------|------------|
| `MEMO_DIR` | メモの保存ディレクトリ | `~/Documents/memo` |
| `EDITOR` | 使用するエディタ | `vim` |

## ディレクトリ構成

メモは `YYYY/MM/YYYY-MM-DD.md` の形式で保存されます。

```
~/Documents/memo/
├── template.md
├── 2026/
│   └── 03/
│       ├── 2026-03-19.md
│       └── 2026-03-20.md
└── ...
```

## 機能

- テンプレート (`template.md`) から新規メモを自動生成
- 前日のメモと今日のメモを双方向リンクで自動接続
- `fzf` による高速なメモ検索・選択

