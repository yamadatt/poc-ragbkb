// APIクライアント設定とユーティリティ関数

// 基本設定
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api';
const DEFAULT_TIMEOUT = 30000; // 30秒

// 共通のAPIレスポンス型
interface APIResponse<T = any> {
  data: T;
  error?: {
    code: string;
    message: string;
    details?: any;
  };
  meta?: {
    timestamp: string;
    requestId: string;
  };
}

// エラークラス
export class APIError extends Error {
  public readonly status: number;
  public readonly code: string;
  public readonly details?: any;

  constructor(status: number, code: string, message: string, details?: any) {
    super(message);
    this.name = 'APIError';
    this.status = status;
    this.code = code;
    this.details = details;
  }

  // エラータイプの判定
  isClientError(): boolean {
    return this.status >= 400 && this.status < 500;
  }

  isServerError(): boolean {
    return this.status >= 500;
  }

  isNetworkError(): boolean {
    return this.status === 0;
  }
}

// リクエストオプション
interface RequestOptions {
  timeout?: number;
  headers?: Record<string, string>;
  signal?: AbortSignal;
}

// 基本的なHTTPクライアント
class HTTPClient {
  private baseURL: string;
  private defaultHeaders: Record<string, string>;

  constructor(baseURL: string) {
    this.baseURL = baseURL;
    this.defaultHeaders = {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    };
  }

  private async request<T>(
    method: string,
    endpoint: string,
    data?: any,
    options: RequestOptions = {}
  ): Promise<T> {
    const { timeout = DEFAULT_TIMEOUT, headers = {}, signal } = options;
    
    // タイムアウト処理
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeout);
    
    // シグナルが既に提供されている場合は、両方を組み合わせる
    const combinedSignal = signal || controller.signal;

    try {
      const config: RequestInit = {
        method,
        headers: { ...this.defaultHeaders, ...headers },
        signal: combinedSignal,
      };

      if (data && method !== 'GET') {
        if (data instanceof FormData) {
          // FormDataの場合はContent-Typeを削除（ブラウザが自動設定）
          delete config.headers!['Content-Type'];
          config.body = data;
        } else {
          config.body = JSON.stringify(data);
        }
      }

      const response = await fetch(`${this.baseURL}${endpoint}`, config);
      
      clearTimeout(timeoutId);

      // レスポンスの処理
      const contentType = response.headers.get('content-type');
      let responseData: any;

      if (contentType?.includes('application/json')) {
        responseData = await response.json();
      } else {
        responseData = await response.text();
      }

      if (!response.ok) {
        const errorMessage = responseData?.error?.message || 
                           responseData?.message || 
                           `HTTP ${response.status}: ${response.statusText}`;
        const errorCode = responseData?.error?.code || 'UNKNOWN_ERROR';
        const errorDetails = responseData?.error?.details;

        throw new APIError(response.status, errorCode, errorMessage, errorDetails);
      }

      return responseData;

    } catch (error) {
      clearTimeout(timeoutId);

      if (error instanceof APIError) {
        throw error;
      }

      if (error instanceof DOMException && error.name === 'AbortError') {
        throw new APIError(0, 'TIMEOUT', 'リクエストがタイムアウトしました');
      }

      // ネットワークエラーや予期しないエラー
      throw new APIError(0, 'NETWORK_ERROR', error instanceof Error ? error.message : 'ネットワークエラーが発生しました');
    }
  }

  async get<T>(endpoint: string, options?: RequestOptions): Promise<T> {
    return this.request<T>('GET', endpoint, undefined, options);
  }

  async post<T>(endpoint: string, data?: any, options?: RequestOptions): Promise<T> {
    return this.request<T>('POST', endpoint, data, options);
  }

  async put<T>(endpoint: string, data?: any, options?: RequestOptions): Promise<T> {
    return this.request<T>('PUT', endpoint, data, options);
  }

  async delete<T>(endpoint: string, options?: RequestOptions): Promise<T> {
    return this.request<T>('DELETE', endpoint, undefined, options);
  }
}

// APIクライアントのインスタンス
export const apiClient = new HTTPClient(API_BASE_URL);

// 型定義（コンポーネントで使用）
export interface Document {
  id: string;
  fileName: string;
  fileSize: number;
  fileType: string;
  status: 'uploaded' | 'processing' | 'indexed' | 'error';
  uploadedAt: string;
  lastModified?: string;
}

export interface Source {
  documentId: string;
  fileName: string;
  excerpt: string;
  confidence: number;
}

export interface RAGResponse {
  id: string;
  answer: string;
  sources: Source[];
  processingTimeMs: number;
  modelUsed: string;
  tokensUsed: number;
  createdAt: string;
}

export interface QueryResponse {
  id: string;
  sessionId: string;
  question: string;
  status: string;
  processingTimeMs: number;
  createdAt: string;
  updatedAt: string;
}

