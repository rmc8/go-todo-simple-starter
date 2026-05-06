# 学習ステップガイド (Go + Echo + HTMX Todo App)

本ガイドは [仕様書](./todo_app_spec.md) に基づき、段階的に Todo アプリを構築するための具体的な手順を示します。

各ステップは「前のステップの成果物の上に積み上げる」形になっています。一つずつ確実に動作を確認しながら進めてください。

---

## Step 0: プロジェクトの初期化

### やること

1. Go モジュールの初期化
2. Echo のインストール
3. `.gitignore` の確認（`todo.db` を除外）

### コマンド

```bash
go mod init github.com/rmc8/go-todo-simple-starter
go get github.com/labstack/echo/v4
```

### 学べる Go の概念

- `go mod init` によるモジュール管理
- `go.mod` / `go.sum` の役割
- パッケージのインポートパスの仕組み

### 完了条件

- `go.mod` が生成されている
- `go get` でエラーが出ない

---

## Step 1: Hello World（Echo の基本）

### やること

`cmd/server/main.go` を作成し、Echo で最小限の HTTP サーバーを起動する。

### 実装の概要

```go
package main

import (
    "net/http"
    "github.com/labstack/echo/v4"
)

func main() {
    e := echo.New()
    e.GET("/", func(c echo.Context) error {
        return c.String(http.StatusOK, "Hello, World!")
    })
    e.Logger.Fatal(e.Start(":8888"))
}
```

### 学べる Go の概念

- `package main` と `func main()` の意味
- 関数リテラル（無名関数 / クロージャ）
- `error` 型と Go のエラーハンドリングの基本
- `net/http` の HTTP ステータスコード定数

### 確認方法

```bash
go run cmd/server/main.go
curl http://localhost:8888/
# => Hello, World!
```

### 完了条件

- ブラウザまたは `curl` で `Hello, World!` が表示される

---

## Step 2: HTML テンプレートの導入

### やること

1. `templates/layout.html` と `templates/index.html` を作成する
2. Echo のカスタム `Renderer` を実装する（`internal/renderer/template.go`）
3. `GET /` でテンプレートを使って HTML を返す

### 学べる Go の概念

- `html/template` パッケージの使い方（`ParseGlob`, `ExecuteTemplate`）
- **インターフェースの実装**: Echo の `echo.Renderer` インターフェースを自分で実装する
  ```go
  type Renderer interface {
      Render(io.Writer, string, interface{}, echo.Context) error
  }
  ```
- `io.Writer` インターフェースの意味（`*bytes.Buffer`, `http.ResponseWriter` など）
- テンプレートの `{{ define }}` / `{{ template }}` / `{{ block }}` 構文

### ファイル構成

```
templates/
├── layout.html    # {{ define "layout" }} ... {{ template "content" . }} ... {{ end }}
└── index.html     # {{ define "content" }} ... {{ end }}
```

### 実装のヒント: テンプレートレンダラー

Echo の `Renderer` インターフェースを実装する具体例です。

```go
package renderer

import (
    "html/template"
    "io"

    "github.com/labstack/echo/v4"
)

// TemplateRenderer は Echo の echo.Renderer インターフェースを実装する構造体
type TemplateRenderer struct {
    templates *template.Template
}

// NewTemplateRenderer はテンプレートをパースしてレンダラーを生成する
func NewTemplateRenderer(pattern string) (*TemplateRenderer, error) {
    t, err := template.ParseGlob(pattern)
    if err != nil {
        return nil, err
    }
    return &TemplateRenderer{templates: t}, nil
}

// Render は echo.Renderer インターフェースの実装
func (r *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
    return r.templates.ExecuteTemplate(w, name, data)
}
```

### 完了条件

- ブラウザで HTML ページが表示される
- レイアウト（head, body）が正しく適用されている

---

## Step 3: 静的ファイルの配信と CSS

### やること

1. `static/css/style.css` を作成する
2. Echo の `e.Static()` で静的ファイルを配信する
3. Todo アプリの見た目を整える

### 学べる Go の概念

- Echo の `Static()` ミドルウェア
- （発展）`embed` パッケージを使ったバイナリ埋め込み
  ```go
  //go:embed static
  var staticFiles embed.FS
  ```

### 完了条件

- CSS が適用され、見た目が整った HTML が表示される

---

## Step 4: データモデルと Repository インターフェース

### やること

1. `internal/model/todo.go` に `Todo` 構造体と `TodoRepository` インターフェースを定義する

### 実装の概要

```go
package model

import (
    "context"
    "time"
)

type Todo struct {
    ID        string
    Title     string
    Completed bool
    CreatedAt time.Time
}

type TodoRepository interface {
    Create(ctx context.Context, todo *Todo) error
    FindAll(ctx context.Context) ([]Todo, error)
    FindByID(ctx context.Context, id string) (*Todo, error)
    Update(ctx context.Context, todo *Todo) error
    Delete(ctx context.Context, id string) error
    DeleteCompleted(ctx context.Context) error
}
```

