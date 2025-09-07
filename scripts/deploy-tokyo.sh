#!/bin/bash
# AWS RAG System Deployment Script for Tokyo Region
set -e

# 色付きログ出力
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[WARNING] $1${NC}"
}

error() {
    echo -e "${RED}[ERROR] $1${NC}"
    exit 1
}

info() {
    echo -e "${BLUE}[INFO] $1${NC}"
}

# 環境変数設定（prod環境のみサポート）
ENVIRONMENT=${1:-prod}
AWS_REGION="ap-northeast-1"
PROJECT_NAME="poc-ragbkb"

# 環境チェック
if [[ "$ENVIRONMENT" != "prod" ]]; then
    error "❌ サポートされていない環境です: $ENVIRONMENT"
    error "本システムはprod環境のみサポートしています"
    error "使用方法: $0 [prod]"
    exit 1
fi

log "🚀 AWS RAG System デプロイメント開始"
log "環境: $ENVIRONMENT (本番環境)"
log "リージョン: $AWS_REGION (東京)"

# AWS CLI設定確認
info "AWS CLI設定を確認中..."
if ! aws sts get-caller-identity > /dev/null 2>&1; then
    error "AWS CLI認証に失敗しました。aws configure を実行してください。"
fi

# リージョン設定確認
CURRENT_REGION=$(aws configure get region)
if [ "$CURRENT_REGION" != "$AWS_REGION" ]; then
    warn "現在のリージョン: $CURRENT_REGION"
    info "東京リージョンに設定中..."
    aws configure set region $AWS_REGION
fi

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
log "AWS Account ID: $ACCOUNT_ID"
log "AWS Region: $AWS_REGION"

# Phase 1: Infrastructure (Terraform)
log "📦 Phase 1: インフラストラクチャのデプロイ (Terraform)"

cd infrastructure

if [ ! -f ".terraform/terraform.tfstate" ]; then
    info "Terraformを初期化中..."
    terraform init
fi

info "Terraformプランを確認中..."
terraform plan \
    -var="environment=$ENVIRONMENT" \
    -var="aws_region=$AWS_REGION" \
    -var="project_name=$PROJECT_NAME"

read -p "Terraformの変更を適用しますか？ (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    info "Terraformを適用中..."
    terraform apply \
        -var="environment=$ENVIRONMENT" \
        -var="aws_region=$AWS_REGION" \
        -var="project_name=$PROJECT_NAME" \
        -auto-approve
    
    log "✅ インフラストラクチャのデプロイ完了"
else
    warn "インフラストラクチャのデプロイをスキップしました"
fi

# Terraform出力値を取得
S3_BUCKET=$(terraform output -raw s3_bucket_name 2>/dev/null || echo "")
DOCUMENTS_TABLE=$(terraform output -raw documents_table_name 2>/dev/null || echo "")
QUERIES_TABLE=$(terraform output -raw queries_table_name 2>/dev/null || echo "")
RESPONSES_TABLE=$(terraform output -raw responses_table_name 2>/dev/null || echo "")
UPLOAD_SESSIONS_TABLE=$(terraform output -raw upload_sessions_table_name 2>/dev/null || echo "")
KNOWLEDGE_BASE_ID=$(terraform output -raw knowledge_base_id 2>/dev/null || echo "")

cd ..

# Phase 2: Backend (SAM)
log "🔧 Phase 2: バックエンドのデプロイ (SAM)"

cd backend

# Goモジュールの依存関係確認
info "Go依存関係を確認中..."
go mod tidy

# ビルド
info "バックエンドをビルド中..."
make clean && make build

# SAMデプロイ（samconfig.tomlを使用し、動的パラメータを追加）
info "SAMでバックエンドをデプロイ中..."
info "Terraform出力値:"
info "  S3 Bucket: $S3_BUCKET"
info "  Documents Table: $DOCUMENTS_TABLE"  
info "  Queries Table: $QUERIES_TABLE"
info "  Responses Table: $RESPONSES_TABLE"
info "  Upload Sessions Table: $UPLOAD_SESSIONS_TABLE"

# samconfig.tomlが存在することを確認
if [ ! -f samconfig.toml ]; then
    error "❌ samconfig.tomlが見つかりません"
    exit 1
fi

# SAMビルドとデプロイ
sam build --cached
sam deploy \
    --parameter-overrides \
    "Environment=$ENVIRONMENT" \
    "S3BucketName=$S3_BUCKET" \
    "DocumentsTableName=$DOCUMENTS_TABLE" \
    "QueriesTableName=$QUERIES_TABLE" \
    "ResponsesTableName=$RESPONSES_TABLE" \
    "UploadSessionsTableName=$UPLOAD_SESSIONS_TABLE"

