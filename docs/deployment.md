# AWS RAG System デプロイメントガイド

## 概要

このドキュメントは AWS RAG System のデプロイメント手順を説明します。
システムは以下の AWS サービスを使用します：

- **AWS Lambda** + **API Gateway**: バックエンドAPI
- **AWS Bedrock Knowledge Base**: RAG処理とベクトル検索
- **Amazon DynamoDB**: メタデータストレージと文書プレビュー保存
- **Amazon S3**: 文書ストレージ（プレサインドURL対応）
- **AWS CloudFront** + **S3**: フロントエンド静的ホスティング
- **Amazon OpenSearch Serverless**: ベクトル検索エンジン
- **AWS SAM**: バックエンドのインフラ管理とデプロイ

## 前提条件

### 必要なツールとアクセス権

- **AWS CLI** v2.0以上
- **SAM CLI** v1.0以上
- **Terraform** v1.0以上
- **Node.js** v18以上
- **Go** v1.21以上
- **AWS アカウント** と適切な権限

### AWS権限要件

デプロイに必要な権限：
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "lambda:*",
        "apigateway:*",
        "iam:*",
        "s3:*",
        "dynamodb:*",
        "bedrock:*",
        "cloudformation:*",
        "cloudfront:*"
      ],
      "Resource": "*"
    }
  ]
}
```

---

## デプロイメント手順

### Phase 1: 環境準備

#### 1.1 リポジトリクローンと設定

```bash
# リポジトリクローン
git clone <repository-url>
cd poc-ragbkb

# 環境変数設定 - 東京リージョン
export AWS_REGION=ap-northeast-1
export AWS_PROFILE=your-profile
export ENVIRONMENT=dev  # dev, staging, prod

# AWS CLIでリージョン確認
aws configure get region
# または
aws configure set region ap-northeast-1
```

#### 1.2 依存関係インストール

```bash
# バックエンド依存関係
cd backend
go mod tidy

# フロントエンド依存関係
cd ../frontend
npm install

cd ..
```

### Phase 2: インフラストラクチャ（Terraform）

#### 2.1 Bedrock Knowledge Base

```bash
cd infrastructure
terraform init
terraform plan -var="environment=$ENVIRONMENT"
terraform apply -var="environment=$ENVIRONMENT"
```

**作成されるリソース:**
- S3 Bucket (文書ストレージ)
- OpenSearch Serverless Collection
- Bedrock Knowledge Base
- IAM Roles and Policies

#### 2.2 Terraformの設定内容

`infrastructure/main.tf`:
```hcl
# S3 Bucket for documents
resource "aws_s3_bucket" "documents" {
  bucket = "rag-documents-${var.environment}-${random_id.bucket_suffix.hex}"
}

# OpenSearch Serverless Collection
resource "aws_opensearchserverless_collection" "vector_search" {
  name = "rag-vectors-${var.environment}"
  type = "VECTORSEARCH"
}

# Bedrock Knowledge Base
resource "aws_bedrockagent_knowledge_base" "main" {
  name        = "rag-kb-${var.environment}"
  description = "RAG System Knowledge Base"
  role_arn    = aws_iam_role.bedrock_execution_role.arn
  
  knowledge_base_configuration {
    vector_knowledge_base_configuration {
      embedding_model_arn = "arn:aws:bedrock:${data.aws_region.current.name}::foundation-model/amazon.titan-embed-text-v1"
    }
    type = "VECTOR"
  }

  storage_configuration {
    type = "OPENSEARCH_SERVERLESS"
    opensearch_serverless_configuration {
      collection_arn    = aws_opensearchserverless_collection.vector_search.arn
      vector_index_name = "rag-index"
      field_mapping {
        vector_field   = "vector"
        text_field     = "text"
        metadata_field = "metadata"
      }
    }
  }
}
```

#### 2.3 出力値の確認

```bash
terraform output
```

出力例:
```
knowledge_base_id = "ABCDEFGHIJ"
s3_bucket_name = "rag-documents-dev-a1b2c3d4"
opensearch_collection_endpoint = "https://xxx.aoss.amazonaws.com"
```

### Phase 3: バックエンドデプロイ（SAM）

#### 3.1 SAM設定

`backend/template.yaml` の主要設定:
```yaml
AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31

