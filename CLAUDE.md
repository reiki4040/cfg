# cfg - Configuration Management Tool

Go言語製のアプリケーション設定管理ツール＆ライブラリ

## 概要

アプリケーションの設定値を、ローカルやdev/stg/prodなど各環境で透過的に使い分けるための統合的なツールです。機密情報はAWS Parameter Storeで安全に管理し、非機密情報はYAMLファイルで柔軟に管理できます。

## 背景課題

- 環境（local/dev/stg/prod）ごとの設定値管理の複雑さ
- 機密情報の安全な取り扱い
- 環境変数だけでは設定の可視性・管理性が低い
- ローカル開発時の柔軟性確保
- 設定値の手動管理の限界

## 主要機能

### 1. 設定値の取得・解決機能
- YAMLベースの設定定義
- 特殊記法による動的値解決（`${ps:path}`, `${env:KEY}`）
- Parameter Store、環境変数からの値取得
- Stage（環境）別の設定切り替え
- 文字列補間機能（`"${env:ENV}-myapp"`）

### 2. CLI管理ツール（cfgctl）
- Parameter StoreのCRUD操作
- 設定値の一覧表示・検索・比較
- Stage間での設定値比較
- 設定ファイルからの一括反映（plan/apply）
- 構造体からのYAMLテンプレート生成

### 3. Goライブラリ（cfg）
- 透過的な設定値アクセス
- 型安全な設定値取得（構造体タグベース）
- バッチでのAWS API呼び出し最適化
- Stage解決の自動化

## 技術仕様

### 言語・形式
- **言語**: Go
- **設定ファイル**: YAML
- **認証**: IAMロール前提
- **キャッシュ**: 初期化時の一括読み込み

### 特殊記法
```yaml
database:
  host: ${ps:/app/{stage}/db/host}
  password: ${ps:/app/{stage}/db/password}
  port: ${env:DB_PORT}
  url: "https://${ps:/app/{stage}/domain}/api"

app:
  name: "${env:APP_NAME}-{stage}"
  debug: ${env:DEBUG}
```

### Stage機能
- Stage指定方法：環境変数`CFG_STAGE`、コマンドラインオプション`--stage`
- パスの埋め込み：`{stage}`プレースホルダーで動的解決
- 優先度：CLIオプション > 環境変数 > デフォルト値（"dev"）

## プロジェクト構成

```
cfg/
├── cmd/
│   └── cfgctl/
│       └── main.go              # CLI エントリーポイント
├── pkg/
│   ├── cfg/                     # ライブラリ本体
│   │   ├── config.go           # メイン設定ロード機能
│   │   ├── resolver.go         # 値解決エンジン
│   │   ├── parser.go           # YAML パース + 補間
│   │   ├── generator.go        # テンプレート生成
│   │   └── stage.go            # Stage解決エンジン
│   ├── aws/
│   │   └── paramstore.go       # Parameter Store クライアント
│   └── cli/
│       ├── root.go             # CLI ルートコマンド
│       ├── get.go              # cfgctl get
│       ├── set.go              # cfgctl set
│       ├── list.go             # cfgctl list
│       ├── delete.go           # cfgctl delete
│       ├── generate.go         # cfgctl generate
│       ├── plan.go             # cfgctl plan
│       ├── apply.go            # cfgctl apply
│       └── diff.go             # cfgctl diff
├── examples/
│   ├── config.yaml
│   └── main.go
├── go.mod
└── README.md
```

## ライブラリAPI設計

### 基本的な使用方法

```go
package main

import (
    "github.com/yourusername/cfg/pkg/cfg"
)

type Config struct {
    Database struct {
        Host     string `cfg:"database.host"`
        Password string `cfg:"database.password"`
        Port     int    `cfg:"database.port"`
    }
    App struct {
        Name  string `cfg:"app.name"`
        Debug bool   `cfg:"app.debug"`
    }
}

func main() {
    var config Config
    
    loader := cfg.New(cfg.LoadOptions{
        Stage: "prod",
    })
    err := loader.LoadFromFile("config.yaml", &config)
    if err != nil {
        panic(err)
    }
    
    fmt.Println(config.App.Name)
}
```

