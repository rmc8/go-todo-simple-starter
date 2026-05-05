# Todo アプリ仕様書 (Go + Echo + HTMX)

## 1. 概要

本プロジェクトは、Go 言語の基本から実践的な Web 開発パターンまでを体系的に学習するための Todo アプリケーションです。

サーバーサイドでの HTML レンダリング（`html/template`）と HTMX による部分更新を組み合わせることで、JavaScript を最小限に抑えつつ SPA のような操作感を実現します。

### 学習で身につくスキル

- Go の基本文法と標準ライブラリの活用（`html/template`, `database/sql`, `embed`, `log/slog`, `context`, `errors` 等）
- Web フレームワーク（Echo）を使ったルーティング・ミドルウェア設計
- HTMX による宣言的な非同期 UI 更新
- Repository パターンによるデータアクセス層の抽象化
- テストの書き方（`testing` パッケージ）

## 2. 技術スタック

| カテゴリ | 技術 | 採用理由 |
| :--- | :--- | :--- |
| 言語 | Go | 静的型付け、高速、標準ライブラリが充実 |
| Web フレームワーク | [Echo](https://echo.labstack.com/) | 軽量で学習コストが低い、ミドルウェアが豊富 |
| フロントエンド | [HTMX](https://htmx.org/) | HTML 属性のみで非同期通信を実現、JS 不要 |
| テンプレート | `html/template` | Go 標準パッケージ、XSS 対策済み |
| データベース | SQLite3 | ファイルベースでセットアップ不要、SQL を学べる |
| UUID 生成 | `github.com/google/uuid` | RFC 9562 準拠の UUID v4 生成 |
| SQLite ドライバ | `github.com/mattn/go-sqlite3` | Go の `database/sql` と統合可能 (CGO 必須) |
| デザイン | Vanilla CSS | フレームワークに頼らず CSS の基礎を学ぶ |

> [!WARNING]
> `mattn/go-sqlite3` は C 言語のライブラリを内部で使用するため、**C コンパイラ（gcc）** が必要です。
> Linux: `sudo apt install gcc` / macOS: `xcode-select --install` で事前にインストールしてください。

## 3. 機能要件

### 3.1 CRUD 操作（必須）

| 操作 | 説明 | 受け入れ基準 |
| :--- | :--- | :--- |
| **Create** | 新しい Todo を追加 | タイトルが空の場合はエラーを返す |
| **Read** | Todo 一覧を表示 | 作成日時の降順で表示する |
| **Update** | 完了/未完了の切り替え | チェックボックスのクリックで即時反映 |
| **Delete** | 指定した Todo を削除 | 確認なしで即時削除 |

### 3.2 追加機能（学習向け）

| 機能 | 説明 | 学べること |
| :--- | :--- | :--- |
| 一括削除 | 完了済み Todo をまとめて削除 | HTMX の `hx-delete` + リスト再描画 |
| バリデーション | タイトルの空文字チェック | Echo の Bind & Validate パターン |
| エラー表示 | バリデーションエラーをインラインで表示 | HTMX の `hx-target` による部分更新 |
| ロギング | リクエスト情報の構造化ログ出力 | `log/slog` の活用 |

## 4. データモデル

### 4.1 Todo テーブル

```sql
CREATE TABLE IF NOT EXISTS todos (
    id         TEXT      PRIMARY KEY,
    title      TEXT      NOT NULL CHECK(length(title) > 0),
    completed  INTEGER   NOT NULL DEFAULT 0,
    created_at DATETIME  NOT NULL DEFAULT (datetime('now'))
);
```

### 4.2 Go 構造体

```go
type Todo struct {
    ID        string
    Title     string
    Completed bool
    CreatedAt time.Time
}
```

> [!NOTE]
> 本アプリは `html/template` で HTML を返すため、`json:"..."` タグは不要です。将来 JSON API を追加する場合に付与します。

> [!TIP]
> ID には `github.com/google/uuid` で生成する UUID v4 を使用します。自動増分の INTEGER と異なり、アプリケーション側で ID を採番するため、DB に依存しない設計を学べます。

### 4.3 Repository インターフェース

```go
type TodoRepository interface {
    Create(ctx context.Context, todo *Todo) error
    FindAll(ctx context.Context) ([]Todo, error)
    FindByID(ctx context.Context, id string) (*Todo, error)
    Update(ctx context.Context, todo *Todo) error
    Delete(ctx context.Context, id string) error
    DeleteCompleted(ctx context.Context) error
}
```

> [!NOTE]
> インターフェースを定義することで、SQLite 実装をテスト用のモック実装に差し替えることが可能になります。Go のインターフェースは暗黙的に実装される（構造的部分型）ため、`implements` キーワードは不要です。

## 5. API エンドポイント設計

| メソッド | パス | 説明 | リクエスト | レスポンス (HTMX) |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/` | メインページ表示 | - | ページ全体の HTML |
| `POST` | `/todos` | Todo 新規作成 | `title` (フォーム) | 追加された Todo 行 HTML |
| `PATCH` | `/todos/:id/toggle` | 完了状態の切り替え | - | 更新された Todo 行 HTML |
| `DELETE` | `/todos/:id` | Todo 削除 | - | 空レスポンス (200) |
| `DELETE` | `/todos/completed` | 完了済み一括削除 | - | Todo リスト全体の HTML |

> [!NOTE]
> `:id` は UUID 文字列です。HTMX はレスポンスの HTML をそのまま DOM に挿入するため、各エンドポイントは HTML フラグメントを返します。

> [!IMPORTANT]
> **ルーティング順序に注意**: Echo は登録順にルートを評価します。`/todos/completed` を `/todos/:id` **より先に**登録しないと、`completed` が `:id` パラメータとして解釈されてしまいます。

## 6. 画面仕様・HTMX インタラクション

### 6.1 メインページ (`/`)

```
┌─────────────────────────────────┐
│         📝 Todo App             │
├─────────────────────────────────┤
│ [___タイトルを入力___] [追加]   │
├─────────────────────────────────┤
│ ☐ 牛乳を買う           [🗑️]   │
│ ☑ メールを返す          [🗑️]   │
│ ☐ Go の勉強をする      [🗑️]   │
├─────────────────────────────────┤
│        [完了済みを削除]         │
└─────────────────────────────────┘
```

### 6.2 HTMX 属性の設計

| UI 操作 | HTMX 属性 | 動作 |
| :--- | :--- | :--- |
| Todo 追加フォーム | `hx-post="/todos"` `hx-target="#todo-list"` `hx-swap="beforeend"` | フォーム送信後、リスト末尾に追加 |
| チェックボックス | `hx-patch="/todos/:id/toggle"` `hx-target="closest .todo-item"` `hx-swap="outerHTML"` | その行のみ更新 |
| 削除ボタン | `hx-delete="/todos/:id"` `hx-target="closest .todo-item"` `hx-swap="outerHTML"` | その行を DOM から消去 |
| 一括削除ボタン | `hx-delete="/todos/completed"` `hx-target="#todo-list"` `hx-swap="innerHTML"` | リスト全体を再描画 |

## 7. ディレクトリ構成

```text
.
├── cmd/
│   └── server/
│       └── main.go              # エントリポイント: 初期化と起動
├── internal/
│   ├── handler/
│   │   └── todo_handler.go      # HTTP ハンドラ (Echo ルート定義)
│   ├── model/
│   │   └── todo.go              # Todo 構造体 & Repository インターフェース
│   ├── repository/
│   │   ├── sqlite.go            # SQLite による Repository 実装
│   │   └── sqlite_test.go       # Repository のテスト
│   └── renderer/
│       └── template.go          # Echo 用テンプレートレンダラー
├── templates/
│   ├── layout.html              # 共通レイアウト (head, body)
│   ├── index.html               # メインページ
│   └── partials/
│       ├── todo_item.html       # Todo 1 行分のテンプレート
│       ├── todo_list.html       # Todo リスト全体のテンプレート
│       └── error_message.html   # エラー表示用テンプレート
├── static/
│   └── css/
│       └── style.css            # スタイルシート
├── docs/
│   └── spec/                    # 仕様書
├── go.mod
├── go.sum
└── todo.db                      # SQLite データベースファイル (自動生成)
```

> [!IMPORTANT]
> **`cmd/`**: 実行可能バイナリのエントリポイント。`main` パッケージはここだけに置きます。
> **`internal/`**: 外部パッケージからインポートできないプライベートなコード。Go のコンパイラがこれを強制します。

## 8. エラーハンドリング方針

| レイヤー | 方針 |
| :--- | :--- |
| Repository | `error` を返す。`fmt.Errorf("failed to create todo: %w", err)` でラップする |
| Handler | Repository からのエラーを受け取り、適切な HTTP ステータスとエラー HTML を返す |
| ミドルウェア | パニックからの回復（`recover`）とログ出力 |

> [!NOTE]
> Go では `%w` 動詞を使ってエラーをラップし、`errors.Is()` や `errors.As()` でアンラップして判定します。これは Go のエラーハンドリングの重要なパターンです。

## 9. 設定管理

```go
type Config struct {
    Port   string // サーバーポート (デフォルト: ":8080")
    DBPath string // SQLite ファイルパス (デフォルト: "todo.db")
}
```

環境変数または `os.Args` から読み込むシンプルな設計とします。外部ライブラリ（viper 等）は使わず、`os.Getenv()` で標準ライブラリのみを使用します。
