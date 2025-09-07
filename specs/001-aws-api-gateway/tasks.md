# Tasks: AWS RAGシステム

**入力**: `/Users/yamadatt/Downloads/poc-ragbkb/poc-ragbkb/specs/001-aws-api-gateway/` からの設計ドキュメント
**前提条件**: plan.md, research.md, data-model.md, contracts/api-spec.yaml, quickstart.md

## 実行フロー (main)
```
1. 機能ディレクトリからplan.mdを読み込み
   → ✓ 実装計画読み込み完了
   → 技術スタック: Go 1.21+, TypeScript 5.0+, React 18, Vite
2. オプション設計ドキュメントを読み込み:
   → data-model.md: Document, Query, Response, UploadSession エンティティ
   → contracts/: 7エンドポイントのOpenAPI仕様
   → research.md: AWS Bedrock KB, Lambda アーキテクチャ決定事項
3. カテゴリ別にタスクを生成:
   → セットアップ: Go/TypeScript プロジェクト、依存関係、Terraform/SAM
   → テスト: 7コントラクトテスト、4統合シナリオ
   → コア: 4モデル、3サービス、7エンドポイント、React コンポーネント
   → 統合: AWSサービス、CORS、ログ
   → 仕上げ: 単体テスト、性能検証、ドキュメント
4. タスクルールを適用:
   → 異なるファイル = [P] で並列実行
   → 実装前にテスト（TDD強制）
5. タスクを順次番号付け（T001-T063）
6. 依存関係グラフと並列実行グループを生成
7. 成功（63タスクが実行準備完了）
```

## 形式: `[ID] [P?] 説明`
- **[P]**: 並列実行可能（異なるファイル、依存関係なし）
- 説明に正確なファイルパスを含める

## パス規約
**Webアプリケーション構造（plan.mdより）**:
- **バックエンド**: `backend/src/`, `backend/tests/`
- **フロントエンド**: `frontend/src/`, `frontend/tests/`
- **インフラストラクチャ**: `infrastructure/` (Terraform), `backend/` (SAM template)

## Phase 3.1: セットアップ・インフラ
- [x] T001 backend/ と frontend/ ディレクトリでプロジェクト構造を作成
- [x] T002 backend/ で Go モジュールを初期化、go.mod 依存関係設定（AWS SDK v2, Gin, testify）
- [x] T003 [P] frontend/ で TypeScript プロジェクトを初期化、package.json 設定（React 18, Vite, Vitest）
- [x] T004 [P] frontend/.eslintrc.js で ESLint と Prettier を設定
- [x] T005 [P] backend/.golangci.yml で golangci-lint による Go リンティング設定
- [x] T006 [P] infrastructure/main.tf で Bedrock Knowledge Base 用 Terraform 設定作成
- [x] T007 [P] backend/ で API Gateway と Lambda 用 SAM template.yaml 作成
- [x] T008 [P] dev-start, test-all, lint, deploy コマンドを含む Makefile 作成

## Phase 3.2: テスト先行（TDD）⚠️ 3.3より前に必ず完了
**重要: これらのテストは実装前に必ず作成し、失敗することを確認すること**

### コントラクトテスト（contracts/api-spec.yaml のAPIエンドポイント）
- [x] T009 [P] GET /health のコントラクトテスト in backend/tests/contract/health_test.go
- [x] T010 [P] GET /documents のコントラクトテスト in backend/tests/contract/documents_list_test.go  
- [x] T011 [P] POST /documents のコントラクトテスト in backend/tests/contract/documents_create_test.go
- [x] T012 [P] GET /documents/{id} のコントラクトテスト in backend/tests/contract/documents_get_test.go
- [x] T013 [P] DELETE /documents/{id} のコントラクトテスト in backend/tests/contract/documents_delete_test.go
- [x] T014 [P] POST /documents/{id}/complete-upload のコントラクトテスト in backend/tests/contract/documents_complete_test.go
- [x] T015 [P] POST /queries のコントラクトテスト in backend/tests/contract/queries_create_test.go
- [x] T016 [P] GET /queries/{sessionId}/history のコントラクトテスト in backend/tests/contract/queries_history_test.go

### 統合テスト（quickstart.md のユーザーシナリオ）
- [x] T017 [P] 文書アップロードフローの統合テスト in backend/tests/integration/document_upload_test.go
- [x] T018 [P] 質問応答フローの統合テスト in backend/tests/integration/rag_query_test.go
- [x] T019 [P] エラーハンドリング統合テスト（ファイルサイズ制限）in backend/tests/integration/error_handling_test.go
- [x] T020 [P] Knowledge Base 同期の統合テスト in backend/tests/integration/kb_sync_test.go

