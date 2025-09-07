import React, { useState, useCallback, useEffect, useRef } from 'react';
import './App.css';

// コンポーネントのインポート
import DocumentUpload from './components/DocumentUpload';
import DocumentList from './components/DocumentList';
import DocumentPreview from './components/DocumentPreview';
import QueryInput from './components/QueryInput';
import ResponseDisplay from './components/ResponseDisplay';

// 型定義（コンポーネント間で共有）
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

interface Document {
  id: string;
  fileName: string;
  fileSize: number;
  fileType: string;
  status: string;
  uploadedAt: string;
  lastModified?: string;
  preview?: string;
  previewLines?: number;
}

// アプリケーションの状態タイプ
type AppView = 'query' | 'upload' | 'documents';

const App: React.FC = () => {
  // セッション管理
  const [sessionId] = useState<string>(() => {
    // 既存セッションをクリアして新しいセッションを生成
    const newSessionId = `session_${Date.now()}_${Math.random().toString(36).substring(2, 15)}`;
    localStorage.setItem('ragSessionId', newSessionId);
    console.log('新しいセッションID:', newSessionId);
    return newSessionId;
  });

  // UI状態管理
  const [currentView, setCurrentView] = useState<AppView>('query');
  const [chatHistory, setChatHistory] = useState<QueryWithResponse[]>([]);
  const [documentListRefresh, setDocumentListRefresh] = useState<number>(0);
  const [appStatus, setAppStatus] = useState<string>('');
  const [isLoading, setIsLoading] = useState<boolean>(false);
  
  // 文書プレビューモーダルの状態管理
  const [previewDocument, setPreviewDocument] = useState<Document | null>(null);
  const [isPreviewOpen, setIsPreviewOpen] = useState<boolean>(false);
  
  // コピーボタンの状態管理
  const [copiedStates, setCopiedStates] = useState<{[key: string]: boolean}>({});
  
  // チャット履歴の自動スクロール用ref
  const chatHistoryRef = useRef<HTMLDivElement>(null);

  // 自動スクロール機能
  const scrollToBottom = useCallback(() => {
    if (chatHistoryRef.current) {
      chatHistoryRef.current.scrollTop = chatHistoryRef.current.scrollHeight;
    }
  }, []);

  // チャット履歴が更新されたら自動スクロール
  useEffect(() => {
    scrollToBottom();
  }, [chatHistory, scrollToBottom]);

  // ヘルスチェック
  useEffect(() => {
    const checkHealth = async () => {
      try {
        const response = await fetch('https://e5q2b7qvd5.execute-api.ap-northeast-1.amazonaws.com/prod/health');
        if (response.ok) {
          const data = await response.json();
          setAppStatus(data.data?.message || 'サービス正常');
        } else {
          setAppStatus('サービス接続エラー');
        }
      } catch (error) {
        console.error('ヘルスチェックエラー:', error);
        setAppStatus('サービス接続エラー');
      }
    };

    checkHealth();
    
    // 定期的なヘルスチェック（5分間隔）
    const interval = setInterval(checkHealth, 5 * 60 * 1000);
    return () => clearInterval(interval);
  }, []);

  // 文書アップロード完了ハンドラー
  const handleUploadComplete = useCallback((document: any) => {
    console.log('アップロード完了:', document);
    setAppStatus(`「${document.fileName}」のアップロードが完了しました`);
    
    // 文書リストを更新
    setDocumentListRefresh(prev => prev + 1);
    
    // ステータスメッセージを3秒後にクリア
    setTimeout(() => setAppStatus(''), 3000);
  }, []);

  // 文書アップロードエラーハンドラー
  const handleUploadError = useCallback((error: string) => {
    console.error('アップロードエラー:', error);
    setAppStatus(`アップロードエラー: ${error}`);
  }, []);

  // 質問送信完了ハンドラー
  const handleQuerySubmit = useCallback((result: QueryWithResponse) => {
    console.log('クエリ結果:', result);
    setChatHistory(prev => [...prev, result]);
    setCurrentView('query');
    setAppStatus('質問への回答が生成されました');
    
    // 少し遅延してスクロール（DOMの更新を待つ）
    setTimeout(() => {
      scrollToBottom();
    }, 100);
    
    // ステータスメッセージを3秒後にクリア
    setTimeout(() => setAppStatus(''), 3000);
  }, [scrollToBottom]);

  // 質問送信エラーハンドラー
  const handleQueryError = useCallback((error: string) => {
    console.error('クエリエラー:', error);
    setAppStatus(`質問処理エラー: ${error}`);
  }, []);

  // レスポンスコピーハンドラー
  const handleCopyResponse = useCallback(async (text: string, messageId: string) => {
    try {
      await navigator.clipboard.writeText(text);
      console.log('テキストコピー完了:', text.length, '文字');
      setAppStatus('テキストをクリップボードにコピーしました');
      
      // コピー成功のリアクション
      setCopiedStates(prev => ({ ...prev, [messageId]: true }));
      
      // 1.5秒後にリアクションをリセット
      setTimeout(() => {
        setCopiedStates(prev => ({ ...prev, [messageId]: false }));
      }, 1500);
      
    } catch (error) {
      console.error('コピーに失敗しました:', error);
      // フォールバック: 従来のコピー方法を試す
      try {
        const textArea = document.createElement('textarea');
        textArea.value = text;
        textArea.style.position = 'fixed';
        textArea.style.left = '-999999px';
        textArea.style.top = '-999999px';
        document.body.appendChild(textArea);
        textArea.focus();
        textArea.select();
        document.execCommand('copy');
        document.body.removeChild(textArea);
        console.log('フォールバック方式でコピー完了:', text.length, '文字');
        setAppStatus('テキストをクリップボードにコピーしました');
        
        // コピー成功のリアクション
        setCopiedStates(prev => ({ ...prev, [messageId]: true }));
        
        // 1.5秒後にリアクションをリセット
        setTimeout(() => {
          setCopiedStates(prev => ({ ...prev, [messageId]: false }));
        }, 1500);
        
      } catch (fallbackError) {
        console.error('フォールバックコピーも失敗しました:', fallbackError);
        setAppStatus('コピーに失敗しました。ブラウザがクリップボードアクセスを許可していない可能性があります。');
      }
    }
    
    // ステータスメッセージを2秒後にクリア
    setTimeout(() => setAppStatus(''), 2000);
  }, []);

  // 文書クリックハンドラー（プレビュー表示）
  const handleDocumentClick = useCallback((document: Document) => {
    console.log('文書選択:', document);
    setPreviewDocument(document);
    setIsPreviewOpen(true);
    setAppStatus(`「${document.fileName}」のプレビューを表示します`);
    
    setTimeout(() => setAppStatus(''), 2000);
  }, []);

  // 文書プレビューを閉じる
  const handlePreviewClose = useCallback(() => {
    setIsPreviewOpen(false);
    setPreviewDocument(null);
  }, []);

  // 文書削除ハンドラー
  const handleDocumentDelete = useCallback((document: Document) => {
    console.log('文書削除:', document);
    setAppStatus(`「${document.fileName}」を削除しました`);
    
    // ステータスメッセージを3秒後にクリア
    setTimeout(() => setAppStatus(''), 3000);
  }, []);

  // ソースクリックハンドラー
  const handleSourceClick = useCallback((source: Source) => {
    console.log('ソース参照:', source);
    setAppStatus(`「${source.fileName}」の詳細を表示中...`);
    
    // 将来的にはソースドキュメントのハイライト表示機能を追加可能
    setTimeout(() => setAppStatus(''), 2000);
  }, []);

  // ナビゲーション変更ハンドラー
  const handleViewChange = useCallback((view: AppView) => {
    setCurrentView(view);
    
    // ビュー切り替え時にステータスをクリア
    setAppStatus('');
  }, []);

  // チャット履歴をクリア
  const handleClearChat = useCallback(() => {
    if (window.confirm('会話履歴をすべてクリアしますか？')) {
      setChatHistory([]);
      setAppStatus('会話履歴をクリアしました');
      setTimeout(() => setAppStatus(''), 2000);
    }
  }, []);

  // チャット履歴全体をコピー
  const handleCopyAllChat = useCallback(async () => {
    if (chatHistory.length === 0) return;
    
    const chatText = chatHistory.map((item, index) => {
      const { query, response } = item;
      return `=== 質問 ${index + 1} ===\n${query.question}\n\n=== 回答 ${index + 1} ===\n${response.answer}\n\n処理時間: ${response.processingTimeMs < 1000 ? response.processingTimeMs + 'ms' : (response.processingTimeMs / 1000).toFixed(1) + '秒'}\n使用モデル: ${response.modelUsed}\n\n`;
    }).join('');

    try {
      await navigator.clipboard.writeText(chatText);
      setAppStatus('会話履歴全体をクリップボードにコピーしました');
      setTimeout(() => setAppStatus(''), 3000);
    } catch (error) {
      console.error('Failed to copy chat history:', error);
      setAppStatus('コピーに失敗しました');
      setTimeout(() => setAppStatus(''), 3000);
    }
  }, [chatHistory]);

  return (
    <div className="app">
      {/* メインコンテンツ */}
      <main className="app-main">
        <div className="content-wrapper">
          {currentView === 'upload' && (
            <div className="view-section full-screen-view">
              <div className="full-screen-container">
                {/* アップロード用トップバー */}
                <div className="chat-topbar">
                  <div className="topbar-left">
                    <h1>🤖 AWS RAG Knowledge Base</h1>
                  </div>
                  <div className="topbar-right">
                    <button
                      className="topbar-button"
                      onClick={() => handleViewChange('query')}
                      title="チャット"
                    >
                      💬
                    </button>
                    <span className="current-page-indicator">📤 文書アップロード</span>
                    <button
                      className="topbar-button"
                      onClick={() => handleViewChange('documents')}
                      title="文書一覧"
                    >
                      📂
                    </button>
                  </div>
                </div>
                
                <div className="full-screen-content">
                  <DocumentUpload
                    onUploadComplete={handleUploadComplete}
                    onUploadError={handleUploadError}
                  />
                </div>
              </div>
            </div>
          )}

          {currentView === 'query' && (
            <div className="view-section chat-view">
              <div className="chat-container">
                {/* チャット用トップバー */}
                <div className="chat-topbar">
                  <div className="topbar-left">
                    <h1>🤖 AWS RAG Knowledge Base</h1>
                  </div>
                  <div className="topbar-right">
                    <span className="current-page-indicator">💬 チャット</span>
                    <button
                      className="topbar-button"
                      onClick={() => handleViewChange('upload')}
                      title="文書アップロード"
                    >
                      📤
                    </button>
                    <button
                      className="topbar-button"
                      onClick={() => handleViewChange('documents')}
                      title="文書一覧"
                    >
                      📂
                    </button>
                  </div>
                </div>
                
                <div className="chat-history" ref={chatHistoryRef}>
                  {chatHistory.map((item, index) => (
                    <div key={`${item.response.id}-${index}`}>
                      {/* ユーザーメッセージ */}
                      <div className="chat-message user-message">
                        <div className="message-wrapper">
                          <div className="message-avatar-section">
                            <div className="message-avatar user-avatar">👤</div>
                            <button 
                              className={`message-copy-btn ${copiedStates[`user-${index}`] ? 'copied' : ''}`}
                              onClick={() => handleCopyResponse(item.query.question, `user-${index}`)}
                              title="質問をコピー"
                            >
                              {copiedStates[`user-${index}`] ? '✅' : '📋'}
                            </button>
                          </div>
                          <div className="message-content">
                            <div className="message-text">{item.query.question}</div>
                            <div className="message-time">{new Date(item.query.createdAt).toLocaleString('ja-JP', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}</div>
                          </div>
                        </div>
                      </div>
                      
                      {/* AIメッセージ */}
                      <div className="chat-message ai-message">
                        <div className="message-wrapper">
                          <div className="message-avatar-section">
                            <div className="message-avatar ai-avatar">🤖</div>
                            <button 
                              className={`message-copy-btn ${copiedStates[`ai-${index}`] ? 'copied' : ''}`}
                              onClick={() => handleCopyResponse(item.response.answer, `ai-${index}`)}
                              title="回答をコピー"
                            >
                              {copiedStates[`ai-${index}`] ? '✅' : '📋'}
                            </button>
                          </div>
                          <div className="message-content">
                            <div className="message-text">{item.response.answer}</div>
                            <div className="message-meta">
                              <span className="message-time">{new Date(item.response.createdAt).toLocaleString('ja-JP', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}</span>
                              <span className="message-info">• {item.response.processingTimeMs < 1000 ? item.response.processingTimeMs + 'ms' : (item.response.processingTimeMs / 1000).toFixed(1) + '秒'}</span>
                              {item.response.sources.length > 0 && (
                                <span className="message-sources">• {item.response.sources.length}件の参考資料</span>
                              )}
                            </div>
                            {/* 情報源の簡易表示 */}
                            {item.response.sources.length > 0 && (
                              <div className="message-sources-list">
                                {item.response.sources.slice(0, 3).map((source, idx) => (
                                  <div key={idx} className="source-badge" title={source.excerpt}>
                                    📄 {source.fileName}
                                    <span className="source-confidence">
                                      ({Math.round(source.confidence * 100)}%)
                                    </span>
                                  </div>
                                ))}
                                {item.response.sources.length > 3 && (
                                  <div className="source-badge more-sources">
                                    +{item.response.sources.length - 3}件
                                  </div>
                                )}
                              </div>
                            )}
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                  
                  {chatHistory.length === 0 && (
                    <div className="chat-empty-state">
                      <div className="empty-icon">🤖</div>
                      <h3>AI アシスタントへようこそ</h3>
                      <p>Knowledge Base の情報をもとに、あなたの質問にお答えします。<br/>何でもお気軽にお聞きください。</p>
                    </div>
                  )}
                </div>
                
                <div className="chat-input-section">
                  <QueryInput
                    sessionId={sessionId}
                    onQuerySubmit={handleQuerySubmit}
                    onQueryError={handleQueryError}
                    disabled={isLoading}
                  />
                  
                  {/* チャット履歴管理ボタン */}
                  {chatHistory.length > 0 && (
                    <div className="chat-controls">
                      <button
                        className="chat-control-button clear-button"
                        onClick={handleClearChat}
                        title="会話履歴をクリア"
                      >
                        🗑️ 会話履歴をクリア
                      </button>
                      <button
                        className="chat-control-button export-button"
                        onClick={handleCopyAllChat}
                        title="会話履歴をコピー"
                      >
                        📋 会話履歴をコピー
                      </button>
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}

          {currentView === 'documents' && (
            <div className="view-section full-screen-view">
              <div className="full-screen-container">
                {/* 文書一覧用トップバー */}
                <div className="chat-topbar">
                  <div className="topbar-left">
                    <h1>🤖 AWS RAG Knowledge Base</h1>
                  </div>
                  <div className="topbar-right">
                    <button
                      className="topbar-button"
                      onClick={() => handleViewChange('query')}
                      title="チャット"
                    >
                      💬
                    </button>
                    <button
                      className="topbar-button"
                      onClick={() => handleViewChange('upload')}
                      title="文書アップロード"
                    >
                      📤
                    </button>
                    <span className="current-page-indicator">📂 文書一覧</span>
                  </div>
                </div>
                
                <div className="full-screen-content">
                  <DocumentList
                    refreshTrigger={documentListRefresh}
                    onDocumentClick={handleDocumentClick}
                    onDocumentDelete={handleDocumentDelete}
                  />
                </div>
              </div>
            </div>
          )}
        </div>
      </main>

      {/* フッター */}
      <footer className="app-footer">
        <div className="footer-content">
          <div className="footer-info">
            <span>AWS Bedrock Knowledge Base RAG System</span>
            <span className="separator">•</span>
            <span>Go + TypeScript + React</span>
          </div>
          <div className="footer-links">
            <button
              className="footer-link"
              onClick={() => window.open('https://e5q2b7qvd5.execute-api.ap-northeast-1.amazonaws.com/prod/health', '_blank')}
            >
              API Status
            </button>
          </div>
        </div>
      </footer>

      {/* 文書プレビューモーダル */}
      <DocumentPreview
        document={previewDocument}
        isOpen={isPreviewOpen}
        onClose={handlePreviewClose}
      />

      {/* ローディングオーバーレイ（将来的な拡張用） */}
      {isLoading && (
        <div className="loading-overlay">
          <div className="loading-content">
            <div className="loading-spinner" />
            <p>処理中...</p>
          </div>
        </div>
      )}
    </div>
  );
};

export default App;