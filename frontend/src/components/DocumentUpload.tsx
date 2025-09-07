import React, { useState, useCallback, useEffect, useRef } from 'react';
import './DocumentUpload.css';

// 文書アップロードの状態
type UploadStatus = 'idle' | 'uploading' | 'success' | 'error';

// アップロードセッション情報
interface UploadSession {
  id: string;
  fileName: string;
  fileSize: number;
  fileType: string;
  uploadUrl: string;
  expiresAt: string;
}

// 完了レスポンス
interface CompleteUploadResponse {
  id: string;
  fileName: string;
  fileSize: number;
  fileType: string;
  status: string;
}

interface DocumentUploadProps {
  onUploadComplete?: (document: CompleteUploadResponse) => void;
  onUploadError?: (error: string) => void;
}

const DocumentUpload: React.FC<DocumentUploadProps> = ({
  onUploadComplete,
  onUploadError,
}) => {
  const [file, setFile] = useState<File | null>(null);
  const [uploadStatus, setUploadStatus] = useState<UploadStatus>('idle');
  const [progress, setProgress] = useState<number>(0);
  const [errorMessage, setErrorMessage] = useState<string>('');
  const fileInputRef = useRef<HTMLInputElement>(null);


  // ファイル選択ハンドラー
  const handleFileSelect = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    try {
      console.log('handleFileSelect called', event);
      const selectedFile = event.target.files?.[0];
      console.log('Selected file:', selectedFile);
      
      if (!selectedFile) {
        console.log('No file selected');
        return;
      }

      // ファイルサイズチェック（50MB）
      if (selectedFile.size > 50 * 1024 * 1024) {
        console.log('File size exceeded limit:', selectedFile.size);
        setErrorMessage('ファイルサイズが制限を超えています（最大50MB）');
        setFile(null);
        return;
      }

      // ファイルタイプチェック
      const allowedTypes = ['text/plain', 'text/markdown'];
      const allowedExtensions = ['.txt', '.md'];
      const fileExtension = selectedFile.name.toLowerCase().slice(selectedFile.name.lastIndexOf('.'));
      console.log('File type:', selectedFile.type, 'Extension:', fileExtension);
      
      if (!allowedTypes.includes(selectedFile.type) && !allowedExtensions.includes(fileExtension)) {
        console.log('Unsupported file type');
        setErrorMessage('サポートされていないファイルタイプです（.txt, .mdのみ）');
        setFile(null);
        return;
      }

      console.log('File validation passed, setting file');
      setFile(selectedFile);
      setErrorMessage('');
      setUploadStatus('idle');
      setProgress(0);
    } catch (error) {
      console.error('Error in handleFileSelect:', error);
      setErrorMessage('ファイル選択処理中にエラーが発生しました');
    }
  }, []);

  // アップロード実行
  const handleUpload = useCallback(async () => {
    if (!file) return;

    try {
      setUploadStatus('uploading');
      setProgress(0);
      setErrorMessage('');

      console.log('アップロード開始:', {
        fileName: file.name,
        fileSize: file.size,
        fileType: file.name.endsWith('.md') ? 'md' : 'txt'
      });

      // Step 1: アップロードセッション開始
      setProgress(20);
      console.log('Step 1: セッション作成リクエスト');
      console.log('API URL:', 'https://e5q2b7qvd5.execute-api.ap-northeast-1.amazonaws.com/prod/documents');
      
      const sessionResponse = await fetch('https://e5q2b7qvd5.execute-api.ap-northeast-1.amazonaws.com/prod/documents', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          fileName: file.name,
          fileSize: file.size,
          fileType: file.name.endsWith('.md') ? 'md' : 'txt',
        }),
      });

      console.log('セッション作成レスポンス:', sessionResponse.status, sessionResponse.statusText);

      if (!sessionResponse.ok) {
        const errorText = await sessionResponse.text();
        console.log('セッション作成エラー:', errorText);
        
        let errorData;
        try {
          errorData = JSON.parse(errorText);
        } catch {
          errorData = { error: { message: errorText } };
        }
        
        throw new Error(errorData.error?.message || `セッション作成エラー (${sessionResponse.status}): ${errorText}`);
      }

      const sessionData = await sessionResponse.json();
      const session: UploadSession = sessionData.data;

      // Step 2: S3に直接アップロード
      setProgress(40);
      const s3Response = await fetch(session.uploadUrl, {
        method: 'PUT',
        body: file,
        headers: {
          'Content-Type': 'application/octet-stream',
        },
      });

      if (!s3Response.ok) {
        throw new Error('ファイルのアップロードに失敗しました');
      }

      // Step 3: アップロード完了通知
      setProgress(80);
      const completeResponse = await fetch(`https://e5q2b7qvd5.execute-api.ap-northeast-1.amazonaws.com/prod/documents/${session.id}/complete-upload`, {
        method: 'POST',
      });

      if (!completeResponse.ok) {
        const errorData = await completeResponse.json();
        throw new Error(errorData.error?.message || 'アップロード完了処理に失敗しました');
      }

      const completeData = await completeResponse.json();
      const document: CompleteUploadResponse = completeData.data;

      setProgress(100);
      setUploadStatus('success');
      
      // 成功コールバック
      if (onUploadComplete) {
        onUploadComplete(document);
      }

      // 成功後の初期化
      setTimeout(() => {
        setFile(null);
        setProgress(0);
        setUploadStatus('idle');
        // ファイル入力をリセット
        if (fileInputRef.current) {
          fileInputRef.current.value = '';
        }
      }, 2000);

    } catch (error) {
      console.error('アップロードエラー詳細:', error);
      
      let message: string;
      if (error instanceof TypeError && error.message.includes('fetch')) {
        message = 'ネットワークエラーが発生しました。インターネット接続を確認してください。';
      } else if (error instanceof Error) {
        message = error.message;
      } else {
        message = '予期しないエラーが発生しました';
      }
      
      console.log('最終エラーメッセージ:', message);
      setErrorMessage(message);
      setUploadStatus('error');
      setProgress(0);
      
      if (onUploadError) {
        onUploadError(message);
      }
    }
  }, [file, onUploadComplete, onUploadError]);

  // ドラッグ&ドロップハンドラー
  const handleDragOver = useCallback((event: React.DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.stopPropagation();
  }, []);

  const handleDrop = useCallback((event: React.DragEvent<HTMLDivElement>) => {
    try {
      event.preventDefault();
      event.stopPropagation();
      
      console.log('Drop event triggered', event);
      console.log('dataTransfer:', event.dataTransfer);
      console.log('dataTransfer.files:', event.dataTransfer.files);
      console.log('dataTransfer.files.length:', event.dataTransfer.files?.length);
      
      const files = event.dataTransfer.files;
      if (files && files.length > 0) {
        const droppedFile = files[0];
        console.log('Dropped file:', droppedFile);
        
        // ファイル選択と同じロジックを適用
        const mockEvent = {
          target: { files: Array.from(files) }
        } as React.ChangeEvent<HTMLInputElement>;
        console.log('Calling handleFileSelect with mock event');
        handleFileSelect(mockEvent);
      } else {
        console.log('No files in drop event or files is null/undefined');
        console.log('dataTransfer items:', event.dataTransfer.items);
        console.log('dataTransfer types:', event.dataTransfer.types);
      }
    } catch (error) {
      console.error('Error in handleDrop:', error);
      setErrorMessage('ファイルのドロップ処理中にエラーが発生しました');
    }
  }, [handleFileSelect]);

  return (
    <div className="document-upload">
      <div className="upload-header">
        <p>テキストファイル（.txt, .md）をアップロードしてKnowledge Baseに追加します</p>
      </div>

      {/* ファイル選択エリア */}
      <div
        className={`upload-area ${uploadStatus === 'uploading' ? 'uploading' : ''}`}
        onDragOver={handleDragOver}
        onDrop={handleDrop}
      >
        <input
          type="file"
          id="file-input"
          ref={fileInputRef}
          accept=".txt,.md"
          onChange={handleFileSelect}
          disabled={uploadStatus === 'uploading'}
          style={{ display: 'none' }}
        />
        <label htmlFor="file-input" className="upload-label">
          <div className="upload-icon">📄</div>
          <div className="upload-text">
            {file ? file.name : 'ファイルを選択またはドラッグ&ドロップ'}
          </div>
          <div className="upload-hint">
            最大50MB、.txt/.mdファイルのみ
          </div>
        </label>
      </div>

      {/* ファイル情報表示 */}
      {file && (
        <div className="file-info">
          <div className="file-detail">
            <span className="file-name">{file.name}</span>
            <span className="file-size">
              {(file.size / 1024 / 1024).toFixed(2)} MB
            </span>
          </div>
        </div>
      )}

      {/* プログレスバー */}
      {uploadStatus === 'uploading' && (
        <div className="progress-section">
          <div className="progress-bar">
            <div
              className="progress-fill"
              style={{ width: `${progress}%` }}
            />
          </div>
          <div className="progress-text">{progress}% 完了</div>
        </div>
      )}

      {/* ステータスメッセージ */}
      {uploadStatus === 'success' && (
        <div className="status-message success">
          ✅ アップロードが完了しました
        </div>
      )}

      {errorMessage && (
        <div className="status-message error">
          ❌ {errorMessage}
        </div>
      )}

      {/* アップロードボタン */}
      <div className="upload-actions">
        <button
          className="upload-button"
          onClick={handleUpload}
          disabled={!file || uploadStatus === 'uploading'}
        >
          {uploadStatus === 'uploading' ? 'アップロード中...' : 'アップロード'}
        </button>
        
        {file && uploadStatus !== 'uploading' && (
          <button
            className="cancel-button"
            onClick={() => {
              setFile(null);
              setErrorMessage('');
              setUploadStatus('idle');
              setProgress(0);
            }}
          >
            キャンセル
          </button>
        )}
      </div>
    </div>
  );
};

export default DocumentUpload;