export interface QueryWithResponse {
  query: QueryResponse;
  response: RAGResponse;
}

export interface UploadSession {
  id: string;
  fileName: string;
  fileSize: number;
  fileType: string;
  uploadUrl: string;
  expiresAt: string;
}

export interface PaginationMeta {
  page: number;
  pageSize: number;
  totalCount: number;
  totalPages: number;
}

export interface DocumentListResponse {
  data: Document[];
  pagination: PaginationMeta;
}

// API関数群

// ヘルスチェック
export const healthAPI = {
  check: (): Promise<APIResponse<{ status: string; message: string; timestamp: string }>> =>
    apiClient.get('/health'),
};

// 文書管理API
export const documentsAPI = {
  // 文書一覧取得
  list: (params: { page?: number; pageSize?: number } = {}): Promise<DocumentListResponse> => {
    const searchParams = new URLSearchParams();
    if (params.page) searchParams.set('page', params.page.toString());
    if (params.pageSize) searchParams.set('pageSize', params.pageSize.toString());
    
    const query = searchParams.toString();
    return apiClient.get(`/documents${query ? `?${query}` : ''}`);
  },

  // 文書詳細取得
  get: (id: string): Promise<APIResponse<Document>> =>
    apiClient.get(`/documents/${id}`),

  // アップロードセッション開始
  createUploadSession: (data: {
    fileName: string;
    fileSize: number;
    fileType: string;
  }): Promise<APIResponse<UploadSession>> =>
    apiClient.post('/documents', data),

  // S3への直接アップロード（外部）
  uploadToS3: async (uploadUrl: string, file: File): Promise<void> => {
    const response = await fetch(uploadUrl, {
      method: 'PUT',
      body: file,
      headers: {
        'Content-Type': 'application/octet-stream',
      },
    });

    if (!response.ok) {
      throw new APIError(response.status, 'UPLOAD_FAILED', 'ファイルのアップロードに失敗しました');
    }
  },

  // アップロード完了通知
  completeUpload: (sessionId: string): Promise<APIResponse<Document>> =>
    apiClient.post(`/documents/${sessionId}/complete-upload`),

  // 文書削除
  delete: (id: string): Promise<APIResponse<void>> =>
    apiClient.delete(`/documents/${id}`),
};

// クエリ・RAG API
export const queriesAPI = {
  // 質問送信
  create: (data: {
    question: string;
    sessionId: string;
  }): Promise<APIResponse<QueryWithResponse>> =>
    apiClient.post('/queries', data, { timeout: 60000 }), // RAGは時間がかかるため60秒

  // セッション履歴取得
  getHistory: (sessionId: string): Promise<APIResponse<QueryWithResponse[]>> =>
    apiClient.get(`/queries/${sessionId}/history`),
};

// 便利なユーティリティ関数

// エラーメッセージの統一フォーマット
export function formatErrorMessage(error: unknown): string {
  if (error instanceof APIError) {
    return error.message;
  }
  
  if (error instanceof Error) {
    return error.message;
  }
  
  return '予期しないエラーが発生しました';
}

// 再試行機能付きリクエスト
export async function withRetry<T>(
  operation: () => Promise<T>,
  maxRetries: number = 3,
  delayMs: number = 1000
): Promise<T> {
  let lastError: Error;

  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    try {
      return await operation();
    } catch (error) {
      lastError = error as Error;
      
      // クライアントエラー（4xx）の場合は再試行しない
      if (error instanceof APIError && error.isClientError()) {
        throw error;
      }

      // 最後の試行の場合は例外を投げる
      if (attempt === maxRetries) {
        throw lastError;
      }

      // 指数バックオフで待機
      const delay = delayMs * Math.pow(2, attempt - 1);
      await new Promise(resolve => setTimeout(resolve, delay));
    }
  }

  throw lastError!;
}

// キャンセル可能なリクエスト
export function createCancellableRequest<T>(
  requestFunction: (signal: AbortSignal) => Promise<T>
): { promise: Promise<T>; cancel: () => void } {
  const controller = new AbortController();
  
  return {
    promise: requestFunction(controller.signal),
    cancel: () => controller.abort(),
  };
}

// デバウンス機能付きリクエスト
export function createDebouncedRequest<T extends any[], R>(
  requestFunction: (...args: T) => Promise<R>,
  delayMs: number = 300
): (...args: T) => Promise<R> {
  let timeoutId: number;
  let resolveFunction: (value: R) => void;
  let rejectFunction: (error: any) => void;

  return (...args: T): Promise<R> => {
    clearTimeout(timeoutId);

    return new Promise<R>((resolve, reject) => {
      resolveFunction = resolve;
      rejectFunction = reject;

      timeoutId = window.setTimeout(async () => {
        try {
          const result = await requestFunction(...args);
          resolveFunction(result);
        } catch (error) {
          rejectFunction(error);
        }
      }, delayMs);
    });
  };
}

export { APIResponse };