Parameters:
  Environment:
    Type: String
    Default: dev
  KnowledgeBaseId:
    Type: String
  DocumentsBucket:
    Type: String

Globals:
  Function:
    Runtime: go1.x
    Timeout: 30
    MemorySize: 512
    Environment:
      Variables:
        ENVIRONMENT: !Ref Environment
        KNOWLEDGE_BASE_ID: !Ref KnowledgeBaseId
        DOCUMENTS_BUCKET: !Ref DocumentsBucket

Resources:
  # DynamoDB Tables
  DocumentsTable:
    Type: AWS::DynamoDB::Table
    Properties:
      TableName: !Sub 'rag-documents-${Environment}'
      BillingMode: PAY_PER_REQUEST
      AttributeDefinitions:
        - AttributeName: id
          AttributeType: S
      KeySchema:
        - AttributeName: id
          KeyType: HASH

  # API Gateway
  Api:
    Type: AWS::Serverless::Api
    Properties:
      StageName: !Ref Environment
      Cors:
        AllowMethods: "'GET,POST,PUT,DELETE,OPTIONS'"
        AllowHeaders: "'Content-Type,X-Amz-Date,Authorization,X-Api-Key'"
        AllowOrigin: "'*'"

  # Lambda Functions
  HealthFunction:
    Type: AWS::Serverless::Function
    Properties:
      CodeUri: src/
      Handler: health
      Events:
        Api:
          Type: Api
          Properties:
            RestApiId: !Ref Api
            Path: /health
            Method: GET

  DocumentsFunction:
    Type: AWS::Serverless::Function
    Properties:
      CodeUri: src/
      Handler: documents
      Events:
        List:
          Type: Api
          Properties:
            RestApiId: !Ref Api
            Path: /documents
            Method: GET
        Create:
          Type: Api
          Properties:
            RestApiId: !Ref Api
            Path: /documents
            Method: POST
```

#### 3.2 バックエンドビルドとデプロイ

```bash
cd backend

# ビルド
make build

# デプロイ
sam deploy \
  --guided \
  --parameter-overrides \
    Environment=$ENVIRONMENT \
    KnowledgeBaseId=$(terraform -chdir=../infrastructure output -raw knowledge_base_id) \
    DocumentsBucket=$(terraform -chdir=../infrastructure output -raw s3_bucket_name)
```

初回デプロイ時の質問への回答例:
```
Stack Name: rag-backend-dev
AWS Region: ap-northeast-1
Confirm changes before deploy: Y
Allow SAM CLI IAM role creation: Y
Save parameters to samconfig.toml: Y
```

#### 3.3 デプロイ確認

```bash
# スタック情報確認
aws cloudformation describe-stacks --stack-name rag-backend-dev

# API Gateway URL取得
aws cloudformation describe-stacks \
  --stack-name rag-backend-dev \
  --query 'Stacks[0].Outputs[?OutputKey==`ApiUrl`].OutputValue' \
  --output text
```

### Phase 4: フロントエンドデプロイ

#### 4.1 フロントエンド機能

現在のフロントエンドには以下の機能が実装されています：

**文書管理:**
- ドラッグ&ドロップによるファイルアップロード
- 文書一覧の表形式表示（コンパクト設計）
- 文書プレビュー機能（モーダル表示）
- 文書削除機能
- アップロード進捗表示

**RAGクエリ:**
- リアルタイム質問応答
- セッション管理
- ソース表示（信頼度スコア付き）
- チャット履歴管理

**UI/UX:**
- レスポンシブデザイン
- アクセシビリティ対応
- リアルタイムAPIステータス表示

#### 4.2 環境設定

```bash
cd frontend

# 環境変数設定
export VITE_API_BASE_URL="https://your-api-gateway-url/prod"

# または .env ファイルに記載
echo "VITE_API_BASE_URL=https://your-api-gateway-url/prod" > .env
```

#### 4.3 ビルドとデプロイ

```bash
# ビルド
npm run build

