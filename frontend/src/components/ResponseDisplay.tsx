import React, { useState, useCallback } from 'react';
import './ResponseDisplay.css';

// 型定義（QueryInputと共通）
interface Source {
  documentId: string;
  fileName: string;
  excerpt: string;
  confidence: number;
}

interface RAGResponse {
  id: string;
  answer: string;
  sources: Source[];
  processingTimeMs: number;
  modelUsed: string;
  tokensUsed: number;
  createdAt: string;
}

interface QueryResponse {
  id: string;
  sessionId: string;
  question: string;
  status: string;
  processingTimeMs: number;
  createdAt: string;
  updatedAt: string;
}

interface QueryWithResponse {
  query: QueryResponse;
  response: RAGResponse;
}

interface ResponseDisplayProps {
  queryWithResponse: QueryWithResponse | null;
  onCopyResponse?: (text: string) => void;
  onSourceClick?: (source: Source) => void;
}

const ResponseDisplay: React.FC<ResponseDisplayProps> = ({
  queryWithResponse,
  onCopyResponse,
  onSourceClick,
}) => {
  const [copiedText, setCopiedText] = useState<string>('');
  const [expandedSources, setExpandedSources] = useState<Set<string>>(new Set());

  // テキストコピー機能
  const handleCopyText = useCallback(async (text: string, label: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedText(label);
      
      if (onCopyResponse) {
        onCopyResponse(text);
      }
      
      // 3秒後にコピー状態をリセット
      setTimeout(() => setCopiedText(''), 3000);
    } catch (error) {
      console.error('Failed to copy text:', error);
    }
  }, [onCopyResponse]);

  // 情報源の展開/折りたたみ
  const toggleSourceExpansion = useCallback((sourceId: string) => {
    setExpandedSources(prev => {
      const newSet = new Set(prev);
      if (newSet.has(sourceId)) {
        newSet.delete(sourceId);
      } else {
        newSet.add(sourceId);
      }
      return newSet;
    });
  }, []);

  // 信頼度表示
  const getConfidenceLevel = (confidence: number): string => {
    if (confidence >= 0.8) return 'high';
    if (confidence >= 0.6) return 'medium';
    return 'low';
  };

  const getConfidenceText = (confidence: number): string => {
    if (confidence >= 0.8) return '高い';
    if (confidence >= 0.6) return '中程度';
    return '低い';
  };

  // 処理時間フォーマット
  const formatProcessingTime = (timeMs: number): string => {
    if (timeMs < 1000) return `${timeMs}ms`;
    return `${(timeMs / 1000).toFixed(1)}秒`;
  };

  // 日時フォーマット
  const formatDateTime = (isoString: string): string => {
    return new Date(isoString).toLocaleString('ja-JP', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  };

  if (!queryWithResponse) {
    return null;
  }

  const { query, response } = queryWithResponse;

  return (
    <div className="response-display">
      {/* 質問の表示 */}
      <div className="question-section">
        <div className="section-header">
          <h4>📝 質問</h4>
          <button
            className={`copy-button ${copiedText === 'question' ? 'copied' : ''}`}
            onClick={() => handleCopyText(query.question, 'question')}
            title="質問をコピー"
          >
            {copiedText === 'question' ? '✓' : '📋'}
          </button>
        </div>
        <div className="question-content">
          {query.question}
        </div>
        <div className="question-meta">
          {formatDateTime(query.createdAt)}
        </div>
      </div>

      {/* 回答の表示 */}
      <div className="answer-section">
        <div className="section-header">
          <h4>🤖 回答</h4>
          <button
            className={`copy-button ${copiedText === 'answer' ? 'copied' : ''}`}
            onClick={() => handleCopyText(response.answer, 'answer')}
            title="回答をコピー"
          >
            {copiedText === 'answer' ? '✓' : '📋'}
          </button>
        </div>
        <div className="answer-content">
          {response.answer.split('\n').map((paragraph, index) => (
            <p key={index}>{paragraph}</p>
          ))}
        </div>
      </div>

      {/* 情報源の表示 */}
      {response.sources.length > 0 && (
        <div className="sources-section">
          <div className="section-header">
            <h4>📚 参考情報源 ({response.sources.length}件)</h4>
          </div>
          <div className="sources-list">
            {response.sources.map((source, index) => {
              const sourceKey = `${source.documentId}-${index}`;
              const isExpanded = expandedSources.has(sourceKey);
              const confidenceLevel = getConfidenceLevel(source.confidence);
              
              return (
                <div key={sourceKey} className="source-item">
                  <div 
                    className="source-header"
                    onClick={() => toggleSourceExpansion(sourceKey)}
                  >
                    <div className="source-info">
                      <span className="source-filename">{source.fileName}</span>
                      <div className="source-confidence">
                        <span className={`confidence-badge ${confidenceLevel}`}>
                          信頼度: {getConfidenceText(source.confidence)} ({Math.round(source.confidence * 100)}%)
                        </span>
                      </div>
                    </div>
                    <button className="expand-button">
                      {isExpanded ? '▼' : '▶'}
                    </button>
                  </div>
                  
                  {isExpanded && (
                    <div className="source-excerpt">
                      <div className="excerpt-content">
                        {source.excerpt}
                      </div>
                      <div className="source-actions">
                        <button
                          className={`copy-button small ${copiedText === sourceKey ? 'copied' : ''}`}
                          onClick={() => handleCopyText(source.excerpt, sourceKey)}
                        >
                          {copiedText === sourceKey ? '✓' : '📋'} 抜粋をコピー
                        </button>
                        {onSourceClick && (
                          <button
                            className="view-document-button"
                            onClick={() => onSourceClick(source)}
                          >
                            📄 文書を表示
                          </button>
                        )}
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* メタデータの表示 */}
      <div className="metadata-section">
        <div className="section-header">
          <h4>ℹ️ 処理情報</h4>
        </div>
        <div className="metadata-grid">
          <div className="metadata-item">
            <span className="metadata-label">処理時間</span>
            <span className="metadata-value">
              {formatProcessingTime(response.processingTimeMs)}
            </span>
          </div>
          <div className="metadata-item">
            <span className="metadata-label">使用モデル</span>
            <span className="metadata-value">{response.modelUsed}</span>
          </div>
          <div className="metadata-item">
            <span className="metadata-label">使用トークン</span>
            <span className="metadata-value">{response.tokensUsed.toLocaleString()}</span>
          </div>
          <div className="metadata-item">
            <span className="metadata-label">回答ID</span>
            <span className="metadata-value" title={response.id}>
              {response.id.substring(0, 8)}...
            </span>
          </div>
        </div>
      </div>

      {/* 全体コピー機能 */}
      <div className="response-actions">
        <button
          className={`copy-all-button ${copiedText === 'all' ? 'copied' : ''}`}
          onClick={() => {
            const fullText = `【質問】\n${query.question}\n\n【回答】\n${response.answer}\n\n【処理時間】${formatProcessingTime(response.processingTimeMs)}`;
            handleCopyText(fullText, 'all');
          }}
        >
          {copiedText === 'all' ? '✓ コピー完了' : '📋 質問と回答をすべてコピー'}
        </button>
      </div>
    </div>
  );
};

export default ResponseDisplay;