### 主要な型定義

```go
type Loader struct {
    awsClient *aws.ParameterStoreClient
    stage     string
    cache     map[string]string
}

type LoadOptions struct {
    Stage     string
    AWSRegion string
    Profile   string
}

func New(opts ...LoadOptions) *Loader
func (l *Loader) LoadFromFile(path string, target interface{}) error
func (l *Loader) LoadFromBytes(data []byte, target interface{}) error
func (l *Loader) SetStage(stage string)
```

### 値解決エンジン

```go
type Resolver struct {
    psClient *aws.ParameterStoreClient
    envVars  map[string]string
    cache    map[string]string
    stage    string
}

type Reference struct {
    Type string // "ps" or "env"
    Key  string
    Raw  string // 元の文字列
}

func (r *Resolver) ExtractReferences(yamlContent string) []Reference
func (r *Resolver) ResolveReferences(refs []Reference) (map[string]string, error)
func (r *Resolver) InterpolateString(input string, values map[string]string) string
```

### Stage解決エンジン

```go
type StageResolver struct {
    stage string
}

func (s *StageResolver) ResolvePath(path string) string
func (s *StageResolver) ResolveString(input string) string
```

## CLI コマンド仕様

### Parameter Store操作

```bash
# 基本操作
cfgctl get /app/{stage}/db/password --stage=prod
cfgctl set /app/{stage}/db/password "new-password" --stage=prod --type=SecureString
cfgctl list /app/{stage}/ --stage=prod
cfgctl delete /app/{stage}/db/password --stage=prod

# 一括操作
cfgctl copy --from-stage=stg --to-stage=prod --path=/app/{stage}/db/
cfgctl import parameters.csv --stage=prod

# 対話式設定
cfgctl set --interactive --stage=prod
```

### 設定管理

```bash
# テンプレート生成
cfgctl generate --struct Config --output config.yaml --use-stage-placeholders

# 設定ファイルの計画・適用
cfgctl plan config.yaml --stage=prod
cfgctl apply config.yaml --stage=prod

# 検証
cfgctl validate config.yaml --stage=prod
cfgctl status config.yaml --stage=prod
```

### 比較機能

```bash
# 設定ファイル vs Parameter Store
cfgctl diff config config.yaml --stage=prod

# Parameter Store間の比較
cfgctl diff ps --stage=dev --compare-stage=stg --path=/app/

# ファイル間の比較
cfgctl diff file config.dev.yaml config.prod.yaml

# 複数ステージでの一括比較
cfgctl diff ps --stages=dev,stg,prod --path=/app/
```

## 詳細機能

### plan/applyワークフロー

**plan（実行計画の表示）:**
```bash
cfgctl plan config.yaml --stage=prod

# 出力例:
Configuration Plan for stage 'prod':

Actions to be performed:
  + CREATE /app/prod/app/timeout = "30" (String)
  + CREATE /app/prod/app/debug = "false" (String)  
  + CREATE /app/prod/app/name = "myapp-prod" (String)

Manual setup required (secrets):
  ! /app/prod/db/password (SecureString)
  ! /app/prod/api/key (SecureString)
```

**apply（実行）:**
```bash
cfgctl apply config.yaml --stage=prod

# 非機密情報のみ自動設定
# 機密情報は手動設定が必要（安全性のため）
```

### Parameter Store間の比較

```bash
cfgctl diff ps --stage=dev --compare-stage=stg --path=/app/

# 出力例:
Parameter Store Comparison: dev vs stg (path: /app/)

Key Differences:
  + /app/stg/api/new-feature-flag     (only in stg)
  - /app/dev/debug/local-only         (only in dev)

Value Differences:
  /app/dev/db/host: "db-dev.internal"
  /app/stg/db/host: "db-stg.internal"
```

