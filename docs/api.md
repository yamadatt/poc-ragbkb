# AWS RAG System API ドキュメント

## 概要

AWS RAG (Retrieval Augmented Generation) System の REST API ドキュメントです。
このシステムは AWS Bedrock Knowledge Base を使用したRAGシステムを提供します。

- **ベースURL**: `/api`
- **認証**: なし（PoC版）
- **フォーマット**: JSON
- **エラー処理**: 統一されたエラーレスポンス形式

## API エンドポイント一覧

### ヘルスチェック

#### GET /health
システムの稼働状況を確認します。

**レスポンス例:**
```json
{
  "data": {
    "status": "healthy",
    "message": "Service is running",
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

---

### 文書管理

#### GET /documents
アップロード済み文書の一覧を取得します。

**クエリパラメータ:**
- `offset` (optional): オフセット (デフォルト: 0)
- `limit` (optional): 取得件数 (デフォルト: 20, 最大: 100)

**レスポンス例:**
```json
{
  "data": {
    "documents": [
      {
        "id": "doc_123456789",
        "fileName": "aws-bedrock-guide.txt",
        "fileSize": 15360,
        "fileType": "txt",
        "status": "ready",
        "preview": "AWS Bedrockは機械学習の基盤モデルを...",
        "previewLines": 30,
        "uploadedAt": "2024-01-01T12:00:00Z",
        "processedAt": "2024-01-01T12:05:00Z"
      }
    ],
    "totalCount": 1,
    "hasMore": false
  }
}
```

**ステータス値:**
- `uploading`: アップロード中
- `processing`: 処理中（Knowledge Base同期中）
- `ready`: 利用可能
- `error`: アップロードエラー
- `kb_sync_error`: Knowledge Base同期エラー（文書は利用可能）

#### GET /documents/{id}
指定された文書の詳細情報を取得します。

**パスパラメータ:**
- `id`: 文書ID

**レスポンス例:**
```json
{
  "data": {
    "id": "doc_123456789",
    "fileName": "aws-bedrock-guide.txt",
    "fileSize": 15360,
    "fileType": "txt",
    "status": "ready",
    "preview": "AWS Bedrockは機械学習の基盤モデルを...",
    "previewLines": 30,
    "uploadedAt": "2024-01-01T12:00:00Z",
    "processedAt": "2024-01-01T12:05:00Z",
    "s3Key": "documents/doc_123456789/aws-bedrock-guide.txt",
    "s3Bucket": "poc-ragbkb-documents-prod",
    "createdAt": "2024-01-01T12:00:00Z",
    "updatedAt": "2024-01-01T12:05:00Z"
  }
}
```

#### POST /documents
新しい文書のアップロードセッションを開始します。

**リクエストボディ:**
```json
{
  "fileName": "sample.txt",
  "fileSize": 1024,
  "fileType": "txt"
}
```

**制約:**
- `fileSize`: 最大 50MB (52,428,800 bytes)
- `fileType`: "txt" または "md" のみ
- `fileName`: 必須、空文字不可

**レスポンス例:**
```json
{
  "data": {
    "id": "upload_session_123",
    "fileName": "sample.txt",
    "fileSize": 1024,
    "fileType": "txt",
    "uploadUrl": "https://s3.amazonaws.com/bucket/presigned-url",
    "expiresAt": "2024-01-01T13:00:00Z"
  }
}
```

#### POST /documents/{id}/complete-upload
S3への直接アップロード完了を通知し、処理を開始します。

**パスパラメータ:**
- `id`: アップロードセッションIDまたは文書ID（互換性維持）

**レスポンス例:**
```json
{
  "data": {
    "id": "doc_123456789",
    "fileName": "sample.txt",
    "fileSize": 1024,
    "fileType": "txt",
    "status": "processing"
  }
}
```

**注意:**
- アップロード完了後、自動的にKnowledge Baseへの同期処理が開始されます
- ステータスは `processing` → `ready` に変わります
- プレビューも自動生成され、文書詳細に含まれます

#### DELETE /documents/{id}
指定された文書を削除します。

**パスパラメータ:**
- `id`: 文書ID

**処理内容:**
1. S3からファイルを削除
2. Knowledge Baseからインデックスを削除（非同期）
3. DynamoDBから文書レコードを削除

**注意:**
- 削除は物理削除（復元不可）です
- Knowledge Baseの更新は非同期で行われるため、削除直後は検索結果に残る可能性があります
- 削除処理のログはCloudWatchで確認できます

**成功時:** 204 No Content

---

### クエリ・RAG処理

#### POST /queries
RAGシステムに質問を送信し、回答を生成します。

**リクエストボディ:**
```json
{
  "question": "AWS Bedrockについて教えてください",
  "sessionId": "session_abc123"
}
```

**制約:**
- `question`: 最大1000文字
- `sessionId`: セッション管理用ID（任意の文字列）

**レスポンス例:**
```json
{
  "data": {
    "query": {
      "id": "query_123456789",
      "sessionId": "session_abc123",
      "question": "AWS Bedrockについて教えてください",
      "status": "completed",
      "processingTimeMs": 2500,
      "createdAt": "2024-01-01T12:00:00Z",
      "updatedAt": "2024-01-01T12:00:02Z"
    },
    "response": {
      "id": "resp_123456789",
      "answer": "AWS Bedrockは、機械学習の基盤モデルを簡単に利用できるフルマネージドサービスです。...",
      "sources": [
        {
          "documentId": "doc_123456789",
          "fileName": "aws-bedrock-guide.txt",
          "excerpt": "AWS Bedrockは基盤モデル（Foundation Models）へのアクセスを提供...",
          "confidence": 0.92
        }
      ],
      "processingTimeMs": 2500,
      "modelUsed": "anthropic.claude-v2",
      "tokensUsed": 150,
      "createdAt": "2024-01-01T12:00:02Z"
    }
  }
}
```

**信頼度スコア:**
- `0.8-1.0`: 高い信頼度
- `0.6-0.8`: 中程度の信頼度
- `0.0-0.6`: 低い信頼度

#### GET /queries/{sessionId}/history
指定されたセッションのクエリ履歴を取得します。

**パスパラメータ:**
- `sessionId`: セッションID

**レスポンス例:**
```json
{
  "data": [
    {
      "query": {
        "id": "query_123456789",
        "sessionId": "session_abc123",
        "question": "AWS Bedrockについて教えてください",
        "status": "completed",
        "processingTimeMs": 2500,
        "createdAt": "2024-01-01T12:00:00Z"
      },
      "response": {
        "id": "resp_123456789",
        "answer": "AWS Bedrockは...",
        "sources": [...],
        "processingTimeMs": 2500,
        "modelUsed": "anthropic.claude-v2",
        "tokensUsed": 150,
        "createdAt": "2024-01-01T12:00:02Z"
      }
    }
  ]
}
```

---

## エラー処理

### エラーレスポンス形式

全てのエラーは以下の統一形式で返されます：

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "エラーの詳細メッセージ",
    "details": {
      "additionalInfo": "追加情報（任意）"
    }
  },
  "meta": {
    "timestamp": "2024-01-01T12:00:00Z",
    "requestId": "req_123456789"
  }
}
```

