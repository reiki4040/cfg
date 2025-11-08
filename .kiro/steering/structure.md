# Project Structure Steering

## Root Directory Organization

```
cfg/
├── go.mod, go.sum              # Go module 定義・ロック
├── README.md                   # プロジェクト全体のドキュメント
├── CLAUDE.md                   # Claude Code の設定と開発ガイドライン
├──
├── config.go                   # メインローダー実装（API の入口）
├── parser.go                   # YAML パース・補間ロジック
├── resolver.go                 # 参照解決エンジン（ps/env/stage-prefix）
├── stage.go                    # Stage 解決ロジック
├── config_error.go             # エラー型定義
├── resolver_test.go            # テスト（resolver 関連）
├──
├── aws/                        # AWS インテグレーション層
│   └── paramstore.go           # AWS Parameter Store クライアント
│
├── cli/                        # CLI 実装（cfgctl）
│   ├── root.go                 # CLI 基本構造・ルート command
│   ├── config.go               # config サブコマンド（init/show/set）
│   ├── config_cmd.go           # config command の内部実装
│   ├── get.go                  # get サブコマンド
│   ├── set.go                  # set サブコマンド
│   ├── delete.go               # delete サブコマンド
│   ├── list.go                 # list サブコマンド
│   ├── diff.go                 # diff（比較）サブコマンド
│   ├── plan.go                 # plan サブコマンド
│   ├── apply.go                # apply サブコマンド
│   ├── export.go               # export サブコマンド
│   ├── generate.go             # generate サブコマンド
│   └── version.go              # version サブコマンド
│
├── cmd/cfgctl/                 # CLI バイナリーエントリーポイント
│   ├── main.go                 # メインエントリー（os.Exit）
│   └── version.go              # バージョン情報
│
├── examples/                   # 使用例・ドキュメント
│   ├── main.go                 # 基本的な使用例
│   ├── main-relative.go        # path prefix 使用例
│   ├── config.yaml             # 設定ファイル例（絶対パス）
│   ├── config-relative.yaml    # 設定ファイル例（相対パス）
│   └── README.md               # 使用例のドキュメント
│
├── .kiro/                      # Kiro spec-driven development
│   ├── steering/               # プロジェクト全体のコンテキスト・ガイドライン
│   │   ├── product.md          # プロダクト概要
│   │   ├── tech.md             # 技術スタック
│   │   └── structure.md        # このファイル（プロジェクト構造）
│   │
│   └── specs/                  # 個別機能の specification
│       └── [feature-name]/     # 機能ごとのディレクトリ
│           ├── spec.json       # メタデータ・承認状況
│           ├── requirements.md # 要件定義（EARS 形式）
│           ├── design.md       # 技術設計
│           └── tasks.md        # 実装タスク（TDD ベース）
│
└── .claude/                    # Claude Code の設定
    └── commands/               # カスタムコマンド定義
```

## Subdirectory Structures

### Root Package (`cfg/`)

**責務**: ライブラリの公開 API と設定読み込みエンジンの実装

**主要ファイル**:
- `config.go`: `Loader` 型と `LoadOptions` 型の定義、public API
- `parser.go`: YAML パース・補間の 2 段階処理
- `resolver.go`: 参照抽出・解決エンジン
- `stage.go`: Stage 解決ロジック（優先順位管理）

**責務分割**:
- `Loader`: 全体オーケストレーション
- `Parser`: YAML 処理
- `Resolver`: 参照マッピング
- `StageResolver`: Stage 名前空間処理

### AWS Package (`aws/`)

**責務**: AWS Services との通信層

**主要ファイル**:
- `paramstore.go`:
  - `ParameterStoreClient` 型（AWS SDK v2 ラッパー）
  - `GetParameter()`, `GetParameters()` など操作 API
  - バッチ処理・キャッシング

**責務分割**:
- AWS SDK ラッピング（認証・リージョン管理）
- API エラーハンドリング
- バッチ最適化（10 パラメータ単位）

### CLI Package (`cli/`)

**責務**: CLI サブコマンド実装

**主要ファイル**:
- `root.go`: 全体フレームワーク（Cobra ベース）
- `config*.go`: 設定管理（init/show/set/unset）
- `get.go`, `set.go`, `delete.go`, `list.go`: Parameter Store CRUD
- `diff.go`: マルチステージ比較
- その他: plan, apply, export, generate, version

**責務分割**:
- コマンド定義・入力検証（各ファイル）
- 共通処理（root.go）
- グローバル状態（config ファイルキャッシュ）

### Binary Entry (`cmd/cfgctl/`)

**責務**: CLI バイナリーのビルドエントリーポイント

**主要ファイル**:
- `main.go`: `cli.Execute()` 呼び出し + exit code 処理
- `version.go`: ビルド時のバージョン情報埋め込み