### セキュリティ配慮

- **機密情報の分離**: 設定ファイルに機密情報を含めない設計
- **SecureStringの隠蔽**: 比較時に機密値を表示しない
- **手動設定**: 機密情報は必ず対話式で手動設定
- **確認プロンプト**: 危険な操作には確認を要求

### パフォーマンス最適化

- **バッチ読み込み**: Parameter Store APIを`GetParameters`でバッチ化
- **並列処理**: 複数ステージの比較時は並列実行
- **キャッシュ**: 一度読み込んだ値はメモリキャッシュ

## エラーハンドリング

```go
type ConfigError struct {
    Type    string // "missing_parameter", "aws_error", "parse_error"
    Path    string
    Message string
    Cause   error
}
```

### エラー処理方針
- Parameter Store値が存在しない：エラー
- Parameter Store記法が設定に存在しない：エラーなし（AWSアクセス不要）
- AWS認証エラー：エラー

## 類似OSSとの差別化

既存の設定管理ツール（Viper、Koanf等）と比較した独自価値：

1. **Parameter Store + 環境変数の統一的な文字列補間**
2. **構造体からのYAMLテンプレート生成**
3. **Parameter Store管理のCLI統合**
4. **条件付きAWSアクセス（記法存在時のみ）**
5. **Stage概念の統合**
6. **機密情報と非機密情報の明確な分離**

## 配布・インストール

```bash
# Go install
go install github.com/yourusername/cfg/cmd/cfgctl@latest

# ライブラリ利用
go get github.com/yourusername/cfg/pkg/cfg
```

## 使用例

### 設定ファイル例（config.yaml）

```yaml
database:
  host: ${ps:/app/{stage}/db/host}
  password: ${ps:/app/{stage}/db/password}
  port: ${env:DB_PORT}
  name: "myapp"
  url: "postgres://${ps:/app/{stage}/db/host}:${env:DB_PORT}/${database.name}"

app:
  name: "${env:APP_NAME}-{stage}"
  debug: ${env:DEBUG}
  api_url: "https://api-{stage}.example.com"
  timeout: 30

redis:
  url: ${ps:/shared/{stage}/redis/url}
  password: ${ps:/shared/{stage}/redis/password}
```

### 構造体定義例

```go
type Config struct {
    Database struct {
        Host     string `cfg:"database.host"`
        Password string `cfg:"database.password"`
        Port     int    `cfg:"database.port"`
        Name     string `cfg:"database.name"`
        URL      string `cfg:"database.url"`
    }
    App struct {
        Name    string `cfg:"app.name"`
        Debug   bool   `cfg:"app.debug"`
        APIURL  string `cfg:"app.api_url"`
        Timeout int    `cfg:"app.timeout"`
    }
    Redis struct {
        URL      string `cfg:"redis.url"`
        Password string `cfg:"redis.password"`
    }
}
```

### 典型的なワークフロー

```bash
# 1. 構造体からテンプレート生成
cfgctl generate --struct Config --output config.yaml --use-stage-placeholders

# 2. 実行計画確認
cfgctl plan config.yaml --stage=prod

# 3. 非機密パラメータの設定
cfgctl apply config.yaml --stage=prod

# 4. 機密パラメータの手動設定
cfgctl set /app/prod/db/password --interactive --type=SecureString
cfgctl set /app/prod/redis/password --interactive --type=SecureString

# 5. 設定状況確認
cfgctl status config.yaml --stage=prod

# 6. 他環境との比較
cfgctl diff ps --stage=prod --compare-stage=stg --path=/app/
```

---

このドキュメントは、cfgツールの完全な設計仕様を記載しています。実装時はこの仕様に基づいて段階的に開発を進めることができます。