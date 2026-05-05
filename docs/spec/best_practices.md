# Go 開発ベストプラクティスと重要コード例

本ドキュメントでは、Todo アプリの開発を通じて学ぶべき Go 言語特有のベストプラクティス、重要な実装手順、および実践的なコード例をまとめます。

---

## 1. 開発におけるベストプラクティス (Go Idioms)

### 1.1 エラーハンドリングは明示的に行う
Go では例外（Exception）を使わず、戻り値として `error` を返します。エラーが発生した場合は、**どこで何が起きたか**の文脈（コンテキスト）を付与して呼び出し元に返すのがベストプラクティスです。
- ❌ 悪い例: `return err` (どこでエラーが起きたか分かりにくい)
- ⭕ 良い例: `return fmt.Errorf("failed to insert todo: %w", err)` (`%w` を使って元のエラーをラップする)

### 1.2 依存性の注入 (Dependency Injection) とインターフェース
特定の実装（例: SQLite）に依存するのではなく、**インターフェース**に依存するように設計します。これにより、テスト時にモック（偽物の DB）への差し替えが容易になります。
- ハンドラ（Echo）は `TodoRepository` インターフェースに依存し、`SQLiteRepository` 構造体には直接依存させません。

### 1.3 `context.Context` の適切な引き回し
データベース操作や外部 API 呼び出しなどの時間のかかる処理には、必ず第一引数に `context.Context` を渡します。これにより、クライアントが通信を切断した際に、不要な DB クエリをキャンセルできます。

### 1.4 テーブル駆動テスト (Table-Driven Tests)
Go のテストでは、テストケースを構造体のスライス（配列）として定義し、それをループで回す「テーブル駆動テスト」が標準的です。これにより、新しいテストパターンの追加が非常に簡単になります。

### 1.5 グローバル変数を避ける
DB のコネクションプール（`*sql.DB`）などをグローバル変数に置くのはアンチパターンです。構造体のフィールドとして保持し、メソッド経由でアクセスするようにします。

---

## 2. 重要な実装パターンとコード例

### 2.1 エラーのラップと判定

`fmt.Errorf` と `%w` を使ってエラーをラップし、呼び出し元で `errors.Is` を使って特定のエラーかどうかを判定します。

```go
package repository

import (
    "database/sql"
    "errors"
    "fmt"
)

var ErrNotFound = errors.New("todo not found")

func (r *SQLiteRepository) FindByID(ctx context.Context, id string) (*model.Todo, error) {
    row := r.db.QueryRowContext(ctx,
        "SELECT id, title, completed, created_at FROM todos WHERE id = ?", id)
    var todo model.Todo
    
    if err := row.Scan(&todo.ID, &todo.Title, &todo.Completed, &todo.CreatedAt); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            // 標準のエラーをアプリケーション固有のエラーに変換してラップ
            return nil, fmt.Errorf("query failed: %w", ErrNotFound)
        }
        return nil, fmt.Errorf("database error: %w", err)
    }
    return &todo, nil
}
```

### 2.2 リポジトリパターンとプリペアドステートメント

SQL インジェクションを防ぐため、パラメータの埋め込みには必ずプレースホルダー（`?`）を使用します。文字列連結は絶対に避けてください。

```go
func (r *SQLiteRepository) Create(ctx context.Context, todo *model.Todo) error {
    query := `INSERT INTO todos (id, title, completed, created_at) VALUES (?, ?, ?, ?)`
    
    // ExecContext を使用し、プレースホルダーに値を渡す
    _, err := r.db.ExecContext(ctx, query, todo.ID, todo.Title, todo.Completed, todo.CreatedAt)
    if err != nil {
        return fmt.Errorf("failed to execute insert: %w", err)
    }
    return nil
}
```

### 2.3 テーブル駆動テスト (Table-Driven Tests)

複数の入力パターンと期待される出力パターンをテーブル形式で定義します。

```go
package handler

import (
    "testing"
)

func TestValidateTitle(t *testing.T) {
    // 1. テストケースの定義 (テーブル)
    tests := []struct {
        name    string
        title   string
        wantErr bool
    }{
        {"Valid title", "買い物に行く", false},
        {"Empty title", "", true},
        {"Whitespace only", "   ", true},
    }

    // 2. テストの実行
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateTitle(tt.title)
            
            // エラーを期待しているのにエラーが出なかった場合
            if tt.wantErr && err == nil {
                t.Errorf("expected error, but got nil")
            }
            // エラーを期待していないのにエラーが出た場合
            if !tt.wantErr && err != nil {
                t.Errorf("unexpected error: %v", err)
            }
        })
    }
}
```

### 2.4 Echo ハンドラでの HTMX 応答 (HTML フラグメントの返却)

Echo では、ページ全体（`index.html`）を返すエンドポイントと、HTMX のために部分的な HTML（例: リストの 1 行）を返すエンドポイントを使い分けます。

```go
func (h *TodoHandler) CreateTodo(c echo.Context) error {
    title := c.FormValue("title")
    if title == "" {
        // バリデーションエラー時はエラースニペットを返す (HTMX向け)
        return c.Render(http.StatusBadRequest, "error_message", "タイトルは必須です")
    }

    // UUID 生成と CreatedAt の設定は Repository 側の責務
    todo := &model.Todo{
        Title: title,
    }

    if err := h.repo.Create(c.Request().Context(), todo); err != nil {
        // ... エラーハンドリング
    }

    // 成功時は新しく作成した Todo の 1 行分の HTML (partial) を返す
    return c.Render(http.StatusCreated, "todo_item", todo)
}
```

### 2.5 フロントエンド (HTMX) の実装パターン

JavaScript を書かずに、HTML の属性だけで非同期通信と DOM の更新を行います。

```html
<!-- Todo を追加するフォーム -->
<!-- hx-post: 送信先 -->
<!-- hx-target: サーバーからのレスポンス(HTML)を挿入する対象の要素 -->
<!-- hx-swap="beforeend": 対象要素の「内側の末尾」に追加する -->
<form hx-post="/todos" hx-target="#todo-list" hx-swap="beforeend">
    <input type="text" name="title" required>
    <button type="submit">追加</button>
</form>

<!-- Todo リストのコンテナ -->
<ul id="todo-list">
    <!-- ここに todo_item の HTML が追加されていく -->
    {{ range .Todos }}
        {{ template "todo_item" . }}
    {{ end }}
</ul>
```