### Examples (`examples/`)

**責務**: ライブラリ利用方法のドキュメント化

**主要ファイル**:
- `main.go`: 基本的な使用例（絶対パス Parameter Store 参照）
- `main-relative.go`: path prefix 使用時の例
- `config.yaml`, `config-relative.yaml`: 設定ファイルサンプル

## Code Organization Patterns

### Reference Resolution Pattern

参照解決は以下のフローで統一：

```
YAML Content
  → ExtractReferences() [regex matching]
    → ResolveReferences() [fetch values]
      → InterpolateString() [replace placeholders]
        → Re-parse YAML
```

各段階は独立し、参照タイプ（ps/env/stage-prefix）は同列に扱われます。

### Error Handling Pattern

カスタムエラー型 `ConfigError` により、エラー情報を構造化：

```go
type ConfigError struct {
    Type    string // "parse_error", "missing_parameter", "aws_error"
    Path    string // ファイルパスまたはパラメータ名
    Message string
    Cause   error  // 元の error
}
```

エラーは呼び出し元で処理可能な情報を含める。

### CLI Command Pattern

Cobra の標準パターンに従う：

```go
// コマンド定義
var setCmd = &cobra.Command{
    Use:   "set",
    Short: "Set parameter value",
    RunE: func(cmd *cobra.Command, args []string) error {
        // 入力検証
        // AWS 操作
        // エラーハンドリング
    },
}
```

全コマンドは `root.go` に登録される。

## File Naming Conventions

### Go ファイル
- **機能別**: `{機能}.go`（例: `parser.go`, `resolver.go`）
- **テスト**: `{対象}_test.go`（例: `resolver_test.go`）
- **型定義**: ファイルのメインの型を filename に反映

### YAML ファイル
- **設定ファイル**: `config*.yaml`（例: `config.yaml`, `config-relative.yaml`）
- **例**: `config-{説明}.yaml`

### ドキュメント
- **英語**: `README.md`, `LICENSE`
- **日本語**: `.md` ファイルの内容（README でも日本語対応）

## Import Organization

### Standard Library First
```go
import (
    "context"
    "fmt"
    "os"
)
```

### Third-Party Dependencies
```go
import (
    "github.com/aws/aws-sdk-go-v2/..."
    "github.com/spf13/cobra"
    "gopkg.in/yaml.v3"
)
```

### Local Imports
```go
import (
    "github.com/reiki4040/cfg/aws"
)
```

## Key Architectural Principles

### 1. 責務の分離（Separation of Concerns）
- **Loader**: オーケストレーション
- **Parser**: YAML 処理
- **Resolver**: 参照解決
- **StageResolver**: Stage 管理
- **ParameterStoreClient**: AWS 通信

各コンポーネントは単一責務を持ち、テスト可能な設計。

### 2. 参照解決の統一
- ps（Parameter Store）
- env（環境変数）
- stage-prefix（ステージプレフィックス）
- {stage}（ステージプレースホルダー）

すべて同じ Resolver エンジンで処理され、参照タイプは対等。

### 3. Two-Pass Processing
1. **First Pass**: YAML パース + 参照抽出
2. **Resolution**: 参照を値に解決
3. **Second Pass**: 補間済み YAML を再パース

この設計により、参照と値が混在した YAML を確実に処理。

### 4. セキュリティバイデザイン
- **パス検証**: `..` によるトラバーサル防止
- **秘密値マスキング**: ログ出力での秘密値非表示
- **SecureString**: KMS 暗号化パラメータ対応
- **安全入力**: ターミナルでのパスワード非表示入力

### 5. AWS ネイティブ設計
- IAM ロール統合（EC2/ECS での自動認証）
- リージョン対応
- 複数プロファイル対応
- KMS キー管理

## Testing Organization

### Unit Tests
- ファイル: `{対象}_test.go`
- フォーカス: 関数・パッケージレベルの単体テスト
- 対象: resolver, parser, stage resolver

### Integration Tests
- Parameter Store が使用可能な環境での設定読み込みテスト
- テスト用秘密値の事前準備が必要

### CLI Tests
- 本来は integration test（実際の AWS API 呼び出し）
- 開発環境での手動テストが主体

## Development Workflow

### Feature Development
1. `.kiro/steering/` で project-wide context を確認
2. `.kiro/specs/[feature]/` で requirement → design → tasks を実施
3. Feature branch で実装（TDD ベース）
4. PR review → merge

### Testing Requirements
- Unit tests: ロジック部分は必須
- Integration tests: Parameter Store 連携部は可能なら実施
- Manual testing: CLI コマンドの動作確認

### Specification-Driven Development
- 新機能は `.kiro/specs/` で spec を定義してから実装
- Steering はプロジェクト方針の変更時に更新
- 実装状況は `/kiro:spec-status [feature]` で確認
