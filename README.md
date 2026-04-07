# BringIt

旅行、キャンプ、イベント、遊びの前に、友達同士で持ち物を分担するためのサービス

## 機能

- **持ち物リスト作成**: タイトルと説明を入力して、共有可能なリストを作成
- **アイテム管理**: 持ち物の追加・削除、担当者の設定、必須/任意の切り替え
- **準備状況のトラッキング**: 各アイテムの「準備済み」状態をトグルしてプログレスを確認
- **共有リンク**: ユニークなトークン付きURLで友達とリストを共有
- **インメモリストレージ**: サーバー再起動まで状態を保持

## 必要な環境

- [Go](https://golang.org/dl/) 1.24 以上

## クイックスタート

```bash
# リポジトリをクローン
git clone https://github.com/mohadayo/bringit.git
cd bringit

# サーバーを起動
go run .
```

ブラウザで http://localhost:8080 を開いてください。

## 環境変数

| 変数名 | デフォルト値 | 説明 |
|--------|-------------|------|
| `PORT` | `8080` | サーバーのリッスンポート |

### 例: ポートを変更して起動

```bash
PORT=3000 go run .
```

## テストの実行

```bash
# すべてのテストを実行
go test ./...

# 詳細出力でテストを実行
go test -v ./...

# カバレッジ付きでテストを実行
go test -v -race -coverprofile=coverage.out ./...

# カバレッジレポートを表示
go tool cover -func=coverage.out

# ブラウザでカバレッジレポートを確認
go tool cover -html=coverage.out
```

## ビルド

```bash
go build -o bringit .
./bringit
```

## プロジェクト構成

```
bringit/
├── main.go          # エントリーポイント、サーバー起動
├── handler.go       # HTTPハンドラー・ルーティング
├── store.go         # インメモリストア（スレッドセーフ）
├── model.go         # データモデル（List, Item）
├── handler_test.go  # HTTPハンドラーのテスト
├── templates/
│   ├── index.html   # トップページテンプレート
│   └── list.html    # リスト詳細ページテンプレート
└── .github/
    └── workflows/
        └── ci.yml   # GitHub Actions CI設定
```

## 技術スタック

- **言語**: Go 1.24
- **サーバー**: `net/http`（標準ライブラリ）
- **テンプレート**: `html/template`（標準ライブラリ）
- **ストレージ**: インメモリ（`sync.RWMutex` でスレッドセーフ）
- **CI**: GitHub Actions（build / test / vet / coverage）
