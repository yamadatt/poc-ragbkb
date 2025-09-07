# Implementation Plan: AWS RAGシステム

**Branch**: `001-aws-api-gateway` | **Date**: 2025-09-04 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/Users/yamadatt/Downloads/poc-ragbkb/poc-ragbkb/specs/001-aws-api-gateway/spec.md`

## 実行フロー (/planコマンドの範囲)
```
1. Input パスから機能仕様を読み込み
   → 見つからない場合: ERROR "機能仕様が{path}にありません"
2. Technical Context を入力 (NEEDS CLARIFICATION をスキャン)
   → コンテキストからプロジェクトタイプを検出 (web=フロントエンド+バックエンド, mobile=アプリ+API)
   → プロジェクトタイプに基づいて構造を決定
3. 以下のConstitution Checkセクションを評価
   → 違反がある場合: Complexity Trackingに記録
   → 正当化不可能な場合: ERROR "まずアプローチを簡素化してください"
   → Progress Tracking更新: 初期憲法チェック
4. Phase 0を実行 → research.md
   → NEEDS CLARIFICATION が残っている場合: ERROR "不明点を解決してください"
5. Phase 1を実行 → contracts, data-model.md, quickstart.md, エージェント固有テンプレートファイル
6. Constitution Checkセクションを再評価
   → 新たな違反がある場合: 設計をリファクタリング、Phase 1に戻る
   → Progress Tracking更新: 設計後憲法チェック