### 学べる Go の概念

- **構造体（struct）**: フィールドとメソッドの定義
- **インターフェース**: メソッドのシグネチャだけを定義し、実装を分離する
- **`context.Context`**: リクエストスコープのキャンセル・タイムアウト伝搬
- ポインタ (`*Todo`) と値 (`Todo`) の使い分け

### 完了条件

- コンパイルが通る（`go build ./...`）
- まだ DB 接続はしない

---

## Step 5: SQLite の接続と Repository 実装

### やること

1. SQLite ドライバのインストール
2. `internal/repository/sqlite.go` に `TodoRepository` インターフェースの SQLite 実装を作成する
3. テーブルの自動作成（`CREATE TABLE IF NOT EXISTS`）

### コマンド

```bash
go get github.com/mattn/go-sqlite3
go get github.com/google/uuid
```

> [!WARNING]
> `mattn/go-sqlite3` は CGO（C 言語バインディング）を使用します。事前に C コンパイラがインストールされていることを確認してください。
> `gcc --version` で確認できます。インストールされていない場合: `sudo apt install gcc` (Linux)

### 学べる Go の概念

- **`database/sql`** パッケージ（`sql.DB`, `sql.Row`, `sql.Rows`）
- プリペアドステートメントと SQL インジェクション対策
- `defer` の使い方（`rows.Close()` のリソース解放）
- **エラーラッピング**: `fmt.Errorf("failed to ...: %w", err)`
- **UUID 生成**: `github.com/google/uuid` の使い方
  ```go
  import "github.com/google/uuid"
  id := uuid.New().String()
  ```
- **構造体がインターフェースを満たす仕組み**: `implements` キーワードなしの暗黙的実装

### 実装のヒント

```go
type SQLiteRepository struct {
    db *sql.DB
}

func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }
    // テーブル作成
    if _, err := db.Exec(createTableSQL); err != nil {
        return nil, fmt.Errorf("failed to create table: %w", err)
    }
    return &SQLiteRepository{db: db}, nil
}

func (r *SQLiteRepository) Create(ctx context.Context, todo *model.Todo) error {
    todo.ID = uuid.New().String()
    todo.CreatedAt = time.Now()
    _, err := r.db.ExecContext(ctx, "INSERT INTO todos ...", ...)
    return err
}
```

> [!TIP]
> UUID の生成は Repository 内で行います。Handler がら UUID ライブラリを知る必要がなくなり、責務が明確になります。

### 完了条件

- `main.go` からDB接続して、テーブルが作成される
- `todo.db` ファイルが生成される

---

## Step 6: Read（一覧表示）

### やること

1. `internal/handler/todo_handler.go` を作成する
2. `GET /` でDBから全件取得し、テンプレートに渡して一覧を表示する
3. `templates/partials/todo_item.html` を作成する

### 学べる Go の概念

- **スライス (`[]Todo`)** の操作
- テンプレートの `{{ range }}` によるループ処理
- ハンドラとリポジトリの依存注入（コンストラクタインジェクション）
  ```go
  type TodoHandler struct {
      repo model.TodoRepository
  }

  func NewTodoHandler(repo model.TodoRepository) *TodoHandler {
      return &TodoHandler{repo: repo}
  }
  ```

### 完了条件

- ブラウザで Todo 一覧が表示される（まだデータはないので空リスト）
- DB に手動でデータを入れると表示される

---

## Step 7: Create（HTMX による非同期追加）

### やること

1. 入力フォームに HTMX 属性を追加する
2. `POST /todos` ハンドラを実装する
3. サーバーから返された HTML フラグメントがリストに追加されることを確認する

### HTMX の属性

```html
<form hx-post="/todos" hx-target="#todo-list" hx-swap="beforeend">
    <input type="text" name="title" required placeholder="タイトルを入力">
    <button type="submit">追加</button>
</form>
```

### 学べる Go の概念

- Echo の `c.FormValue()` によるフォームデータの取得
- 部分テンプレートの返却（ページ全体ではなく、追加された行のみ）
- **バリデーション**: 空文字チェックとエラーレスポンスの返し方

### 学べる HTMX の概念

- `hx-post`: フォーム送信先
- `hx-target`: レスポンスを挿入する対象要素
- `hx-swap`: 挿入方法（`beforeend` = 末尾に追加）

### 完了条件

- フォームにタイトルを入力して送信すると、ページリロードなしで一覧に追加される
- 空のタイトルでは追加されない

---

## Step 8: Update（完了状態の切り替え）

### やること