### HTTPステータスコード

| コード | 説明 | 例 |
|--------|------|-----|
| 200 | 成功 | 正常なレスポンス |
| 201 | 作成成功 | 文書アップロードセッション作成 |
| 204 | 成功（レスポンスボディなし） | 文書削除 |
| 400 | リクエストエラー | 無効なパラメータ |
| 404 | 見つからない | 存在しない文書ID |
| 413 | ペイロード過大 | ファイルサイズ制限超過 |
| 422 | 処理不可 | サポートされていないファイルタイプ |
| 429 | レート制限 | リクエスト過多 |
| 500 | サーバーエラー | 内部エラー |
| 502 | Bad Gateway | AWS サービス接続エラー |
| 503 | サービス利用不可 | システムメンテナンス |

### エラーコード一覧

#### 一般エラー
- `INVALID_REQUEST`: 無効なリクエスト
- `MISSING_PARAMETER`: 必須パラメータ不足
- `INVALID_PARAMETER`: パラメータの値が無効
- `INTERNAL_ERROR`: 内部サーバーエラー

#### 文書管理エラー
- `FILE_TOO_LARGE`: ファイルサイズ制限超過
- `UNSUPPORTED_FILE_TYPE`: サポートされていないファイルタイプ
- `DOCUMENT_NOT_FOUND`: 文書が見つからない
- `UPLOAD_SESSION_EXPIRED`: アップロードセッションの有効期限切れ
- `DOCUMENT_PROCESSING`: 文書が処理中のため操作不可