# S3バケット作成（フロントエンド用）
aws s3 mb s3://rag-frontend-$ENVIRONMENT-$(date +%s)

# デプロイ
aws s3 sync dist/ s3://your-frontend-bucket --delete

# パブリック読み取り権限設定
aws s3api put-bucket-policy --bucket your-frontend-bucket --policy '{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": "*",
    "Action": "s3:GetObject",
    "Resource": "arn:aws:s3:::your-frontend-bucket/*"
  }]
}'

# ウェブサイトホスティング有効化
aws s3api put-bucket-website --bucket your-frontend-bucket --website-configuration '{
  "IndexDocument": {"Suffix": "index.html"},
  "ErrorDocument": {"Key": "index.html"}
}'
```

#### 4.3 CloudFront設定（推奨）

```bash
# CloudFront Distribution作成
aws cloudfront create-distribution --distribution-config '{
  "CallerReference": "'$(date +%s)'",
  "Comment": "RAG System Frontend",
  "DefaultCacheBehavior": {
    "TargetOriginId": "S3Origin",
    "ViewerProtocolPolicy": "redirect-to-https",
    "TrustedSigners": {
      "Enabled": false,
      "Quantity": 0
    },
    "ForwardedValues": {
      "QueryString": false,
      "Cookies": {"Forward": "none"}
    },
    "MinTTL": 0
  },
  "Origins": {
    "Quantity": 1,
    "Items": [{
      "Id": "S3Origin",
      "DomainName": "your-frontend-bucket.s3.amazonaws.com",
      "S3OriginConfig": {
        "OriginAccessIdentity": ""
      }
    }]
  },
  "Enabled": true,
  "PriceClass": "PriceClass_100"
}'
```

---

## 環境別設定

### 開発環境 (dev) - 東京リージョン
```bash
export ENVIRONMENT=dev
export AWS_REGION=ap-northeast-1
```
- 最小リソース構成
- デバッグログ有効
- CORS: 全オリジン許可
- **東京リージョン**: データとユーザーが地理的に近く、レイテンシ最小化

### ステージング環境 (staging) - 東京リージョン
```bash
export ENVIRONMENT=staging
export AWS_REGION=ap-northeast-1
```
- 本番同等構成
- ログレベル: INFO
- CORS: 特定ドメインのみ
- **東京リージョン**: 本番と同一リージョンでテスト

### 本番環境 (prod) - 東京リージョン
```bash
export ENVIRONMENT=prod
export AWS_REGION=ap-northeast-1
```
- 高可用性構成
- ログレベル: WARN
- セキュリティ強化設定
- **東京リージョン**: 日本のユーザーに最適化

---

## 設定管理

### 環境変数一覧

#### バックエンド (Lambda)
| 変数名 | 説明 | 例 |
|--------|------|-----|
| `ENVIRONMENT` | 環境識別子 | dev, staging, prod |
| `AWS_REGION` | AWSリージョン | ap-northeast-1 |
| `KNOWLEDGE_BASE_ID` | Bedrock Knowledge Base ID | ABCDEFGHIJ |
| `DOCUMENTS_BUCKET` | S3バケット名 | rag-documents-dev-xxx |
| `DOCUMENTS_TABLE` | DynamoDB テーブル名 | rag-documents-dev |
| `QUERIES_TABLE` | DynamoDB テーブル名 | rag-queries-dev |
| `LOG_LEVEL` | ログレベル | DEBUG, INFO, WARN, ERROR |

#### フロントエンド
| 変数名 | 説明 | 例 |
|--------|------|-----|
| `VITE_API_BASE_URL` | API Gateway URL | https://xxx.execute-api.ap-northeast-1.amazonaws.com/dev/api |

### 設定ファイル

#### samconfig.toml
```toml
version = 0.1
[default.deploy.parameters]
stack_name = "rag-backend-dev"
s3_bucket = "sam-deployment-bucket"
s3_prefix = "rag-backend"
region = "ap-northeast-1"
capabilities = "CAPABILITY_IAM"
parameter_overrides = [
  "Environment=dev",
  "KnowledgeBaseId=ABCDEFGHIJ",
  "DocumentsBucket=rag-documents-dev-xxx"
]
```

---

## 監視・ログ

### CloudWatch ログ確認

```bash
# Lambda関数ログ
aws logs describe-log-groups --log-group-name-prefix /aws/lambda/rag-backend