### フロントエンド統合テスト
- [x] T021 [P] 文書アップロードフローのE2Eテスト in frontend/tests/e2e/document-upload.test.ts
- [x] T022 [P] 質問応答フローのE2Eテスト in frontend/tests/e2e/query-response.test.ts
- [x] T023 [P] セッション履歴のE2Eテスト in frontend/tests/e2e/history.test.ts

## Phase 3.3: コア実装（テストが失敗することを確認した後のみ）

### データモデル（data-model.md から4エンティティ）
- [x] T024 [P] Document エンティティ in backend/src/models/document.go
- [x] T025 [P] Query エンティティ in backend/src/models/query.go
- [x] T026 [P] Response エンティティ in backend/src/models/response.go
- [x] T027 [P] UploadSession エンティティ in backend/src/models/upload_session.go

### サービス層（3つのコアサービス）
- [x] T028 DocumentService CRUD 操作 in backend/src/services/document_service.go
- [x] T029 RAG 処理用 QueryService in backend/src/services/query_service.go
- [x] T030 Bedrock 統合用 KnowledgeBaseService in backend/src/services/kb_service.go

### APIエンドポイント（contracts/ から7エンドポイント）
- [x] T031 GET /health エンドポイント in backend/src/handlers/health_handler.go
- [x] T032 GET /documents エンドポイント in backend/src/handlers/documents_handler.go
- [x] T033 POST /documents エンドポイント in backend/src/handlers/documents_handler.go
- [x] T034 GET /documents/{id} エンドポイント in backend/src/handlers/documents_handler.go
- [x] T035 DELETE /documents/{id} エンドポイント in backend/src/handlers/documents_handler.go
- [x] T036 POST /documents/{id}/complete-upload エンドポイント in backend/src/handlers/documents_handler.go
- [x] T037 POST /queries エンドポイント in backend/src/handlers/queries_handler.go
- [x] T038 GET /queries/{sessionId}/history エンドポイント in backend/src/handlers/queries_handler.go

### フロントエンドコンポーネント（React/TypeScript）
- [x] T039 [P] 文書アップロードコンポーネント in frontend/src/components/DocumentUpload.tsx
- [x] T040 [P] 質問入力コンポーネント in frontend/src/components/QueryInput.tsx
- [x] T041 [P] 応答表示コンポーネント in frontend/src/components/ResponseDisplay.tsx
- [x] T042 [P] 文書一覧コンポーネント in frontend/src/components/DocumentList.tsx
- [x] T043 全機能を接続するメインAppコンポーネント in frontend/src/App.tsx

## Phase 3.4: 統合・設定
- [x] T044 CORS とセキュリティヘッダー設定 in backend/src/handlers/middleware.go
- [x] T045 リクエスト/レスポンス ログミドルウェア in backend/src/handlers/middleware.go
- [x] T046 エラーハンドリングミドルウェア in backend/src/handlers/middleware.go
- [x] T047 AWS DynamoDB へのサービス接続 in backend/src/main.go
- [x] T048 AWS Bedrock クライアント設定 in backend/src/main.go
- [x] T049 文書保存用 S3 クライアント設定 in backend/src/main.go
- [x] T050 環境設定 in backend/src/main.go
- [x] T051 フロントエンド API クライアント設定 in frontend/src/services/api.ts
- [x] T052 フロントエンド ルーティング設定 in frontend/src/routes/index.tsx

## Phase 3.5: 仕上げ・検証
- [ ] T053 [P] Document 検証の単体テスト in backend/tests/unit/document_validation_test.go
- [ ] T054 [P] Query 処理の単体テスト in backend/tests/unit/query_processing_test.go
- [ ] T055 [P] Response フォーマットの単体テスト in backend/tests/unit/response_format_test.go
- [ ] T056 [P] フロントエンドコンポーネント単体テスト in frontend/tests/unit/components.test.tsx
- [ ] T057 性能テスト：3同時ユーザー in backend/tests/performance/concurrent_users_test.go
- [ ] T058 性能テスト：5秒以内レスポンス in backend/tests/performance/response_time_test.go
- [ ] T059 [P] API ドキュメント更新 in docs/api.md
- [ ] T060 [P] デプロイメントガイド作成 in docs/deployment.md
- [ ] T061 quickstart.md 検証シナリオ実行
- [ ] T062 コード品質レビューとリファクタリング
- [ ] T063 セキュリティレビューと OWASP コンプライアンス確認

