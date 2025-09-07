import React, { useState, useEffect, useCallback } from 'react';
import './DocumentList.css';

// 文書の状態
type DocumentStatus = 'uploading' | 'processing' | 'ready' | 'error' | 'kb_sync_error';

// 文書リストアイテム
interface Document {
  id: string;
  fileName: string;
  fileSize: number;
  fileType: string;
  status: DocumentStatus;
  uploadedAt: string;
  lastModified?: string;
}

// APIレスポンス型
interface DocumentListResponse {
  data: {
    documents: Document[];
    total: number;
    offset: number;
    limit: number;
    hasMore: boolean;
  };
}

interface DocumentListProps {
  refreshTrigger?: number; // 外部からのリフレッシュトリガー
  onDocumentClick?: (document: Document) => void;
  onDocumentDelete?: (document: Document) => void;
}

const DocumentList: React.FC<DocumentListProps> = ({
  refreshTrigger,
  onDocumentClick,
  onDocumentDelete,
}) => {
  const [documents, setDocuments] = useState<Document[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string>('');
  const [page, setPage] = useState<number>(1);
  const [totalPages, setTotalPages] = useState<number>(1);
  const [totalCount, setTotalCount] = useState<number>(0);
  const [selectedDocuments, setSelectedDocuments] = useState<Set<string>>(new Set());

  // 文書リスト取得
  const fetchDocuments = useCallback(async (pageNumber: number = 1) => {
    try {
      setLoading(true);
      setError('');

      const limit = 10;
      const offset = (pageNumber - 1) * limit;
      const response = await fetch(`https://e5q2b7qvd5.execute-api.ap-northeast-1.amazonaws.com/prod/documents?offset=${offset}&limit=${limit}&sortBy=uploadedAt&sortOrder=desc`);
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error?.message || '文書一覧の取得に失敗しました');
      }

      const data: DocumentListResponse = await response.json();
      
      // フロントエンドでも念のためソート（アップロード日時の降順）
      const sortedDocuments = data.data.documents.sort((a, b) => 
        new Date(b.uploadedAt).getTime() - new Date(a.uploadedAt).getTime()
      );
      
      setDocuments(sortedDocuments);
      setPage(pageNumber);
      setTotalPages(Math.ceil(data.data.total / 10)); // Backend returns page count; keep behavior
      setTotalCount(data.data.total);

    } catch (error) {
      const message = error instanceof Error ? error.message : '予期しないエラーが発生しました';
      setError(message);
      console.error('文書一覧取得エラー:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  // 初回読み込みとリフレッシュトリガー対応
  useEffect(() => {
    fetchDocuments(1);
  }, [fetchDocuments, refreshTrigger]);

  // 文書削除
  const handleDeleteDocument = useCallback(async (document: Document) => {
    if (!window.confirm(`「${document.fileName}」を削除しますか？この操作は元に戻せません。`)) {
      return;
    }

    try {
      const response = await fetch(`https://e5q2b7qvd5.execute-api.ap-northeast-1.amazonaws.com/prod/documents/${document.id}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error?.message || '文書の削除に失敗しました');
      }

      // 成功時のコールバック
      if (onDocumentDelete) {
        onDocumentDelete(document);
      }

      // リストを再取得
      fetchDocuments(page);

    } catch (error) {
      const message = error instanceof Error ? error.message : '予期しないエラーが発生しました';
      setError(message);
      console.error('文書削除エラー:', error);
    }
  }, [page, fetchDocuments, onDocumentDelete]);

  // 複数文書の一括削除
  const handleBulkDelete = useCallback(async () => {
    const selectedDocs = documents.filter(doc => selectedDocuments.has(doc.id));
    
    if (selectedDocs.length === 0) return;

    if (!window.confirm(`選択した${selectedDocs.length}件の文書を削除しますか？この操作は元に戻せません。`)) {
      return;
    }

    try {
      setLoading(true);
      
      // 並列削除実行
      const deletePromises = Array.from(selectedDocuments).map(docId =>
        fetch(`https://e5q2b7qvd5.execute-api.ap-northeast-1.amazonaws.com/prod/documents/${docId}`, { method: 'DELETE' })
      );

      const results = await Promise.allSettled(deletePromises);
      
      // 失敗したものがあるかチェック
      const failures = results.filter(result => result.status === 'rejected');
      if (failures.length > 0) {
        console.warn(`${failures.length}件の削除に失敗しました`);
      }

      // 選択をクリア
      setSelectedDocuments(new Set());
      
      // リストを再取得
      await fetchDocuments(page);

    } catch (error) {
      const message = error instanceof Error ? error.message : '一括削除中にエラーが発生しました';
      setError(message);
      console.error('一括削除エラー:', error);
    } finally {
      setLoading(false);
    }
  }, [documents, selectedDocuments, page, fetchDocuments]);

  // 文書選択の切り替え
  const toggleDocumentSelection = useCallback((docId: string) => {
    setSelectedDocuments(prev => {
      const newSet = new Set(prev);
      if (newSet.has(docId)) {
        newSet.delete(docId);
      } else {
        newSet.add(docId);
      }
      return newSet;
    });
  }, []);

  // 全選択/全解除
  const toggleSelectAll = useCallback(() => {
    if (selectedDocuments.size === documents.length) {
      setSelectedDocuments(new Set());
    } else {
      setSelectedDocuments(new Set(documents.map(doc => doc.id)));
    }
  }, [selectedDocuments.size, documents]);

  // ステータス表示
  const getStatusText = (status: DocumentStatus): string => {
    switch (status) {
      case 'uploading': return 'アップロード中';
      case 'processing': return '処理中';
      case 'ready': return '利用可能';
      case 'error': return 'アップロードエラー';
      case 'kb_sync_error': return 'KB同期エラー';
      default: return status;
    }
  };

  const getStatusClass = (status: DocumentStatus): string => {
    switch (status) {
      case 'uploading': return 'uploading';
      case 'processing': return 'processing';
      case 'ready': return 'indexed';
      case 'error': return 'error';
      case 'kb_sync_error': return 'kb_sync_error';
      default: return '';
    }
  };

  // ファイルサイズフォーマット
  const formatFileSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
    return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
  };

  // 日時フォーマット
  const formatDateTime = (isoString: string): string => {
    return new Date(isoString).toLocaleString('ja-JP', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  // ページ変更
  const handlePageChange = useCallback((newPage: number) => {
    if (newPage >= 1 && newPage <= totalPages) {
      fetchDocuments(newPage);
    }
  }, [totalPages, fetchDocuments]);

  if (loading && documents.length === 0) {
    return (
      <div className="document-list">
        <div className="loading-container">
          <div className="loading-spinner" />
          <p>文書一覧を読み込み中...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="document-list">
      <div className="list-header">
        <div>
          <h3>📂 アップロード済み文書</h3>
          <div className="list-stats">合計 {totalCount} 件の文書</div>
        </div>
        
        {documents.length > 0 && (
          <div className="list-actions">
            {selectedDocuments.size > 0 && (
              <div className="bulk-actions">
                <button
                  className="delete-selected-button"
                  onClick={handleBulkDelete}
                  disabled={loading}
                >
                  🗑️ 選択した{selectedDocuments.size}件を削除
                </button>
              </div>
            )}
            
            <button
              className="refresh-button"
              onClick={() => fetchDocuments(page)}
              disabled={loading}
              title="リストを更新"
            >
              🔄 更新
            </button>
          </div>
        )}
      </div>

      {error && (
        <div className="error-container">
          <div className="error-message">エラーが発生しました</div>
          <p>{error}</p>
          <button className="retry-button" onClick={() => fetchDocuments(page)}>
            再試行
          </button>
        </div>
      )}

      {documents.length === 0 && !loading ? (
        <div className="empty-state">
          <div className="empty-icon">📄</div>
          <div className="empty-title">文書がありません</div>
          <div className="empty-description">文書をアップロードすると、ここに表示されます。</div>
        </div>
      ) : (
        <>
          <div className="documents-table">
            <div className="table-header">
              <div className="header-checkbox">
                <input
                  type="checkbox"
                  checked={documents.length > 0 && selectedDocuments.size === documents.length}
                  onChange={toggleSelectAll}
                  title="全選択/全解除"
                />
              </div>
              <div className="header-name">文書名</div>
              <div className="header-size">サイズ</div>
              <div className="header-date">アップロード日</div>
              <div className="header-status">状態</div>
              <div className="header-actions">操作</div>
            </div>
            
            {documents.map((document) => (
              <div
                key={document.id}
                className={`document-row ${selectedDocuments.has(document.id) ? 'selected' : ''}`}
                onClick={() => onDocumentClick && onDocumentClick(document)}
              >
                <div className="row-checkbox">
                  <input
                    type="checkbox"
                    checked={selectedDocuments.has(document.id)}
                    onChange={() => toggleDocumentSelection(document.id)}
                    onClick={(e) => e.stopPropagation()}
                  />
                </div>
                
                <div className="row-name">
                  <span className="document-icon">
                    {document.fileType === 'md' ? '📝' : '📄'}
                  </span>
                  <span className="document-title" title={document.fileName}>
                    {document.fileName}
                  </span>
                </div>
                
                <div className="row-size">
                  {formatFileSize(document.fileSize)}
                </div>
                
                <div className="row-date">
                  {formatDateTime(document.uploadedAt)}
                </div>
                
                <div className="row-status">
                  <span className={`status-badge ${getStatusClass(document.status)}`}>
                    {getStatusText(document.status)}
                  </span>
                </div>
                
                <div className="row-actions">
                  <button
                    className="action-button view-button"
                    onClick={(e) => {
                      e.stopPropagation();
                      onDocumentClick && onDocumentClick(document);
                    }}
                    title="文書を表示"
                  >
                    👁️
                  </button>
                  <button
                    className="action-button delete-button"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleDeleteDocument(document);
                    }}
                    title="文書を削除"
                  >
                    🗑️
                  </button>
                </div>
              </div>
            ))}
          </div>

          {/* ページネーション */}
          {totalPages > 1 && (
            <div className="pagination">
              <div className="pagination-info">
                {page} / {totalPages} ページ ({totalCount}件)
              </div>
              
              <div className="pagination-controls">
                <button
                  className="page-button"
                  onClick={() => handlePageChange(page - 1)}
                  disabled={page === 1 || loading}
                >
                  ← 前
                </button>
                
                {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                  const pageNum = Math.max(1, Math.min(totalPages - 4, page - 2)) + i;
                  return (
                    <button
                      key={pageNum}
                      className={`page-button ${pageNum === page ? 'active' : ''}`}
                      onClick={() => handlePageChange(pageNum)}
                      disabled={loading}
                    >
                      {pageNum}
                    </button>
                  );
                })}
                
                <button
                  className="page-button"
                  onClick={() => handlePageChange(page + 1)}
                  disabled={page === totalPages || loading}
                >
                  次 →
                </button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
};

export default DocumentList;