# 特定の関数のログ
aws logs tail /aws/lambda/rag-backend-dev-DocumentsFunction --follow

# エラーログフィルタリング
aws logs filter-log-events \
  --log-group-name /aws/lambda/rag-backend-dev-DocumentsFunction \
  --filter-pattern "ERROR"
```

### メトリクス監視

```bash
# API Gateway メトリクス
aws cloudwatch get-metric-statistics \
  --namespace AWS/ApiGateway \
  --metric-name Count \
  --dimensions Name=ApiName,Value=rag-backend-dev \
  --start-time 2024-01-01T00:00:00Z \
  --end-time 2024-01-01T23:59:59Z \
  --period 3600 \
  --statistics Sum

# Lambda メトリクス
aws cloudwatch get-metric-statistics \
  --namespace AWS/Lambda \
  --metric-name Duration \
  --dimensions Name=FunctionName,Value=rag-backend-dev-QueryFunction \
  --start-time 2024-01-01T00:00:00Z \
  --end-time 2024-01-01T23:59:59Z \
  --period 300 \
  --statistics Average,Maximum
```

---

## セキュリティ設定

### IAMロールと権限

#### Lambda実行ロール
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:UpdateItem",
        "dynamodb:DeleteItem",
        "dynamodb:Query",
        "dynamodb:Scan"
      ],
      "Resource": [
        "arn:aws:dynamodb:*:*:table/rag-documents-*",
        "arn:aws:dynamodb:*:*:table/rag-queries-*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject"
      ],
      "Resource": "arn:aws:s3:::rag-documents-*/*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "bedrock:RetrieveAndGenerate",
        "bedrock:Retrieve"
      ],
      "Resource": "*"
    }
  ]
}
```

### ネットワークセキュリティ

#### API Gateway設定
```yaml
# CORS設定（本番環境）
Cors:
  AllowMethods: "'GET,POST,DELETE,OPTIONS'"
  AllowHeaders: "'Content-Type,Authorization'"
  AllowOrigin: "'https://yourdomain.com'"

# スロットリング設定
ThrottleConfig:
  RateLimit: 100
  BurstLimit: 200
```

### データ暗号化

#### S3暗号化
```bash
# S3バケット暗号化有効化
aws s3api put-bucket-encryption --bucket your-bucket --server-side-encryption-configuration '{
  "Rules": [{
    "ApplyServerSideEncryptionByDefault": {
      "SSEAlgorithm": "AES256"
    }
  }]
}'
```

#### DynamoDB暗号化
```yaml
# SAM Template
DocumentsTable:
  Type: AWS::DynamoDB::Table
  Properties:
    SSESpecification:
      SSEEnabled: true
```

---

## トラブルシューティング

### よくある問題と解決方法

#### 1. Terraform Apply エラー
```bash
# エラー: Insufficient permissions
# 解決: IAM権限確認
aws sts get-caller-identity
aws iam get-user

# エラー: Resource already exists
# 解決: 既存リソース確認
terraform import aws_s3_bucket.documents existing-bucket-name
```

#### 2. SAM Deploy エラー
```bash
# エラー: Code size too large
# 解決: ビルド最適化
make clean && make build

# エラー: Template validation failed
# 解決: テンプレート構文確認
sam validate
```

#### 3. Lambda Function エラー
```bash
# エラー: Module not found
# 解決: 依存関係確認
go mod tidy
go mod vendor

# エラー: Timeout
# 解決: タイムアウト設定調整
sam deploy --parameter-overrides Timeout=60
```

#### 4. フロントエンド接続エラー
```bash
# エラー: CORS error
# 解決: API Gateway CORS設定確認
aws apigateway get-resource --rest-api-id YOUR_API_ID --resource-id YOUR_RESOURCE_ID

# エラー: 404 Not Found
# 解決: API URL確認
echo $VITE_API_BASE_URL
```

### ログを使った診断

