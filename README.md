# cfg - Configuration Management Tool

Go言語製のアプリケーション設定管理ツール＆ライブラリ。AWS Parameter Storeと環境変数を統合的に管理し、環境（local/dev/stg/prod）ごとの設定値を透過的に使い分けできます。

## ライブラリ（pkg/cfg）

### 概要

アプリケーションの設定値をYAMLファイルで定義し、Parameter Storeや環境変数から動的に値を解決するGoライブラリです。

### 主要機能

- **動的値解決**: `${ps:path}`, `${env:KEY}` 記法による値の動的取得
- **Stage管理**: `{stage}` プレースホルダーによる環境別設定
- **文字列補間**: `"${env:ENV}-myapp"` のような文字列内での値埋め込み
- **型安全**: 構造体タグベースの設定値マッピング
- **バッチ最適化**: AWS APIの効率的な呼び出し

### 基本的な使用方法

```go
package main

import (
    "fmt"
    "github.com/reiki4040/cfg"
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

### 設定ファイル例

**config.yaml (絶対パス形式)**
```yaml
database:
  host: ${ps:/app/{stage}/db/host}
  password: ${ps:/app/{stage}/db/password}
  port: ${env:DB_PORT}
  name: "myapp"
  url: "postgres://${ps:/app/{stage}/db/host}:${env:DB_PORT}/myapp"

app:
  name: "${env:APP_NAME}-{stage}"
  debug: ${env:DEBUG}
  api_url: "https://api-{stage}.example.com"
  timeout: 30

redis:
  url: ${ps:/shared/{stage}/redis/url}
  password: ${ps:/shared/{stage}/redis/password}
```

**config-relative.yaml (path prefix使用)**
```yaml
database:
  host: ${ps:/{stage}/db/host}
  password: ${ps:/{stage}/db/password}
  port: ${env:DB_PORT}
  name: "myapp"

app:
  name: "${env:APP_NAME}-{stage}"
  debug: ${env:DEBUG}
  timeout: 30

redis:
  url: ${ps:/shared/{stage}/redis/url}
  password: ${ps:/shared/{stage}/redis/password}
```

### 特殊記法

- **Parameter Store参照**: `${ps:/path/to/parameter}`
  - path prefixなし: そのまま使用
  - path prefixあり: `prefix + path` で結合
- **環境変数参照**: `${env:VARIABLE_NAME}`
- **Stage プレースホルダー**: `{stage}` - 現在のstageに置換
- **文字列補間**: `"prefix-${env:VAR}-{stage}-suffix"`

### Stage設定

Stageは以下の優先順位で決定されます：

1. LoadOptionsで指定
2. 環境変数 `CFG_STAGE`
3. デフォルト値 `"dev"`

```bash
# 環境変数で指定
export CFG_STAGE=prod

# コードで指定
loader := cfg.New(cfg.LoadOptions{Stage: "prod"})
```

### API仕様

```go
// ローダーの作成
func New(opts ...LoadOptions) *Loader

// 設定ファイルからの読み込み
func (l *Loader) LoadFromFile(path string, target interface{}) error

// バイト配列からの読み込み
func (l *Loader) LoadFromBytes(data []byte, target interface{}) error

// Stage設定
func (l *Loader) SetStage(stage string)

// LoadOptions
type LoadOptions struct {
    Stage     string
    AWSRegion string
    Profile   string
}
```

## cfgctl - CLI管理ツール

Parameter Storeの管理と設定値の比較を行うCLIツールです。

### インストール

```bash
go install github.com/reiki4040/cfg/cmd/cfgctl@latest
```

### 基本設定

```bash
# 初期設定（対話式）
cfgctl config init

# 個別設定
cfgctl config set default_region ap-northeast-1
cfgctl config set default_stage prod
cfgctl config set path_prefix /myapp

# 設定確認
cfgctl config show
```

### Parameter Store操作

#### 基本操作

```bash
# 値の設定（対話式がデフォルト）
cfgctl set {stage}/db/password --SS --stage=prod

# 値の設定（非対話式）
cfgctl set {stage}/app/timeout "30" -S --no-interactive --stage=prod

# 値の取得
cfgctl get {stage}/db/password --stage=prod

# 一覧表示
cfgctl list --stage=prod
cfgctl list {stage}/db --stage=prod --values

# 削除
cfgctl delete {stage}/db/old_param --stage=prod
```

#### パラメータタイプ指定

- `-S` または `--string`: String型
- `--SS`: SecureString型
- `--SL`: StringList型
- `--type=string`: 従来形式（lowercase対応）

#### Secret値の表示

```bash
# Secret値も表示（危険）
cfgctl list --stage=prod --values --show-secrets
```

### Parameter Store比較

```bash
# 2ステージ間の比較
cfgctl diff --stage=dev --compare-stage=stg

# マルチステージ比較
cfgctl diff --stages=dev,stg,prod

# 特定パスの比較
cfgctl diff --stage=dev --compare-stage=stg --path=/app/

# キーのみ表示
cfgctl diff --stage=dev --compare-stage=stg --keys-only

# Secret値も表示
cfgctl diff --stage=dev --compare-stage=stg --show-secrets

# 異なるプロファイルとの比較
cfgctl diff --stage=dev --compare-stage=stg --compare-profile=prod-account
```

### KMS鍵の管理

```bash
# KMS鍵の設定
cfgctl config set kms_key alias/myapp-key --region=ap-northeast-1 --stage=prod

# KMS鍵を使用してSecureString設定
cfgctl set {stage}/api/secret --SS --kms-key=alias/custom-key
```

### パス管理

path prefixを設定することで、パスの結合管理ができます：

```bash
# path prefix設定
cfgctl config set path_prefix /myapp

# /始まりのパスでの操作（prefix + pathで結合: /myapp/{stage}/db/password）
cfgctl set /{stage}/db/password --SS --stage=prod
cfgctl list /{stage}/db --stage=prod
```

### 設定ファイル

設定は `~/.cfgctl.config` に保存されます：

```yaml
default_region: ap-northeast-1
default_stage: dev
path_prefix: /myapp
kms_keys:
  ap-northeast-1:
    stages:
      prod: alias/myapp-prod-key
      stg: alias/myapp-stg-key
  us-east-1:
    stages:
      prod: alias/myapp-us-prod-key
```

### コマンド一覧

| コマンド | 説明 |
|----------|------|
| `cfgctl config` | 設定管理（init, show, set, unset） |
| `cfgctl get` | Parameter Store値の取得 |
| `cfgctl set` | Parameter Store値の設定 |
| `cfgctl list` | Parameter Store値の一覧表示 |
| `cfgctl delete` | Parameter Store値の削除 |
| `cfgctl diff` | Parameter Store値の比較 |

### セキュリティ

- **SecureString**: 暗号化されたパラメータは `*****` で表示
- **対話入力**: 機密値はターミナルで隠して入力
- **KMS統合**: リージョン・ステージ別のKMS鍵管理
- **プロファイル分離**: 異なるAWSアカウント間での比較

## ライセンス

MIT License
