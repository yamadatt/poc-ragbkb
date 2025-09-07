import React, { useState, useCallback } from 'react';
import './QueryInput.css';

// RAGクエリレスポンス
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

interface QueryInputProps {
  sessionId: string;
  onQuerySubmit?: (result: QueryWithResponse) => void;
  onQueryError?: (error: string) => void;
  disabled?: boolean;
}

const QueryInput: React.FC<QueryInputProps> = ({
  sessionId,
  onQuerySubmit,
  onQueryError,
  disabled = false,
}) => {
  const [question, setQuestion] = useState<string>('');
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [errorMessage, setErrorMessage] = useState<string>('');

  // 質問送信ハンドラー
  const handleSubmit = useCallback(async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    
    const trimmedQuestion = question.trim();
    if (!trimmedQuestion) {
      setErrorMessage('質問を入力してください');
      return;
    }

    if (trimmedQuestion.length > 1000) {
      setErrorMessage('質問は1000文字以内で入力してください');
      return;
    }

    try {
      setIsLoading(true);
      setErrorMessage('');

      console.log('送信データ:', { question: trimmedQuestion, sessionId });

      const response = await fetch('https://e5q2b7qvd5.execute-api.ap-northeast-1.amazonaws.com/prod/queries', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          question: trimmedQuestion,
          sessionId,
        }),
      });

      console.log('レスポンスステータス:', response.status);

      if (!response.ok) {
        const errorText = await response.text();
        console.log('エラーレスポンス:', errorText);
        
        let errorData;
        try {
          errorData = JSON.parse(errorText);
        } catch {
          errorData = { error: { message: errorText } };
        }
        
        if (response.status === 404) {
          throw new Error('関連する情報が見つかりませんでした。別の質問をお試しください。');
        }
        
        if (response.status === 400 && errorData.error?.message?.includes('セッション')) {
          throw new Error('セッションが無効です。ページを再読み込みしてください。');
        }
        
        throw new Error(errorData.error?.message || `API エラー (${response.status}): ${errorText}`);
      }

      const data = await response.json();
      const result: QueryWithResponse = data.data;

      // 成功コールバック
      if (onQuerySubmit) {
        onQuerySubmit(result);
      }

      // 質問をクリア
      setQuestion('');

    } catch (error) {
      const message = error instanceof Error ? error.message : '予期しないエラーが発生しました';
      setErrorMessage(message);
      
      if (onQueryError) {
        onQueryError(message);
      }
    } finally {
      setIsLoading(false);
    }
  }, [question, sessionId, onQuerySubmit, onQueryError]);

  // 質問入力ハンドラー
  const handleQuestionChange = useCallback((event: React.ChangeEvent<HTMLTextAreaElement>) => {
    const value = event.target.value;
    setQuestion(value);
    
    // エラーメッセージをクリア
    if (errorMessage) {
      setErrorMessage('');
    }
  }, [errorMessage]);

  // キーボードショートカット（Ctrl+Enter / Cmd+Enter で送信）
  const handleKeyDown = useCallback((event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
      event.preventDefault();
      const form = event.currentTarget.form;
      if (form) {
        form.requestSubmit();
      }
    }
  }, []);

  // 文字数カウント
  const remainingChars = 1000 - question.length;
  const isOverLimit = remainingChars < 0;

  return (
    <div className="query-input">
      <div className="query-header">
        <h3>質問を入力</h3>
        <p>Knowledge Baseに登録された文書から回答を生成します</p>
      </div>

      <form onSubmit={handleSubmit} className="query-form">
        <div className="input-section">
          <textarea
            className={`question-input ${isOverLimit ? 'over-limit' : ''}`}
            placeholder="例: AWS Bedrock Knowledge Baseの使い方を教えてください"
            value={question}
            onChange={handleQuestionChange}
            onKeyDown={handleKeyDown}
            disabled={disabled || isLoading}
            rows={4}
            maxLength={1200} // 少し余裕を持たせる
          />
          
          <div className="input-footer">
            <div className={`char-count ${isOverLimit ? 'over-limit' : ''}`}>
              {question.length} / 1000文字
              {isOverLimit && (
                <span className="over-limit-text">
                  （{Math.abs(remainingChars)}文字超過）
                </span>
              )}
            </div>
            
            <div className="input-hint">
              Ctrl+Enter または Cmd+Enter で送信
            </div>
          </div>
        </div>

        {/* エラーメッセージ */}
        {errorMessage && (
          <div className="error-message">
            ❌ {errorMessage}
          </div>
        )}

        {/* 送信ボタン */}
        <div className="submit-section">
          <button
            type="submit"
            className="submit-button"
            disabled={disabled || isLoading || !question.trim() || isOverLimit}
          >
            {isLoading ? (
              <>
                <div className="loading-spinner" />
                回答生成中...
              </>
            ) : (
              '質問を送信'
            )}
          </button>
          
          {question.trim() && !isLoading && (
            <button
              type="button"
              className="clear-button"
              onClick={() => {
                setQuestion('');
                setErrorMessage('');
              }}
            >
              クリア
            </button>
          )}
        </div>
      </form>

      {/* セッション情報 */}
      <div className="session-info">
        <small>セッションID: {sessionId}</small>
      </div>

    </div>
  );
};

export default QueryInput;