# Language and Communication Guidance

## Communication Language Policy

### Primary Language: 日本語（Japanese）

**All explanations and responses to users must be generated in Japanese.**

このプロジェクトでは、Claude Code との対話は日本語で行います。ユーザーへのすべての説明、提案、ドキュメント生成は日本語で実施してください。

## Guidelines for Japanese Communication

### 1. Technical Explanations
- 技術用語は英語のままでよい（例：GitHub、API、Parameter Store）
- ただし、その説明は日本語で詳しく記述
- 複雑な概念は日本語での説明で理解を深める

### 2. Documentation and Comments
- ドキュメント（README、設計書など）の生成時は日本語
- コード内のコメントは日本語またはニュアンスが重要な場合は英語
- 特に実装の意図やビジネスロジックは日本語で明記

### 3. Progress Reports
- タスク完了報告は日本語で詳細に説明
- 実装内容、テスト結果、変更点をすべて日本語で記述
- ユーザーが容易に理解できるような構成を心掛ける

### 4. Error Messages and Warnings
- エラーメッセージはコード内では英語でよい
- しかしユーザーへの説明は日本語で
- 問題の原因、対処方法を日本語で明確に伝える

### 5. Code Review Comments
- コード内のコメント：日本語推奨（複雑な部分）
- PR description：日本語で変更内容を説明
- Commit message：日本語で変更理由を記述

### 6. Interactive Communication
- ユーザーからの質問には日本語で回答
- 提案や確認事項は日本語で明確に述べる
- 曖昧な部分は日本語での詳しい説明で解消

## Implementation Notes

### When to Use English
- 標準ライブラリやフレームワーク名：そのまま使用
- 一般的な技術用語（JSON、REST など）：英語でよい
- ツール名やサービス名：英語表記

### When to Use Japanese
- すべてのドキュメント生成
- ユーザー向けの説明
- ビジネスロジック説明
- エラーや警告の詳細説明
- プロジェクト固有の概念説明

## Examples

### ❌ 不適切（英語での説明）
```
The diff command outputs parameter differences between stages.
Use --color flag to enable color output.
The three modes are: auto, always, never.
```

### ✅ 適切（日本語での説明）
```
diff コマンドは複数ステージ間でのパラメータ値の差分を比較表示します。
--color フラグを使用して色出力を制御できます。
3つのモード：
- auto：ターミナル判定による自動制御（デフォルト）
- always：色を常に有効化
- never：色を常に無効化
```

## Steering File Language

### Steering Document Contents
- `.kiro/steering/product.md`：日本語
- `.kiro/steering/tech.md`：日本語
- `.kiro/steering/structure.md`：日本語
- `.kiro/steering/language-guidance.md`：日本語（このファイル）

Steering ファイルはプロジェクトチーム全体のガイドラインであり、日本語での詳しい説明が必要です。

## Spec Document Language

### Requirements and Design
- `requirements.md`：日本語（要件は日本語で正確に）
- `design.md`：日本語（設計思想を日本語で説明）
- `tasks.md`：日本語（タスク説明は日本語で明確に）

Spec ドキュメントはチーム間の認識統一が重要であり、日本語での正確な記述が不可欠です。

## Summary

**Golden Rule: 内部での思考は英語でよいが、すべてのユーザー向け出力は日本語で生成する。**

これにより：
1. プロジェクトチーム全体が容易に理解できる
2. ドキュメントが統一された言語で管理される
3. 実装の意図が日本語で明確に記録される
4. 国内チームでの開発効率が向上する