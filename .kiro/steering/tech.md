# Technology Stack Steering

## Architecture Overview

**cfg** は 2 層アーキテクチャで構成されています：

### Layer 1: ライブラリ層（pkg/cfg）
- YAML 設定ファイルの読み込みと解析
- 参照解決エンジン（Parameter Store、環境変数、Stage プレースホルダー）
- Go 構造体への型安全なマッピング

### Layer 2: CLI 層（cmd/cfgctl）
- Parameter Store 値の CRUD 操作
- マルチステージ比較機能
- 設定ファイル管理（~/.cfgctl.config）
- 秘密値の安全な入力・表示管理

## Core Technology Stack

### Programming Language
- **Go 1.21**: モダン Go の機能を活用（ジェネリクス、improved error handling）

### AWS Integration
- **AWS SDK for Go v2** (`github.com/aws/aws-sdk-go-v2`): 最新の AWS SDK
  - **Systems Manager Parameter Store** (`service/ssm`): パラメータの読み書き
  - **IAM Integration**: 自動認証（IAM ロール、プロファイル、環境変数）
  - **Region Support**: 複数 AWS リージョンでの動作

### Configuration Management
- **YAML Parsing** (`gopkg.in/yaml.v3`): YAML 設定ファイルのパース
- **Reference Resolution**: 正規表現ベースの参照抽出
  - Parameter Store: `${ps:/path}`
  - Environment Variables: `${env:VAR}`
  - Stage Placeholder: `{stage}`
- **Two-Pass Parsing**: 参照抽出 → 解決 → YAML 再パース

### CLI Framework
- **Cobra** (`github.com/spf13/cobra`): サブコマンド型 CLI フレームワーク
- **Terminal I/O** (`golang.org/x/term`): 秘密値の非表示入力

## Development Environment

### Prerequisites
- Go 1.21 以上
- AWS CLI または AWS 認証情報（IAM ロール、~/.aws/credentials）
- 対象環境での Parameter Store へのアクセス権限

### Build & Installation
```bash
# ライブラリのビルド
go build ./...

# CLI ツールのビルド
go build -o cfgctl ./cmd/cfgctl

# インストール（GOPATH/bin へ）
go install ./cmd/cfgctl
```

### Development Commands
```bash
# テスト実行（ユニットテスト）
go test ./...

# 特定パッケージのテスト
go test ./pkg/cfg -v

# カバレッジ測定
go test -cover ./...

# 依存関係の更新チェック
go list -u -m all

# コードフォーマット（各タスク完了後に必ず実行）
go fmt ./...
```

### Code Quality Standards

#### Formatting & Linting
- **Format Enforcement**: 全コード変更後は `go fmt ./...` を **必ず実行** する
  - タスク 1, 2, 3, 4, 6, 8 の実装完了後
  - テストファイル追加時
  - 新規ファイル作成時
- **目的**: Go のコードスタイルガイドへの準拠、一貫性の確保

#### GoDoc Comments
- 全 **公開関数** (`func FunctionName`) に GoDoc コメント付与
- 全 **公開型** (`type TypeName`) に GoDoc コメント付与
- 非公開関数・型のコメントは複雑なロジックに限定

#### Code Comments
- **複雑なアルゴリズム**: 処理の流れを説明するコメント付与
  - 再帰的ロジック（`updateObjectWithKeys`）
  - 状態遷移（JSONPath 検出フロー）
  - エラーハンドリングロジック
- **意図の明確化**: "なぜ" の説明を優先（"何を" ではなく）

## Environment Variables

### Application Configuration
- `CFG_STAGE`: 設定読み込み時のデフォルト Stage（dev/stg/prod など）
  - デフォルト: `dev`
  - LoadOptions での指定が優先

### AWS Authentication
- `AWS_PROFILE`: AWS 認証プロファイル名
- `AWS_REGION`: AWS リージョン
- `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY`: IAM ユーザー認証情報

### Debug/Development
- `DEBUG`: デバッグモード（出力される場合はログに詳細情報）

## Port Configuration

このプロジェクトは HTTP サーバーを提供しないため、特定のポート使用はありません。

AWS Parameter Store への通信は HTTPS（443 ポート）で実行されます。

## Common Commands

### ライブラリ利用（コード内）
```go
// ローダーの初期化
loader := cfg.New(cfg.LoadOptions{
    Stage: "prod",
    AWSRegion: "ap-northeast-1",
})

// ファイルからの読み込み
err := loader.LoadFromFile("config.yaml", &config)

// バイト配列からの読み込み
err := loader.LoadFromBytes(yamlBytes, &config)
```

### CLI コマンド（cfgctl）
```bash
# 初期設定
cfgctl config init

# Parameter Store 値の設定
cfgctl set {stage}/db/password --SS --stage=prod

# 値の取得
cfgctl get {stage}/db/password --stage=prod

# 一覧表示
cfgctl list {stage}/ --stage=prod --values

# マルチステージ比較
cfgctl diff --stage=dev --compare-stage=prod

# 削除
cfgctl delete {stage}/old_param --stage=prod
```

## Dependency Management

### Direct Dependencies
```
github.com/aws/aws-sdk-go-v2
github.com/aws/aws-sdk-go-v2/config
github.com/aws/aws-sdk-go-v2/service/ssm
github.com/spf13/cobra
golang.org/x/term
gopkg.in/yaml.v3
```

### Update Strategy
- AWS SDK v2: 定期的なマイナーバージョン更新推奨
- Cobra: 互換性維持が高いため定期更新可
- YAML v3: 安定版のためアップデートは慎重に

Go modules を使用しているため、`go get -u` での更新と `go mod tidy` での整理を実施。

## Performance Considerations

### Optimization Points
1. **API 呼び出し最適化**: Parameter Store への一括取得（バッチ処理）
2. **キャッシング**: メモリキャッシュによる同一参照の効率化
3. **YAML パース**: 2 段階パース（初期 parse → 参照解決 → 再 parse）

### Scalability
- YAML ファイルサイズ: 10MB 以下（YAML bomb 攻撃対策）
- Parameter Store バッチ: 最大 10 パラメータ単位での API 呼び出し
- 環境変数: プロセス起動時に全展開（メモリ効率重視）

## Security Posture

- **SecureString Parameters**: KMS で暗号化された機密値
- **IAM Role Integration**: EC2/ECS での自動認証（認証情報をコード内に含めない）
- **Path Validation**: パストラバーサル（`..`）の検出・防止
- **Secure Input**: ターミナルでのパスワード入力は非表示（`term.ReadPassword`）
- **Log Masking**: ログ出力での秘密値マスキング