7. Phase 2を計画 → タスク生成アプローチを記述 (tasks.mdは作成しない)
8. 停止 - /tasksコマンドの準備完了
```

**重要**: /planコマンドはステップ7で停止します。Phase 2-4は他のコマンドで実行されます：
- Phase 2: /tasksコマンドがtasks.mdを作成
- Phase 3-4: 実装の実行（手動またはツール経由）

## 概要
AWS Bedrock Knowledge BaseとAPI Gateway、Lambdaを活用したRAG（Retrieval Augmented Generation）システムを構築する。ユーザーが自然言語で質問を投稿すると、Knowledge Baseから関連文書を検索し、Bedrock AIモデルが回答を生成する。フロントエンドはReact/TypeScript、バックエンドはGo言語、インフラはTerraform（Knowledge Base/S3）とSAM（API Gateway/Lambda）で構築する。認証不要のオープンアクセス設計。

## 技術的コンテキスト
**Language/Version**: Go 1.21+ (バックエンド), TypeScript 5.0+ (フロントエンド)  
**Primary Dependencies**: AWS SDK Go v2, Gin Web Framework, React 18, Vite, AWS Bedrock, AWS API Gateway  
**Storage**: AWS S3 (文書保存), AWS Bedrock Knowledge Base (ベクトルDB), DynamoDB (ログ)  
**Testing**: Go標準testing, testify, React Testing Library, Vitest  
**Target Platform**: AWS Lambda (サーバーレス), WebブラウザHTML5+
**Project Type**: web - frontend + backend structure  
**Performance Goals**: 同時3ユーザー対応, 質問応答5秒以内レスポンス  
**Constraints**: 文書サイズ50MB上限, 認証不要設計, コスト最適化  
**Scale/Scope**: 小規模PoC, txt/mdファイル対応, 将来拡張可能設計

## 憲法チェック
*ゲート: Phase 0のリサーチ前に通過必須。Phase 1設計後に再チェック。*

**簡潔性**:
- Projects: 3 (backend API, frontend Web, infrastructure)
- Using framework directly? ✓ (Gin/React directly, no wrappers)
- Single data model? ✓ (Document, Query, Response entities)
- Avoiding patterns? ✓ (Direct AWS SDK calls, no Repository layer)

**アーキテクチャ**:
- EVERY feature as library? ✓ (document-processor, query-handler, knowledge-base-client)
- Libraries listed: 
  - document-processor (文書処理・S3アップロード)
  - query-handler (質問処理・Bedrock連携) 
  - knowledge-base-client (Knowledge Base操作)
- CLI per library: ✓ (各ライブラリに --help/--version/--json対応)
- Library docs: ✓ (llms.txt format planned)

**テスト (交渉不可)**:
- RED-GREEN-Refactor cycle enforced? ✓ (テスト先行開発)
- Git commits show tests before implementation? ✓ (計画済み)
- Order: Contract→Integration→E2E→Unit strictly followed? ✓
- Real dependencies used? ✓ (実際のS3/Bedrock/Knowledge Base)
- Integration tests for: ✓ (AWS API統合、文書処理フロー、RAG機能)
- FORBIDDEN: Implementation before test, skipping RED phase ✓

**可観測性**:
- Structured logging included? ✓ (JSON形式ログ)
- Frontend logs → backend? ✓ (統合ログストリーム計画)
- Error context sufficient? ✓ (リクエストID、タイムスタンプ、エラー詳細)

**バージョニング**:
- Version number assigned? ✓ (0.1.0 - 初期PoC)
- BUILD increments on every change? ✓ (計画済み)
- Breaking changes handled? ✓ (並行テスト、移行計画)

## プロジェクト構造

### ドキュメント（この機能）
```
specs/[###-feature]/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### ソースコード（リポジトリルート）
```
# Option 1: Single project (DEFAULT)
src/
├── models/
├── services/
├── cli/
└── lib/

tests/
├── contract/
├── integration/
└── unit/

# Option 2: Web application (when "frontend" + "backend" detected)
backend/
├── src/
│   ├── models/
│   ├── services/
│   └── api/
└── tests/

frontend/
├── src/
│   ├── components/
│   ├── pages/
│   └── services/
└── tests/

# Option 3: Mobile + API (when "iOS/Android" detected)
api/
└── [same as backend above]

ios/ or android/
└── [platform-specific structure]
```

**構造決定**: オプション2 - Webアプリケーション（技術的コンテキストでフロントエンド+バックエンドを検出）

## Phase 0: 概要・調査
1. **上記の技術的コンテキストから不明点を抽出**:
   - 各NEEDS CLARIFICATION → 調査タスク
   - 各依存関係 → ベストプラクティスタスク
   - 各統合 → パターンタスク

2. **調査エージェントを生成・派遣**:
   ```
   技術的コンテキストの各不明点について:
     タスク: "{機能コンテキスト}で{不明点}を調査"
   各技術選択について:
     タスク: "{ドメイン}で{技術}のベストプラクティスを見つける"
   ```

3. **調査結果を統合** `research.md`に以下の形式で記録:
   - 決定: [選択されたもの]
   - 根拠: [選択理由]
   - 検討した代替案: [評価したその他の選択肢]

**出力**: すべてのNEEDS CLARIFICATIONを解決したresearch.md

## Phase 1: 設計・契約
*前提条件: research.md完了*

1. **機能仕様からエンティティを抽出** → `data-model.md`:
   - エンティティ名、フィールド、関係性
   - 要件からの検証ルール
   - 該当する場合の状態遷移

2. **機能要件からAPIコントラクトを生成**:
   - 各ユーザーアクション → エンドポイント
   - 標準的なREST/GraphQLパターンを使用
   - OpenAPI/GraphQLスキーマを`/contracts/`に出力

3. **コントラクトからコントラクトテストを生成**:
   - エンドポイントごとに1つのテストファイル
   - リクエスト/レスポンススキーマをアサート
   - テストは失敗する必要がある（まだ実装なし）

4. **ユーザーストーリーからテストシナリオを抽出**:
   - 各ストーリー → 統合テストシナリオ
   - クイックスタートテスト = ストーリー検証ステップ

5. **エージェントファイルを段階的に更新** (O(1)操作):
   - AIアシスタント用に`/scripts/update-agent-context.sh [claude|gemini|copilot]`を実行
   - 存在する場合: 現在の計画から新しい技術のみを追加
   - マーカー間の手動追加を保持
   - 最近の変更を更新（最新3つを保持）
   - トークン効率のため150行以下に保持
   - リポジトリルートに出力

**出力**: data-model.md, /contracts/*, 失敗するテスト, quickstart.md, エージェント固有ファイル

## Phase 2: タスク計画アプローチ
*このセクションは/tasksコマンドが実行する内容を説明しています - /plan中は実行しないでください*

**タスク生成戦略**:
- `/templates/tasks-template.md`をベースとして読み込み
- Phase 1設計ドキュメント（contracts, data model, quickstart）からタスクを生成
- 各コントラクト → コントラクトテストタスク [P]
- 各エンティティ → モデル作成タスク [P] 
- 各ユーザーストーリー → 統合テストタスク
- テストを通すための実装タスク

**順序戦略**:
- TDD順序: 実装前にテスト 
- 依存関係順序: モデル → サービス → UI
- [P]で並列実行可能をマーク（独立ファイル）

**予想出力**: tasks.mdに25-30個の番号付き順序タスク

**重要**: このフェーズは/tasksコマンドで実行されます、/planではありません

## Phase 3+: 今後の実装
*これらのフェーズは/planコマンドの範囲外です*

**Phase 3**: タスク実行（/tasksコマンドがtasks.mdを作成）  
**Phase 4**: 実装（憲法原則に従ってtasks.mdを実行）  
**Phase 5**: 検証（テスト実行、quickstart.md実行、性能検証）

## 複雑性追跡
*憲法チェックで正当化が必要な違反がある場合のみ入力*

| 違反 | 必要な理由 | より簡単な代替案が却下された理由 |
|------|-----------|---------------------------|
| [例: 4つ目のプロジェクト] | [現在のニーズ] | [なぜ3プロジェクトでは不十分] |
| [例: Repositoryパターン] | [特定の問題] | [なぜ直接DB アクセスでは不十分] |


## 進捗追跡
*このチェックリストは実行フロー中に更新されます*

**フェーズ状況**:
- [x] Phase 0: 調査完了（/planコマンド）
- [x] Phase 1: 設計完了（/planコマンド）
- [x] Phase 2: タスク計画完了（/planコマンド - アプローチのみ記述）
- [ ] Phase 3: タスク生成（/tasksコマンド）
- [ ] Phase 4: 実装完了
- [ ] Phase 5: 検証通過

**ゲート状況**:
- [x] 初期憲法チェック: 通過
- [x] 設計後憲法チェック: 通過
- [x] すべてのNEEDS CLARIFICATION解決済み
- [x] 複雑性逸脱記録済み（不要）

---
*憲法 v2.1.1に基づく - `/memory/constitution.md`を参照*
