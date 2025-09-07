import React, { useState, useEffect, useCallback } from 'react';
import './DocumentPreview.css';

interface Document {
  id: string;
  fileName: string;
  fileSize: number;
  fileType: string;
  status: string;
  uploadedAt: string;
  preview?: string;
  previewLines?: number;
}

interface DocumentPreviewProps {
  document: Document | null;
  isOpen: boolean;
  onClose: () => void;
}

interface DocumentContentResponse {
  data: {
    id: string;
    fileName: string;
    content: string;
    fileSize: number;
    fileType: string;
  };
}

const DocumentPreview: React.FC<DocumentPreviewProps> = ({
  document,
  isOpen,
  onClose,
}) => {
  const [content, setContent] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string>('');

  // 文書内容を取得
  const fetchDocumentContent = useCallback(async (documentId: string) => {
    try {
      setLoading(true);
      setError('');
      
      // 既存の文書詳細APIから文書情報（プレビューを含む）を取得
      const response = await fetch(`https://e5q2b7qvd5.execute-api.ap-northeast-1.amazonaws.com/prod/documents/${documentId}`);
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error?.message || '文書情報の取得に失敗しました');
      }

      const data = await response.json();
      const documentData = data.data;
      
      // プレビューが存在する場合はそれを使用、無い場合は代替メッセージを表示
      if (documentData.preview) {
        setContent(documentData.preview);
      } else {
        const fallbackContent = `文書ID: ${documentId}

プレビュー情報が利用できません。

文書の基本情報:
- ファイル名: ${documentData.fileName}
- ファイルサイズ: ${documentData.fileSize} bytes
- ファイルタイプ: ${documentData.fileType}
- アップロード日時: ${documentData.uploadedAt}

この文書は新しいプレビュー機能が実装される前にアップロードされたか、
プレビュー生成に失敗した可能性があります。

文書の完全な内容を確認するには、文書をダウンロードしてください。`;

        setContent(fallbackContent);
      }

    } catch (error) {
      const message = error instanceof Error ? error.message : '予期しないエラーが発生しました';
      setError(message);
      console.error('文書内容取得エラー:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  // モーダルが開かれた時に文書内容を取得
  useEffect(() => {
    if (isOpen && document) {
      fetchDocumentContent(document.id);
    } else {
      // モーダルが閉じられた時に状態をリセット
      setContent('');
      setError('');
      setLoading(false);
    }
  }, [isOpen, document, fetchDocumentContent]);

  // ESCキーでモーダルを閉じる
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && isOpen) {
        onClose();
      }
    };

    if (isOpen && typeof window !== 'undefined' && window.document) {
      window.document.addEventListener('keydown', handleKeyDown);
      return () => {
        window.document.removeEventListener('keydown', handleKeyDown);
      };
    }
  }, [isOpen, onClose]);

  // モーダル背景クリックで閉じる
  const handleBackdropClick = useCallback((event: React.MouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget) {
      onClose();
    }
  }, [onClose]);

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

  // コンテンツをコピー
  const handleCopyContent = useCallback(async () => {
    if (!content) return;

    try {
      await navigator.clipboard.writeText(content);
      // 一時的な成功表示（簡易実装）
      if (typeof window !== 'undefined' && window.document) {
        const originalText = window.document.querySelector('.copy-button')?.textContent;
        const copyButton = window.document.querySelector('.copy-button') as HTMLButtonElement;
        if (copyButton && originalText) {
          copyButton.textContent = '✅ コピー済み';
          setTimeout(() => {
            copyButton.textContent = originalText;
          }, 2000);
        }
      }
    } catch (error) {
      console.error('コピーに失敗しました:', error);
    }
  }, [content]);

  if (!isOpen || !document) {
    return null;
  }

  return (
    <div className="document-preview-overlay" onClick={handleBackdropClick}>
      <div className="document-preview-modal">
        {/* ヘッダー */}
        <div className="modal-header">
          <div className="modal-title">
            <span className="document-icon">
              {document.fileType === 'md' ? '📝' : '📄'}
            </span>
            <div className="title-info">
              <h2>{document.fileName}</h2>
              <div className="document-meta">
                {formatFileSize(document.fileSize)} • {formatDateTime(document.uploadedAt)}
                {document.previewLines && (
                  <> • プレビュー: 先頭{document.previewLines}行</>
                )}
              </div>
            </div>
          </div>
          <div className="modal-actions">
            {content && (
              <button className="copy-button" onClick={handleCopyContent}>
                📋 コピー
              </button>
            )}
            <button className="close-button" onClick={onClose}>
              ✕
            </button>
          </div>
        </div>

        {/* コンテンツエリア */}
        <div className="modal-content">
          {loading && (
            <div className="loading-container">
              <div className="loading-spinner" />
              <p>文書内容を読み込み中...</p>
            </div>
          )}

          {error && (
            <div className="error-container">
              <div className="error-message">エラーが発生しました</div>
              <p>{error}</p>
              <button 
                className="retry-button" 
                onClick={() => fetchDocumentContent(document.id)}
              >
                再試行
              </button>
            </div>
          )}

          {!loading && !error && content && (
            <div className="document-content">
              <pre className="content-text">{content}</pre>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default DocumentPreview;