1. チェックボックスに HTMX 属性を追加する
2. `PATCH /todos/:id/toggle` ハンドラを実装する
3. クリックで状態が切り替わることを確認する

### HTMX の属性

```html
<input type="checkbox"
       hx-patch="/todos/{{.ID}}/toggle"
       hx-target="closest .todo-item"
       hx-swap="outerHTML"
       {{if .Completed}}checked{{end}}>
```

### 学べる Go の概念

- Echo の `c.Param("id")` によるパスパラメータの取得
- Bool 値の反転ロジック
- `PATCH` メソッドの意味（リソースの部分更新）

### 完了条件

- チェックボックスをクリックすると、ページリロードなしで状態が切り替わる
- 完了した Todo に取り消し線などの視覚的な変化がある

---

## Step 9: Delete（削除）

### やること

1. 削除ボタンに HTMX 属性を追加する
2. `DELETE /todos/:id` ハンドラを実装する
3. `DELETE /todos/completed` ハンドラを実装する（一括削除）

### 学べる Go の概念

- `DELETE` メソッドの意味
- 空レスポンスの返し方
- HTMX の `hx-swap="outerHTML"` と空レスポンスで要素が消える仕組み

### 完了条件

- 削除ボタンをクリックすると、その Todo が即座に消える
- 「完了済みを削除」ボタンで、完了済み Todo がまとめて消える

---

## Step 10: ミドルウェアとロギング

### やること

1. `log/slog` を使ったリクエストロガーミドルウェアを自作する
2. Echo の `Recover` ミドルウェアを適用する

### 実装のヒント

```go
func RequestLogger(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        start := time.Now()
        err := next(c)
        slog.Info("request",
            "method", c.Request().Method,
            "path", c.Request().URL.Path,
            "status", c.Response().Status,
            "latency", time.Since(start).String(),
        )
        return err
    }
}
```

### 学べる Go の概念

- **ミドルウェアパターン**: 関数を受け取って関数を返す高階関数
- `log/slog`: Go 1.21+ の構造化ログパッケージ
- `time.Now()` と `time.Since()` による処理時間計測

### 完了条件

- リクエストごとにメソッド、パス、ステータスコード、処理時間がログに出力される

---

## Step 11: テストの導入

### やること

1. `internal/repository/sqlite_test.go` を作成する
2. Repository のテストを書く

### 実装のヒント

```go
func TestCreate(t *testing.T) {
    // テスト用に一時ファイルの DB を使う
    repo, err := NewSQLiteRepository(":memory:")
    if err != nil {
        t.Fatalf("failed to create repository: %v", err)
    }

    todo := &model.Todo{Title: "テスト Todo"}
    if err := repo.Create(context.Background(), todo); err != nil {
        t.Fatalf("failed to create todo: %v", err)
    }

    if todo.ID == "" {
        t.Error("expected ID to be set")
    }
}
```

### 学べる Go の概念

- `testing` パッケージの基本（`*testing.T`, `t.Errorf`, `t.Fatalf`）
- テーブル駆動テスト（Table-Driven Tests）パターン
- SQLite の `:memory:` モードを使った一時的な DB テスト
- `go test ./...` コマンドの使い方

### 完了条件

- `go test ./internal/repository/...` が全件パスする

---

## Step 12: リファクタリングと仕上げ

### やること

1. 設定管理を `os.Getenv()` で実装する
2. エラーハンドリングを見直し、`%w` ラッピングを徹底する
3. コードの整理（不要な `import` の削除、コメントの追加）
4. `go vet ./...` と `go fmt ./...` で品質チェック

### 学べる Go の概念

- `os.Getenv()` による環境変数の読み込み
- `go vet`: 静的解析ツール
- `go fmt`: コードフォーマッター（Go ではフォーマットが統一されている）
- `errors.Is()` / `errors.As()` によるエラー判定

### 完了条件

- `go vet ./...` で警告がない
- 環境変数でポートや DB パスを変更できる
- すべての CRUD 操作が問題なく動作する

---

## まとめ: 各ステップで学べる Go の主要概念

| ステップ | 主な学習テーマ |
| :--- | :--- |
| 0 | Go モジュール管理 |
| 1 | 基本文法、`error` 型 |
| 2 | インターフェース、`html/template` |
| 3 | 静的ファイル配信、`embed` |
| 4 | 構造体、インターフェース設計、`context` |
| 5 | `database/sql`、`defer`、エラーラッピング |
| 6 | スライス操作、依存注入 |
| 7 | フォーム処理、バリデーション |
| 8 | パスパラメータ、HTTP メソッド |
| 9 | リソース削除パターン |
| 10 | ミドルウェア、`log/slog`、高階関数 |
| 11 | `testing`、テーブル駆動テスト |
| 12 | `go vet`、`go fmt`、環境変数 |
