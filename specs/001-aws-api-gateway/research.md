# Research: AWS RAG System

## AWS Bedrock Knowledge Base Integration

**Decision**: Amazon Bedrock Knowledge Base with OpenSearch Serverlessバックエンド  
**Rationale**: 
- マネージドサービスで運用負荷が軽減
- 自動的なドキュメント処理とベクトル化
- RAGに特化した検索・回答生成機能
- Terraformで再現可能な構築

**Alternatives considered**:
- 独自ベクトルDB + Embedding API → 運用コストが高い
- AWS Kendra → RAG専用ではない、コスト高
- OpenSearch自前構築 → 仕様でOpenSearch除外指定

## Go Lambda Function Architecture

**Decision**: 単一Lambda関数で複数エンドポイント処理（Gin Router使用）  
**Rationale**:
- コールドスタート最小化
- 共通ライブラリ再利用
- SAMでのデプロイが簡潔
- AWS SDK v2の効率的な利用

**Alternatives considered**:
- エンドポイント別Lambda → コールドスタート増加
- Lambda関数URL直接 → ルーティング柔軟性不足
- ECS Fargate → サーバーレス要件に不適合

## Frontend Build & Deploy

**Decision**: Vite + React SPA, S3静的ホスティング + CloudFront  
**Rationale**:
- 高速ビルド（Vite）
- TypeScript標準サポート
- S3 + CloudFrontで低コスト配信
- API GatewayとのCORS連携容易

**Alternatives considered**:
- Next.js SSR → サーバーレス要件に複雑性追加
- Webpack → ビルド速度劣る  
- Amplify Hosting → Terraformとの統合複雑

## Document Processing Strategy

**Decision**: S3直接アップロード → Knowledge Base自動インデックス  
**Rationale**:
- Knowledge Baseの自動処理機能活用
- S3イベント連携でリアルタイム更新
- チャンク化・ベクトル化が自動実行
- 50MB制限をS3で前段チェック

**Alternatives considered**:
- Lambda経由アップロード → ファイルサイズ制限、コスト増
- 手動インデックス → リアルタイム性欠如
- バッチ処理 → ユーザビリティ低下

## Testing Strategy

**Decision**: 
- Contract Tests: OpenAPI仕様ベース
- Integration Tests: Localstack使用
- E2E Tests: Cypress + 実際のAWSリソース
- Unit Tests: モック最小限

**Rationale**:
- 実際のAWS API動作を検証
- Localstackでローカル開発可能
- CypressでUI操作含む全フロー検証
- モック依存を避け実動作を保証

**Alternatives considered**:
- 全モックテスト → AWS API変更の検出不可
- 手動テストのみ → 回帰テスト不十分
- Jest のみ → AWS統合テスト困難

## Infrastructure as Code Strategy

**Decision**: 
- Terraform: Bedrock KB, S3, DynamoDB, IAM
- SAM: API Gateway, Lambda, デプロイ
- 分離理由: 各ツールの得意分野活用

**Rationale**:
- Terraformは汎用AWSリソース管理に優れる
- SAMはサーバーレスアプリの統合管理に特化
- CIパイプラインで順次デプロイ可能
- 依存関係の明確な分離

**Alternatives considered**:
- CDK全統一 → Go/TS両対応で複雑性増
- SAM全統一 → Bedrock KB対応不十分  
- Terraform全統一 → Lambda zip管理複雑

## Security & CORS Configuration

**Decision**: 
- API Gateway CORS自動設定
- IAM Role最小権限原則
- VPC不使用（パブリックサブネット）
- CloudFront OAI設定

**Rationale**:
- 認証不要のオープンアクセス要件
- サーバーレスでVPC不要
- 最小権限でセキュリティ確保
- 静的配信の最適化

**Alternatives considered**:
- VPC内Lambda → 認証不要では過剰
- 認証機能追加 → 仕様要件外
- API Key認証 → オープンアクセス要件に反する