```bash
# Lambda関数の詳細ログ
aws logs start-query \
  --log-group-name /aws/lambda/rag-backend-dev-QueryFunction \
  --start-time 1640995200 \
  --end-time 1640998800 \
  --query-string 'fields @timestamp, @message | filter @message like /ERROR/'

# DynamoDB アクセスログ
aws dynamodb describe-table --table-name rag-documents-dev
```

---

## メンテナンス

### 定期メンテナンス

#### 1. ログローテーション
```bash
# CloudWatchログの保持期間設定
aws logs put-retention-policy \
  --log-group-name /aws/lambda/rag-backend-dev-QueryFunction \
  --retention-in-days 30
```

#### 2. バックアップ
```bash
# DynamoDB バックアップ
aws dynamodb create-backup \
  --table-name rag-documents-dev \
  --backup-name "rag-documents-backup-$(date +%Y%m%d)"

# S3 バージョニング有効化
aws s3api put-bucket-versioning \
  --bucket your-documents-bucket \
  --versioning-configuration Status=Enabled
```

#### 3. セキュリティアップデート
```bash
# 依存関係更新
go get -u ./...
npm audit fix

# Lambda ランタイム更新
sam deploy --parameter-overrides Runtime=go1.x
```

### スケーリング調整

#### Lambda同時実行数
```bash
# 同時実行数制限設定
aws lambda put-provisioned-concurrency-config \
  --function-name rag-backend-dev-QueryFunction \
  --qualifier '$LATEST' \
  --provisioned-concurrency-count 10
```

#### DynamoDB容量調整
```bash
# オンデマンド → プロビジョンド切り替え
aws dynamodb modify-table \
  --table-name rag-documents-dev \
  --billing-mode PROVISIONED \
  --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5
```

---

## 削除手順

### 環境の完全削除

```bash
# 1. フロントエンドリソース削除
aws s3 rm s3://your-frontend-bucket --recursive
aws s3api delete-bucket --bucket your-frontend-bucket

# 2. バックエンドスタック削除
sam delete --stack-name rag-backend-dev

# 3. インフラストラクチャ削除
cd infrastructure
terraform destroy -var="environment=$ENVIRONMENT"

# 4. ログ削除
aws logs describe-log-groups --log-group-name-prefix /aws/lambda/rag-backend-dev | \
  jq -r '.logGroups[].logGroupName' | \
  xargs -I {} aws logs delete-log-group --log-group-name {}
```

**注意**: 削除は取り消し不可能です。重要なデータは事前にバックアップしてください。

---

## 付録

### デプロイメントスクリプト

`scripts/deploy.sh`:
```bash
#!/bin/bash
set -e

ENVIRONMENT=${1:-dev}
echo "Deploying to environment: $ENVIRONMENT"

# Infrastructure
cd infrastructure
terraform apply -var="environment=$ENVIRONMENT" -auto-approve

# Backend
cd ../backend
sam deploy --parameter-overrides Environment=$ENVIRONMENT

# Frontend
cd ../frontend
npm run build
aws s3 sync dist/ s3://rag-frontend-$ENVIRONMENT --delete

echo "Deployment completed successfully!"
```

### CI/CD Pipeline設定例

`.github/workflows/deploy.yml`:
```yaml
name: Deploy RAG System

on:
  push:
    branches: [main, develop]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup AWS CLI
        uses: aws-actions/configure-aws-credentials@v2
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: ap-northeast-1
      
      - name: Deploy Infrastructure
        run: |
          cd infrastructure
          terraform init
          terraform apply -auto-approve
      
      - name: Deploy Backend
        run: |
          cd backend
          sam deploy --no-confirm-changeset
      
      - name: Deploy Frontend
        run: |
          cd frontend
          npm install
          npm run build
          aws s3 sync dist/ s3://$FRONTEND_BUCKET --delete
```

---

## 連絡先・サポート

- **開発チーム**: dev-team@company.com
- **システム管理**: ops-team@company.com
- **緊急時**: emergency@company.com

---

## 関連ドキュメント

- [API ドキュメント](api.md)
- [開発者ガイド](../README.md)
- [アーキテクチャ設計書](../specs/001-aws-api-gateway/plan.md)