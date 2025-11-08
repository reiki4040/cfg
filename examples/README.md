# cfg Examples

このディレクトリには、cfgライブラリの使用例が含まれています。

## 例の説明

### 1. main.go + config.yaml (絶対パス例)
- **特徴**: Parameter Storeの絶対パスを使用
- **設定**: 設定ファイル内で`${ps:/cfgtool/{stage}/...}`形式で絶対パスを指定
- **用途**: Parameter Storeのパス構造が明確な場合

### 2. main-relative.go + config-relative.yaml (path prefix例)
- **特徴**: path prefixを使用してパスを自動的に結合
- **設定**: `NewWithPrefix("/cfgtool/{stage}", ...)`でpath prefixを指定
- **パス結合ルール**:
  - `/`始まりのパス: `prefix + path` で結合
  - `${ps:/app/api_url}` → `/cfgtool/{stage}/app/api_url`
  - 相対パス: `prefix + "/" + path` で結合
  - `${ps:app/api_url}` → `/cfgtool/{stage}/app/api_url`
- **用途**: 複数のアプリケーションで共通のParameter Store階層を使用する場合

### 3. config-jsonpath.yaml (JSONPath例)
- **特徴**: Parameter StoreのJSON値から特定キーを抽出
- **設定**: `${ps:/path:json.path.to.key}`形式でJSONPathを指定
- **メリット**:
  - API呼び出し削減: 複数値を1つのJSONパラメータに格納可能
  - ネストされたJSON構造のサポート
  - 既存の従来型参照との完全な後方互換性
  - 複数の同じパラメータへの参照は自動的に重複排除（1回のAPI呼び出しに統合）
- **用途**: JSONで複数の設定値を一括保存する場合

## 実行に必要なParameter Store設定

以下のコマンドを実行して、必要なParameter Storeパラメータを設定してください：

```bash
# AWS CLI でParameter Storeパラメータを設定

# dev環境用
aws ssm put-parameter --name "/cfgtool/dev/app/api_url" --value "https://api-dev.example.com" --type "String"
aws ssm put-parameter --name "/cfgtool/dev/app/name2" --value "myapp-dev" --type "String"
aws ssm put-parameter --name "/cfgtool/dev/app1/himitsu" --value "dev-secret-123" --type "SecureString"

# stg環境用
aws ssm put-parameter --name "/cfgtool/stg/app/api_url" --value "https://api-stg.example.com" --type "String"
aws ssm put-parameter --name "/cfgtool/stg/app/name2" --value "myapp-stg" --type "String"
aws ssm put-parameter --name "/cfgtool/stg/app1/himitsu" --value "stg-secret-456" --type "SecureString"

# prod環境用
aws ssm put-parameter --name "/cfgtool/prod/app/api_url" --value "https://api.example.com" --type "String"
aws ssm put-parameter --name "/cfgtool/prod/app/name2" --value "myapp-prod" --type "String"
aws ssm put-parameter --name "/cfgtool/prod/app1/himitsu" --value "prod-secret-789" --type "SecureString"
```

## 実行例

### 1. 絶対パス例の実行
```bash
# dev環境で実行
APP_NAME=myapp DEBUG=true CFG_STAGE=dev go run ./main.go

# stg環境で実行
APP_NAME=myapp DEBUG=false CFG_STAGE=stg go run ./main.go

# prod環境で実行
APP_NAME=myapp DEBUG=false CFG_STAGE=prod go run ./main.go
```

### 2. 相対パス + path prefix例の実行
```bash
# dev環境で実行
APP_NAME=myapp DEBUG=true CFG_STAGE=dev go run ./main-relative.go

# stg環境で実行
APP_NAME=myapp DEBUG=false CFG_STAGE=stg go run ./main-relative.go

# prod環境で実行
APP_NAME=myapp DEBUG=false CFG_STAGE=prod go run ./main-relative.go
```

### 3. JSONPath例の実行
```bash
# dev環境で実行
# このサンプルは、JSONPath機能を使用して複数の値を単一のJSONパラメータから抽出します
go run ./main-jsonpath.go

# または明示的にstageを指定
APP_STAGE=dev go run ./main-jsonpath.go

# prod環境で実行
APP_STAGE=prod go run ./main-jsonpath.go
```

**注記**: JSONPath例を実行するには、以下のParameter Storeパラメータが必要です:
- `/app/{stage}/database-config`: データベース設定（JSON形式）
- `/app/{stage}/server-config`: サーバー設定（JSON形式）
- `/app/{stage}/secrets`: シークレット設定（JSON形式）

### 4. Makefileを使用した一括実行
```bash
# プロジェクトルートで実行
make example-all
```

## 期待される出力

### 1, 2. 従来型の例（main.go, main-relative.go）の出力
```
App Name: myapp-dev-https://api-dev.example.com
API URL: https://api-dev.example.com
Debug Mode: true
Timeout: 30
Secret: dev-secret-123
API domain: dev-api.example.com
Web domain: dev-web.mycompany.com
```

### 3. JSONPath例（main-jsonpath.go）の出力
```
=== Application Configuration ===
App Name: cfg-example-app
App Version: 1.0.0

=== Database Configuration ===
Primary DB Host: primary-db.example.com
Primary DB Port: 5432
Primary DB User: db_user
Replica DB Host: replica-db.example.com
Replica DB Port: 5432
Pool Min Connections: 5
Pool Max Connections: 20
Pool Idle Timeout: 30

=== Server Configuration ===
Server Host: 0.0.0.0
Server Port: 8080
Server Timeout: 30 seconds
TLS Enabled: true
TLS Cert Path: /etc/certs/server.crt
TLS Key Path: /etc/certs/server.key

=== Logging Configuration ===
Log Level: info
Log Format: json

=== Environment ===
Stage: dev
Environment Type: development
Region: us-east-1

=== Secrets (Summary) ===
API Key: [32 chars]
Database Password: [16 chars]

=== JSONPath Benefits ===
✓ API Call Reduction: Multiple values from 1 parameter = 1 API call
✓ Nested JSON Navigation: Dot notation for deep object traversal
✓ Stage Placeholder: {stage} automatically replaced with current stage
✓ Type Conversion: JSON types converted to Go types automatically
✓ Backward Compatible: Works alongside traditional references
✓ Automatic Caching: Parsed JSON cached for 5 minutes
```

## 機能デモ

この例では以下の機能が確認できます：

- **Parameter Store統合**: `${ps:...}`記法でParameter Storeから値を取得
- **Path Prefix機能**: prefixを使用した柔軟なパス管理
  - `/`始まりのパス: `prefix + path` で自動結合
  - 相対パス: `prefix + "/" + path` で自動結合
- **JSONPath機能** (config-jsonpath.yaml): JSON値から特定キーを抽出
  - 形式: `${ps:/path:json.path.to.key}`
  - API呼び出し削減: 複数の同じパラメータ参照を1回のAPI呼び出しに統合
  - ネストされたJSON構造のサポート
  - 型変換: JSON値を自動的にYAML値に変換
- **環境変数統合**: `${env:...}`記法で環境変数から値を取得
- **stage-prefix機能**: `${stage-prefix:...}`記法でstage別のプレフィックス生成
- **Stage解決**: `{stage}`プレースホルダーでstage別のパス生成
- **文字列補間**: 複数の値を組み合わせた文字列生成

## 注意事項

- AWS認証が正しく設定されている必要があります
- Parameter Storeへの読み取り権限が必要です
- SecureStringパラメータの場合、KMSの復号化権限も必要です