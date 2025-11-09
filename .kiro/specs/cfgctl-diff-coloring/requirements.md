# Requirements Document

## Project Description (Input)
cfgctl diffの差分に色をつけて人間が見やすくしたいです（実装方針により、テキストベースのANSI色が利用される）。

## Introduction
このフィーチャーは、cfgctl diff コマンドの出力に色（カラー表示）を追加し、ターミナル上での視認性と読みやすさを向上させるものです。複数ステージ間のパラメータ差分を比較する際に、削除されたパラメータ（赤）、追加されたパラメータ（緑）、変更されたパラメータ（黄など）を色分けして表示することで、ユーザーが差分を直感的に把握できるようになります。

## Requirements

### Requirement 1: 差分タイプに基づいた色分け表示
**Objective:** ユーザーとして、diff出力に色分けされた差分を見たいので、削除・追加・変更の違いをパッと見で認識できる

#### Acceptance Criteria
1. WHEN cfgctl diff コマンドが実行されるとき、差分として「削除されたパラメータ」を表示する場合、THEN diff コマンド SHALL 削除行（- で始まる行）を赤色で表示する
2. WHEN cfgctl diff コマンドが実行されるとき、差分として「追加されたパラメータ」を表示する場合、THEN diff コマンド SHALL 追加行（+ で始まる行）を緑色で表示する
3. WHEN cfgctl diff コマンドが実行されるとき、差分として「変更されたパラメータ」を表示する場合、THEN diff コマンド SHALL 変更行（~ で始まる行）を黄色で表示する
4. WHEN cfgctl diff コマンドが実行されるとき、差分として「同一のパラメータ」を表示する場合、THEN diff コマンド SHALL 変更がない行をデフォルト色で表示する

### Requirement 2: マルチステージ比較表での色分け表示
**Objective:** ユーザーとして、マルチステージ比較（--stages オプション）で各ステージの値が異なる場合に色分けして見たいので、値の違いをすぐに認識できる

#### Acceptance Criteria
1. WHEN cfgctl diff --stages=dev,stg,prod が実行されるとき、THEN diff コマンド SHALL テーブル形式で各パラメータの値を複数ステージで並べて表示する
2. WHEN マルチステージ比較テーブルに複数ステージで異なる値がある場合、THEN diff コマンド SHALL 異なる値を視覚的に区別できる色で強調表示する
3. WHEN マルチステージ比較テーブルに複数ステージで同一の値がある場合、THEN diff コマンド SHALL デフォルト色で表示する
4. WHEN パラメータが一部のステージにのみ存在する場合、THEN diff コマンド SHALL 存在しないセルを視覚的に区別できる色またはフォーマットで表示する

### Requirement 3: JSON 属性レベル差分での色分け表示
**Objective:** ユーザーとして、JSON 値の属性単位での差分を見やすくしたいので、複雑な JSON 構造でも修正された部分をすぐに見つけられる

#### Acceptance Criteria
1. WHEN JSON 値の属性レベル差分が表示される場合、THEN diff コマンド SHALL 差分がある属性（削除・追加・変更）の行を対応する色で表示する
2. WHEN JSON 差分出力にネストされた属性が含まれる場合、THEN diff コマンド SHALL インデント構造を保ちながら色分けして表示する
3. WHEN JSON 差分出力の変更箇所が複数の属性に及ぶ場合、THEN diff コマンド SHALL すべての変更箇所を一貫した色スキームで表示する

### Requirement 4: 色出力の制御オプション
**Objective:** ユーザーとして、出力先やログファイル用途に応じて色出力を制御したいので、適切な場面で色を有効・無効にできる

#### Acceptance Criteria
1. WHEN cfgctl diff コマンドが実行されるとき、AND 出力がパイプ処理される場合または標準出力がターミナルでない場合、THEN diff コマンド SHALL 色出力を自動的に無効化する
2. WHEN cfgctl diff --color=always フラグが指定される場合、THEN diff コマンド SHALL 色出力を強制的に有効にする
3. WHEN cfgctl diff --color=never フラグが指定される場合、THEN diff コマンド SHALL 色出力を強制的に無効にする
4. WHEN cfgctl diff --color=auto フラグが指定される場合（デフォルト）、THEN diff コマンド SHALL 出力がターミナルかどうかを判定して色出力を自動制御する

### Requirement 5: セキュリティと秘密値の表示
**Objective:** ユーザーとして、SecureString パラメータが色分けされた場合でも秘密値が漏露しないようにしたいので、マスク表示を色分けで効果的に表示できる

#### Acceptance Criteria
1. WHEN --show-secrets フラグなしで SecureString パラメータが表示される場合、THEN diff コマンド SHALL マスク文字列（***masked secret*** など）を対応する差分タイプの色で表示する
2. WHEN --show-secrets フラグ付きで SecureString パラメータの実際の値が表示される場合、THEN diff コマンド SHALL 値を対応する差分タイプの色で表示する
3. WHILE SecureString パラメータが表示される場合、THE diff コマンド SHALL 常にマスク機能を保持し、--show-secrets フラグなしではマスク表示を優先する

### Requirement 6: ターミナルの互換性
**Objective:** ユーザーとして、異なるターミナルやOS環境で色表示が正常に動作してほしいので、デバイス依存性なく一貫した色表示が得られる

#### Acceptance Criteria
1. WHEN cfgctl diff コマンドが異なるターミナルエミュレータ（macOS Terminal、iTerm2、Linux bash、Windows Terminal など）で実行される場合、THEN diff コマンド SHALL ANSI エスケープコードを使用して色を表現する
2. WHEN --color=always で色出力が強制される場合、THEN diff コマンド SHALL ANSI エスケープコードの標準的な色定義（黒・赤・緑・黄・青・マゼンタ・シアン・白）を使用する
3. WHERE 古いターミナル環境など ANSI 色をサポートしない環境では、THEN diff コマンド SHALL 色出力を無効化した方が望ましい
