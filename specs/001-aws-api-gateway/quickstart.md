# Quickstart Guide: AWS RAG System

## Prerequisites

- AWS Account with appropriate permissions
- Go 1.21+
- Node.js 18+
- Terraform 1.0+
- AWS SAM CLI 1.0+
- Docker (for local testing)

## Quick Setup (15 minutes)

### 1. Infrastructure Deployment

```bash
# Clone repository
git clone <repository-url>
cd poc-ragbkb

# Deploy infrastructure (Terraform)
cd infrastructure
terraform init
terraform plan -var="environment=dev"
terraform apply -auto-approve

# Deploy API (SAM)
cd ../backend
sam build --use-container
sam deploy --guided  # First time only
```

### 2. Frontend Setup

```bash
# Install dependencies
cd frontend
npm install

# Start development server
npm run dev
# → http://localhost:5173
```

### 3. Test Document Upload

```bash
# Create test document
echo "AWS Bedrock Knowledge Baseは強力なRAGシステムです。" > test-doc.txt

# Upload via UI or API
curl -X POST http://localhost:5173/api/documents \
  -H "Content-Type: application/json" \
  -d '{
    "fileName": "test-doc.txt",
    "fileSize": 100,
    "fileType": "txt"
  }'
```

### 4. Test Question-Answer

```bash
# Submit question
curl -X POST http://localhost:5173/api/queries \
  -H "Content-Type: application/json" \
  -d '{
    "question": "Bedrock Knowledge Baseとは何ですか？",
    "sessionId": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

## User Journey Validation

### Journey 1: Document Upload & Processing
**Goal**: Verify document upload and Knowledge Base integration

1. **Access Application**
   ```
   → Navigate to http://localhost:5173
   ✓ Page loads without errors
   ✓ Upload interface visible
   ```

2. **Upload Document**
   ```
   → Select test-doc.txt file
   → Click "Upload"
   ✓ Upload progress shown
   ✓ Status changes: uploading → processing → ready
   ✓ Document appears in document list
   ```

3. **Verify Knowledge Base**
   ```
   → Check AWS Console: Bedrock Knowledge Base
   ✓ Document indexed successfully
   ✓ Vectors generated
   ```

### Journey 2: Question-Answer Flow
**Goal**: Verify RAG functionality end-to-end

1. **Submit Question**
   ```
   → Type: "Bedrock Knowledge Baseとは何ですか？"
   → Click "Ask"
   ✓ Processing indicator shown
   ✓ Response generated within 5 seconds
   ```

2. **Verify Response Quality**
   ```
   ✓ Answer includes relevant information
   ✓ Source documents listed
   ✓ Confidence scores displayed
   ✓ Answer relates to uploaded document
   ```

3. **Check History**
   ```
   → View session history
   ✓ Previous questions shown
   ✓ Answers preserved
   ✓ Timestamps accurate
   ```

### Journey 3: Error Handling
**Goal**: Verify system gracefully handles errors

1. **File Size Limit**
   ```
   → Upload 101MB file
   ✓ Error message: "File too large (max 50MB)"
   ✓ Upload rejected
   ```

2. **Unsupported Format**
   ```
   → Upload .pdf file
   ✓ Error message: "Only .txt and .md files supported"
   ✓ Upload rejected
   ```

3. **No Relevant Documents**
   ```
   → Ask: "量子コンピュータとは？"
   ✓ Response: "関連情報が見つかりません"
   ✓ Suggestion to upload relevant documents
   ```

## Performance Benchmarks

### Response Time Targets
- Document upload initiation: < 1 second
- Question processing: < 5 seconds
- Page load: < 2 seconds
- File processing: < 30 seconds (per document)

### Concurrent User Testing
```bash
# Load test script
cd scripts
./load-test.sh --users=3 --duration=60s
# ✓ All 3 users can use system simultaneously
# ✓ No timeouts or errors under normal load
```

## Development Workflow

### 1. Local Development
```bash
# Start all services
make dev-start

# Run tests
make test-all

# Check code quality
make lint
make format
```

### 2. Testing Strategy
```bash
# Unit tests
cd backend && go test ./...
cd frontend && npm test

# Integration tests
make test-integration

# E2E tests
make test-e2e
```

### 3. Deployment Pipeline
```bash
# Deploy to staging
make deploy-staging

# Run smoke tests
make test-smoke

# Deploy to production
make deploy-prod
```

## Troubleshooting

### Common Issues

**1. "Knowledge Base not found"**
```bash
# Check Terraform outputs
terraform output knowledge_base_id
# Verify AWS Console: Bedrock Knowledge Bases
```

**2. "CORS errors in browser"**
```bash
# Check API Gateway CORS settings
aws apigateway get-method --rest-api-id <api-id> --resource-id <resource-id> --http-method OPTIONS
```

**3. "Lambda timeout errors"**
```bash
# Check CloudWatch logs
aws logs describe-log-groups --log-group-name-prefix /aws/lambda/poc-ragbkb
```

### Health Checks

```bash
# API health
curl http://localhost:8080/health

# Knowledge Base status
aws bedrock list-knowledge-bases --region us-east-1

# S3 bucket access
aws s3 ls s3://poc-ragbkb-documents/
```

## Next Steps

1. **Production Readiness**
   - Configure custom domain
   - Set up monitoring/alerting
   - Implement backup strategy

2. **Feature Extensions**
   - Add PDF support
   - Implement user sessions
   - Add conversation memory

3. **Performance Optimization**
   - Enable CloudFront caching
   - Optimize Lambda cold starts
   - Implement connection pooling

## Support

- **Documentation**: `/docs/`
- **API Reference**: `/contracts/api-spec.yaml`
- **Issue Tracking**: GitHub Issues
- **Monitoring**: CloudWatch Dashboard