#### クエリエラー
- `QUESTION_TOO_LONG`: 質問が長すぎる
- `NO_DOCUMENTS_INDEXED`: インデックス済み文書なし
- `PROCESSING_TIMEOUT`: 処理タイムアウト
- `BEDROCK_SERVICE_ERROR`: Bedrock サービスエラー

---

## 使用例

### 1. 文書アップロードフロー

```javascript
// 1. アップロードセッション開始
const uploadSession = await fetch('/api/documents', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    fileName: 'guide.txt',
    fileSize: 1024,
    fileType: 'txt'
  })
});

const { data: session } = await uploadSession.json();

// 2. S3に直接アップロード
await fetch(session.uploadUrl, {
  method: 'PUT',
  body: file,
  headers: { 'Content-Type': 'application/octet-stream' }
});

// 3. アップロード完了通知
const completion = await fetch(`/api/documents/${session.id}/complete-upload`, {
  method: 'POST'
});

const { data: document } = await completion.json();
console.log('アップロード完了:', document.id);
```

### 2. RAGクエリフロー

```javascript
// 質問送信
const response = await fetch('/api/queries', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    question: 'AWS Bedrockの料金体系について教えてください',
    sessionId: 'user_session_123'
  })
});

const { data: result } = await response.json();
console.log('回答:', result.response.answer);
console.log('情報源:', result.response.sources);
```

### 3. セッション履歴取得

```javascript
const history = await fetch('/api/queries/user_session_123/history');
const { data: queries } = await history.json();

queries.forEach(item => {
  console.log(`Q: ${item.query.question}`);
  console.log(`A: ${item.response.answer}`);
});
```

---

## レート制限

現在のPoC版では以下の制限があります：

- **同時接続**: 最大3ユーザー
- **ファイルサイズ**: 最大50MB
- **クエリ**: セッションあたり毎分10回まで
- **アップロード**: ユーザーあたり毎時10ファイルまで

本格運用時には要件に応じて制限を調整します。

---

## パフォーマンス

### レスポンス時間目標

| エンドポイント | 目標時間 | 最大許容時間 |
|----------------|----------|-------------|
| GET /health | 100ms | 500ms |
| GET /documents | 500ms | 2s |
| POST /documents | 1s | 3s |
| POST /queries | 3s | 5s |

### 同時実行性能

- **3同時ユーザー**: 成功率95%以上を維持
- **レスポンス時間**: 95パーセンタイル値が上記目標時間以内
- **スループット**: 最低30リクエスト/分

---

## セキュリティ

### 現在の実装（PoC版）

- **CORS**: すべてのオリジンを許可（開発用）
- **認証**: なし
- **入力検証**: 基本的なバリデーションのみ
- **ログ**: 全リクエスト/レスポンスをログ出力

### 本格運用での推奨事項

- **認証**: JWT トークンベース認証
- **CORS**: 特定ドメインのみ許可
- **レート制限**: IP/ユーザーベースの制限
- **入力サニタイゼーション**: XSS対策の強化
- **監査ログ**: セキュリティイベントの記録

---

## トラブルシューティング

### よくある問題

#### 1. アップロードが失敗する
- ファイルサイズ、タイプを確認
- アップロードセッションの有効期限をチェック
- S3への直接アップロード時のCORSエラーを確認

#### 2. クエリが遅い
- インデックス済み文書数を確認
- Bedrockサービスの可用性をチェック
- ネットワーク接続を確認

#### 3. 文書が検索されない
- 文書のステータスが `indexed` か確認
- Knowledge Baseの同期状況をチェック
- 質問内容と文書内容の関連性を確認

### ログ確認方法

```bash
# アプリケーションログ
kubectl logs -f deployment/rag-backend

# エラーログのフィルタリング
kubectl logs deployment/rag-backend | grep ERROR

# 特定のリクエストID追跡
kubectl logs deployment/rag-backend | grep "req_123456789"
```

---

## 更新履歴

| バージョン | 日付 | 変更内容 |
|------------|------|----------|
| 1.0.0 | 2024-01-01 | 初版リリース |

---

## 関連リンク

- [デプロイメントガイド](deployment.md)
- [開発者ガイド](../README.md)
- [AWS Bedrock ドキュメント](https://docs.aws.amazon.com/bedrock/)
- [OpenAPI仕様書](../specs/001-aws-api-gateway/contracts/api-spec.yaml)