## 依存関係
```
セットアップ (T001-T008) → テスト (T009-T023) → モデル (T024-T027) → サービス (T028-T030) → 
エンドポイント (T031-T038) → フロントエンド (T039-T043) → 統合 (T044-T052) → 仕上げ (T053-T063)

クリティカルパス:
T001 → T002,T003 → T009-T023 → T024-T027 → T028-T030 → T031-T038 → T044-T052 → T061
```

### ブロッキング関係
- T002 が T024-T027 をブロック（モデルに Go プロジェクトが必要）
- T003 が T039-T043 をブロック（コンポーネントに TypeScript プロジェクトが必要）
- T024-T027 が T028-T030 をブロック（サービスにモデルが必要）
- T028-T030 が T031-T038 をブロック（エンドポイントにサービスが必要）
- T047-T050 が T061 をブロック（検証に AWS 設定が必要）
- テスト T009-T023 は T024-T063 より前に作成し、失敗することを確認する必要あり

## 並列実行例

### Phase 3.1 セットアップ（同時実行可能）
```bash
タスク: "backend/ で依存関係を含む Go モジュール初期化"
タスク: "frontend/ で React/Vite を含む TypeScript プロジェクト初期化"
タスク: "ESLint と Prettier の設定"
タスク: "Go リンティングの設定"
タスク: "Terraform 設定作成"
タスク: "SAM テンプレート作成"
タスク: "Makefile 作成"
```

### Phase 3.2 コントラクトテスト（同時実行可能）
```bash
タスク: "GET /health のコントラクトテスト"
タスク: "GET /documents のコントラクトテスト"
タスク: "POST /documents のコントラクトテスト"
タスク: "POST /queries のコントラクトテスト"
タスク: "文書アップロードフローの統合テスト"
タスク: "質問応答フローの統合テスト"
```

### Phase 3.3 モデル（同時実行可能）
```bash
タスク: "backend/src/models/document.go で Document エンティティ"
タスク: "backend/src/models/query.go で Query エンティティ"
タスク: "backend/src/models/response.go で Response エンティティ"
タスク: "backend/src/models/upload_session.go で UploadSession エンティティ"
```

### Phase 3.3 フロントエンドコンポーネント（同時実行可能）
```bash
タスク: "文書アップロードコンポーネント"
タスク: "質問入力コンポーネント"
タスク: "応答表示コンポーネント"
タスク: "文書一覧コンポーネント"
```

### Phase 3.5 単体テスト（同時実行可能）
```bash
タスク: "Document 検証の単体テスト"
タスク: "Query 処理の単体テスト"
タスク: "Response フォーマットの単体テスト"
タスク: "フロントエンドコンポーネント単体テスト"
```

## 注記
- [P] タスク = 異なるファイル、相互依存なし
- **RED-GREEN-REFACTOR**: 実装前にテストの失敗を確認
- 各タスク完了後にコミット
- 統合テストでは実際のAWSサービスを使用（モックではない）
- 憲法原則に従う: TDD、ライブラリ、CLIコマンド
- コードの明確性のため日本語コメントを歓迎

## 適用されたタスク生成ルール
*main() 実行中に適用*

1. **コントラクトから (contracts/api-spec.yaml)**:
   - 8エンドポイント → 8コントラクトテストタスク [P] (T009-T016)
   - 8エンドポイント → 8実装タスク (T031-T038)

2. **データモデルから (data-model.md)**:
   - 4エンティティ → 4モデル作成タスク [P] (T024-T027)
   - 関係性 → 3サービス層タスク (T028-T030)

3. **ユーザーストーリーから (quickstart.md)**:
   - 4シナリオ → 4統合テスト [P] (T017-T020)
   - 3フロントエンドフロー → 3E2Eテスト [P] (T021-T023)

4. **技術スタックから (plan.md)**:
   - Go + TypeScript → デュアルプロジェクト設定 (T002-T003)
   - React コンポーネント → 4コンポーネントタスク [P] (T039-T042)
   - AWS サービス → 4統合タスク (T047-T050)

## 検証チェックリスト
*ゲート: main() で完了前にチェック*

- [x] 全コントラクトに対応するテストあり (T009-T016 → contracts/)
- [x] 全エンティティにモデルタスクあり (T024-T027 → data-model.md エンティティ)
- [x] 全テストが実装前に来る (T009-T023 が T024-T063 より前)
- [x] 並列タスクは真に独立 (異なるファイル、共有状態なし)
- [x] 各タスクが正確なファイルパスを指定
- [x] 他の [P] タスクと同一ファイルを変更するタスクなし
- [x] TDD順序強制: テスト → モデル → サービス → エンドポイント
- [x] 憲法準拠: ライブラリ構造、CLI サポート、可観測性