# API Gateway URLを取得
API_URL=$(aws cloudformation describe-stacks \
    --stack-name "rag-backend-$ENVIRONMENT" \
    --query 'Stacks[0].Outputs[?OutputKey==`RAGApiUrl`].OutputValue' \
    --output text \
    --region $AWS_REGION)

log "✅ バックエンドのデプロイ完了"
log "API Gateway URL: $API_URL"

cd ..

# Phase 3: Frontend
log "🎨 Phase 3: フロントエンドのデプロイ"

cd frontend

# 依存関係インストール
info "フロントエンド依存関係をインストール中..."
npm install

# 環境変数設定
info "フロントエンド環境変数を設定中..."
cat > .env << EOF
VITE_API_BASE_URL=$API_URL/api
VITE_AWS_REGION=$AWS_REGION
VITE_ENVIRONMENT=$ENVIRONMENT
EOF

# ビルド
info "フロントエンドをビルド中..."
npm run build

# S3バケット作成（フロントエンド用）
FRONTEND_BUCKET="rag-frontend-$ENVIRONMENT-$(date +%s)"
info "フロントエンド用S3バケットを作成中: $FRONTEND_BUCKET"

aws s3 mb s3://$FRONTEND_BUCKET --region $AWS_REGION

# フロントエンドデプロイ
info "フロントエンドをS3にデプロイ中..."
aws s3 sync dist/ s3://$FRONTEND_BUCKET --delete --region $AWS_REGION

# パブリック読み取り権限設定
info "S3バケットのパブリック設定中..."
aws s3api put-bucket-policy --bucket $FRONTEND_BUCKET --policy "$(cat << EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": "*",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::$FRONTEND_BUCKET/*"
    }
  ]
}
EOF
)" --region $AWS_REGION

# ウェブサイトホスティング有効化
aws s3api put-bucket-website --bucket $FRONTEND_BUCKET --website-configuration '{
  "IndexDocument": {"Suffix": "index.html"},
  "ErrorDocument": {"Key": "index.html"}
}' --region $AWS_REGION

FRONTEND_URL="http://$FRONTEND_BUCKET.s3-website-ap-northeast-1.amazonaws.com"

log "✅ フロントエンドのデプロイ完了"
log "フロントエンドURL: $FRONTEND_URL"

cd ..

# Phase 4: デプロイ確認
log "🔍 Phase 4: デプロイ確認"

info "APIヘルスチェック中..."
if curl -s "$API_URL/health" > /dev/null; then
    log "✅ APIは正常に動作しています"
else
    warn "❌ APIヘルスチェックに失敗しました"
fi

# デプロイ情報表示
log "🎉 デプロイメント完了!"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 デプロイメント情報"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🌍 リージョン: $AWS_REGION (東京)"
echo "🏷️  環境: $ENVIRONMENT"
echo ""
echo "🔗 URLs:"
echo "   API Gateway: $API_URL"
echo "   フロントエンド: $FRONTEND_URL"
echo ""
echo "📦 AWS リソース:"
echo "   S3 Bucket (文書): $S3_BUCKET"
echo "   S3 Bucket (フロントエンド): $FRONTEND_BUCKET"
echo "   DynamoDB (Documents): $DOCUMENTS_TABLE"
echo "   DynamoDB (Queries): $QUERIES_TABLE"
echo "   DynamoDB (Responses): $RESPONSES_TABLE"
echo "   DynamoDB (Upload Sessions): $UPLOAD_SESSIONS_TABLE"
if [ -n "$KNOWLEDGE_BASE_ID" ]; then
echo "   Bedrock Knowledge Base: $KNOWLEDGE_BASE_ID"
fi
echo ""
echo "🛠️  CloudFormation Stack: rag-backend-$ENVIRONMENT"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
log "次のステップ:"
echo "1. フロントエンドURLにアクセスしてテスト"
echo "2. 文書をアップロードしてRAGクエリを実行"
echo "3. CloudWatchでログを監視"
echo ""

# 設定ファイル作成
cat > deploy-info-$ENVIRONMENT.json << EOF
{
  "environment": "$ENVIRONMENT",
  "region": "$AWS_REGION",
  "deployedAt": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "urls": {
    "api": "$API_URL",
    "frontend": "$FRONTEND_URL"
  },
  "resources": {
    "s3DocumentsBucket": "$S3_BUCKET",
    "s3FrontendBucket": "$FRONTEND_BUCKET",
    "documentsTable": "$DOCUMENTS_TABLE",
    "queriesTable": "$QUERIES_TABLE",
    "responsesTable": "$RESPONSES_TABLE",
    "uploadSessionsTable": "$UPLOAD_SESSIONS_TABLE",
    "knowledgeBaseId": "$KNOWLEDGE_BASE_ID",
    "cloudFormationStack": "rag-backend-$ENVIRONMENT"
  }
}
EOF

info "デプロイ情報をdeploy-info-$ENVIRONMENT.jsonに保存しました"
log "デプロイメントが正常に完了しました! 🎊"