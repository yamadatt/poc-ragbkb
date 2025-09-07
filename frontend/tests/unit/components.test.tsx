// フロントエンドコンポーネント単体テスト
import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react';
import '@testing-library/jest-dom';
import userEvent from '@testing-library/user-event';

// コンポーネントのインポート
import DocumentUpload from '../../src/components/DocumentUpload';
import QueryInput from '../../src/components/QueryInput';
import ResponseDisplay from '../../src/components/ResponseDisplay';
import DocumentList from '../../src/components/DocumentList';

// モックデータ
const mockDocument = {
  id: 'doc123',
  fileName: 'test.txt',
  fileSize: 1024,
  fileType: 'txt' as const,
  status: 'indexed' as const,
  uploadedAt: '2024-01-01T12:00:00Z',
};

const mockQueryWithResponse = {
  query: {
    id: 'query123',
    sessionId: 'session456',
    question: 'AWS Bedrockについて教えてください',
    status: 'completed',
    processingTimeMs: 1500,
    createdAt: '2024-01-01T12:00:00Z',
    updatedAt: '2024-01-01T12:00:01Z',
  },
  response: {
    id: 'resp123',
    answer: 'AWS Bedrockは機械学習モデルを簡単に利用できるマネージドサービスです。',
    sources: [
      {
        documentId: 'doc123',
        fileName: 'aws-bedrock.txt',
        excerpt: 'AWS Bedrockは...',
        confidence: 0.9,
      },
    ],
    processingTimeMs: 1500,
    modelUsed: 'claude-v1',
    tokensUsed: 150,
    createdAt: '2024-01-01T12:00:01Z',
  },
};

// fetch のモック
global.fetch = vi.fn();

beforeEach(() => {
  // fetchのモックをリセット
  vi.resetAllMocks();
});

afterEach(() => {
  cleanup();
});

describe('DocumentUpload', () => {
  it('コンポーネントが正常にレンダリングされる', () => {
    render(<DocumentUpload />);
    
    expect(screen.getByText('文書アップロード')).toBeInTheDocument();
    expect(screen.getByText('ファイルを選択またはドラッグ&ドロップ')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'アップロード' })).toBeDisabled();
  });

  it('有効なファイルが選択されるとアップロードボタンが有効になる', async () => {
    const user = userEvent.setup();
    render(<DocumentUpload />);
    
    const fileInput = screen.getByLabelText(/ファイルを選択またはドラッグ&ドロップ/);
    const testFile = new File(['test content'], 'test.txt', { type: 'text/plain' });
    
    await user.upload(fileInput, testFile);
    
    expect(screen.getByText('test.txt')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'アップロード' })).toBeEnabled();
  });

  it('ファイルサイズ制限を超えるとエラーメッセージが表示される', async () => {
    const user = userEvent.setup();
    render(<DocumentUpload />);
    
    const fileInput = screen.getByLabelText(/ファイルを選択またはドラッグ&ドロップ/);
    const largeFile = new File(['x'.repeat(51 * 1024 * 1024)], 'large.txt', { type: 'text/plain' });
    
    await user.upload(fileInput, largeFile);
    
    expect(screen.getByText('ファイルサイズが制限を超えています（最大50MB）')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'アップロード' })).toBeDisabled();
  });

  it('無効なファイルタイプでエラーメッセージが表示される', async () => {
    const user = userEvent.setup();
    render(<DocumentUpload />);
    
    const fileInput = screen.getByLabelText(/ファイルを選択またはドラッグ&ドロップ/);
    const invalidFile = new File(['test'], 'test.pdf', { type: 'application/pdf' });
    
    await user.upload(fileInput, invalidFile);
    
    expect(screen.getByText('サポートされていないファイルタイプです（.txt, .mdのみ）')).toBeInTheDocument();
  });

  it('アップロード完了時にコールバックが呼ばれる', async () => {
    const mockOnUploadComplete = vi.fn();
    const user = userEvent.setup();
    
    // fetchのモック設定
    (global.fetch as any)
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: { id: 'session123', uploadUrl: 'https://s3.example.com/upload' } }),
      })
      .mockResolvedValueOnce({ ok: true }) // S3アップロード
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: mockDocument }),
      });
    
    render(<DocumentUpload onUploadComplete={mockOnUploadComplete} />);
    
    const fileInput = screen.getByLabelText(/ファイルを選択またはドラッグ&ドロップ/);
    const testFile = new File(['test content'], 'test.txt', { type: 'text/plain' });
    
    await user.upload(fileInput, testFile);
    await user.click(screen.getByRole('button', { name: 'アップロード' }));
    
    await waitFor(() => {
      expect(mockOnUploadComplete).toHaveBeenCalledWith(mockDocument);
    });
  });
});

describe('QueryInput', () => {
  const defaultProps = {
    sessionId: 'session123',
    onQuerySubmit: vi.fn(),
    onQueryError: vi.fn(),
  };

  it('コンポーネントが正常にレンダリングされる', () => {
    render(<QueryInput {...defaultProps} />);
    
    expect(screen.getByText('質問を入力')).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/AWS Bedrock Knowledge Baseの使い方/)).toBeInTheDocument();
    expect(screen.getByText('質問を送信')).toBeInTheDocument();
  });

  it('質問を入力すると文字数カウントが更新される', async () => {
    const user = userEvent.setup();
    render(<QueryInput {...defaultProps} />);
    
    const textarea = screen.getByRole('textbox');
    await user.type(textarea, 'テスト質問');
    
    expect(screen.getByText('5 / 1000文字')).toBeInTheDocument();
  });

  it('文字数制限を超えると警告が表示される', async () => {
    const user = userEvent.setup();
    render(<QueryInput {...defaultProps} />);
    
    const textarea = screen.getByRole('textbox');
    const longText = 'あ'.repeat(1001);
    
    await user.type(textarea, longText);
    
    expect(screen.getByText(/文字超過/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '質問を送信' })).toBeDisabled();
  });

  it('Ctrl+Enterで質問が送信される', async () => {
    const user = userEvent.setup();
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: mockQueryWithResponse }),
    });
    
    render(<QueryInput {...defaultProps} />);
    
    const textarea = screen.getByRole('textbox');
    await user.type(textarea, 'テスト質問');
    await user.keyboard('{Control>}{Enter}{/Control}');
    
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith('/api/queries', expect.any(Object));
    });
  });

  it('空の質問では送信できない', async () => {
    const user = userEvent.setup();
    render(<QueryInput {...defaultProps} />);
    
    const submitButton = screen.getByRole('button', { name: '質問を送信' });
    
    expect(submitButton).toBeDisabled();
    
    // 空白のみ入力
    const textarea = screen.getByRole('textbox');
    await user.type(textarea, '   ');
    
    expect(submitButton).toBeDisabled();
  });

  it('質問送信が成功するとコールバックが呼ばれる', async () => {
    const mockOnQuerySubmit = vi.fn();
    const user = userEvent.setup();
    
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: mockQueryWithResponse }),
    });
    
    render(<QueryInput {...defaultProps} onQuerySubmit={mockOnQuerySubmit} />);
    
    const textarea = screen.getByRole('textbox');
    await user.type(textarea, 'テスト質問');
    await user.click(screen.getByRole('button', { name: '質問を送信' }));
    
    await waitFor(() => {
      expect(mockOnQuerySubmit).toHaveBeenCalledWith(mockQueryWithResponse);
    });
  });
});

describe('ResponseDisplay', () => {
  it('質問がない場合は空の状態が表示される', () => {
    render(<ResponseDisplay queryWithResponse={null} />);
    
    expect(screen.getByText('質問をお待ちしています')).toBeInTheDocument();
    expect(screen.getByText(/上記のフォームから質問を入力すると/)).toBeInTheDocument();
  });

  it('質問と回答が正常に表示される', () => {
    render(<ResponseDisplay queryWithResponse={mockQueryWithResponse} />);
    
    expect(screen.getByText('AWS Bedrockについて教えてください')).toBeInTheDocument();
    expect(screen.getByText('AWS Bedrockは機械学習モデルを簡単に利用できるマネージドサービスです。')).toBeInTheDocument();
  });

  it('情報源が表示される', () => {
    render(<ResponseDisplay queryWithResponse={mockQueryWithResponse} />);
    
    expect(screen.getByText('参考情報源 (1件)')).toBeInTheDocument();
    expect(screen.getByText('aws-bedrock.txt')).toBeInTheDocument();
    expect(screen.getByText('信頼度: 高い (90%)')).toBeInTheDocument();
  });

  it('メタデータが表示される', () => {
    render(<ResponseDisplay queryWithResponse={mockQueryWithResponse} />);
    
    expect(screen.getByText('1.5秒')).toBeInTheDocument();
    expect(screen.getByText('claude-v1')).toBeInTheDocument();
    expect(screen.getByText('150')).toBeInTheDocument();
  });

  it('コピーボタンが動作する', async () => {
    // Clipboard API のモック
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn(() => Promise.resolve()),
      },
    });
    
    const user = userEvent.setup();
    render(<ResponseDisplay queryWithResponse={mockQueryWithResponse} />);
    
    const copyButtons = screen.getAllByTitle(/コピー/);
    await user.click(copyButtons[0]); // 質問のコピーボタン
    
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('AWS Bedrockについて教えてください');
  });

  it('情報源の展開/折りたたみが動作する', async () => {
    const user = userEvent.setup();
    render(<ResponseDisplay queryWithResponse={mockQueryWithResponse} />);
    
    const sourceHeader = screen.getByText('aws-bedrock.txt').closest('.source-header');
    expect(sourceHeader).toBeInTheDocument();
    
    // 初期状態では折りたたまれている
    expect(screen.queryByText('AWS Bedrockは...')).not.toBeInTheDocument();
    
    // クリックして展開
    await user.click(sourceHeader!);
    expect(screen.getByText('AWS Bedrockは...')).toBeInTheDocument();
  });
});

describe('DocumentList', () => {
  beforeEach(() => {
    (global.fetch as any).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({
        data: [mockDocument],
        pagination: {
          page: 1,
          pageSize: 10,
          totalCount: 1,
          totalPages: 1,
        },
      }),
    });
  });

  it('コンポーネントが正常にレンダリングされる', async () => {
    render(<DocumentList />);
    
    expect(screen.getByText('アップロード済み文書')).toBeInTheDocument();
    
    await waitFor(() => {
      expect(screen.getByText('test.txt')).toBeInTheDocument();
    });
  });

  it('文書リストが表示される', async () => {
    render(<DocumentList />);
    
    await waitFor(() => {
      expect(screen.getByText('test.txt')).toBeInTheDocument();
      expect(screen.getByText('1.0 KB')).toBeInTheDocument();
      expect(screen.getByText('インデックス済み')).toBeInTheDocument();
    });
  });

  it('更新ボタンが動作する', async () => {
    const user = userEvent.setup();
    render(<DocumentList />);
    
    await waitFor(() => {
      expect(screen.getByText('test.txt')).toBeInTheDocument();
    });
    
    const refreshButton = screen.getByRole('button', { name: /更新/ });
    await user.click(refreshButton);
    
    // fetch が再度呼ばれることを確認
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledTimes(2); // 初回 + 更新
    });
  });

  it('文書選択ができる', async () => {
    const user = userEvent.setup();
    render(<DocumentList />);
    
    await waitFor(() => {
      expect(screen.getByText('test.txt')).toBeInTheDocument();
    });
    
    const checkbox = screen.getByRole('checkbox', { name: '' });
    await user.click(checkbox);
    
    expect(checkbox).toBeChecked();
  });

  it('文書削除が動作する', async () => {
    const mockOnDocumentDelete = vi.fn();
    const user = userEvent.setup();
    
    // confirm のモック
    global.confirm = vi.fn(() => true);
    
    // 削除API のモック
    (global.fetch as any)
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({
          data: [mockDocument],
          pagination: { page: 1, pageSize: 10, totalCount: 1, totalPages: 1 },
        }),
      })
      .mockResolvedValueOnce({ ok: true }) // DELETE
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({
          data: [],
          pagination: { page: 1, pageSize: 10, totalCount: 0, totalPages: 1 },
        }),
      });
    
    render(<DocumentList onDocumentDelete={mockOnDocumentDelete} />);
    
    await waitFor(() => {
      expect(screen.getByText('test.txt')).toBeInTheDocument();
    });
    
    const deleteButton = screen.getByTitle('文書を削除');
    await user.click(deleteButton);
    
    await waitFor(() => {
      expect(mockOnDocumentDelete).toHaveBeenCalledWith(mockDocument);
    });
  });

  it('空の状態が表示される', async () => {
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({
        data: [],
        pagination: { page: 1, pageSize: 10, totalCount: 0, totalPages: 1 },
      }),
    });
    
    render(<DocumentList />);
    
    await waitFor(() => {
      expect(screen.getByText('文書がありません')).toBeInTheDocument();
      expect(screen.getByText('文書をアップロードすると、ここに表示されます。')).toBeInTheDocument();
    });
  });
});

// 統合テスト的なテスト
describe('Component Integration', () => {
  it('DocumentUpload と DocumentList の連携', async () => {
    const user = userEvent.setup();
    let refreshTrigger = 0;
    
    const MockApp = () => {
      const [trigger, setTrigger] = React.useState(refreshTrigger);
      
      const handleUploadComplete = () => {
        setTrigger(prev => prev + 1);
      };
      
      return (
        <div>
          <DocumentUpload onUploadComplete={handleUploadComplete} />
          <DocumentList refreshTrigger={trigger} />
        </div>
      );
    };
    
    // fetchのモック
    (global.fetch as any)
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({
          data: [],
          pagination: { page: 1, pageSize: 10, totalCount: 0, totalPages: 1 },
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: { id: 'session123', uploadUrl: 'https://s3.example.com/upload' } }),
      })
      .mockResolvedValueOnce({ ok: true })
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: mockDocument }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({
          data: [mockDocument],
          pagination: { page: 1, pageSize: 10, totalCount: 1, totalPages: 1 },
        }),
      });
    
    render(<MockApp />);
    
    // 初期状態では文書がない
    await waitFor(() => {
      expect(screen.getByText('文書がありません')).toBeInTheDocument();
    });
    
    // ファイルをアップロード
    const fileInput = screen.getByLabelText(/ファイルを選択またはドラッグ&ドロップ/);
    const testFile = new File(['test content'], 'test.txt', { type: 'text/plain' });
    
    await user.upload(fileInput, testFile);
    await user.click(screen.getByRole('button', { name: 'アップロード' }));
    
    // アップロード完了後、文書リストが更新される
    await waitFor(() => {
      expect(screen.getByText('test.txt')).toBeInTheDocument();
    }, { timeout: 5000 });
  });
});

// エラーハンドリングのテスト
describe('Error Handling', () => {
  it('アップロードエラーが正しく処理される', async () => {
    const mockOnUploadError = vi.fn();
    const user = userEvent.setup();
    
    (global.fetch as any).mockRejectedValueOnce(new Error('Network error'));
    
    render(<DocumentUpload onUploadError={mockOnUploadError} />);
    
    const fileInput = screen.getByLabelText(/ファイルを選択またはドラッグ&ドロップ/);
    const testFile = new File(['test'], 'test.txt', { type: 'text/plain' });
    
    await user.upload(fileInput, testFile);
    await user.click(screen.getByRole('button', { name: 'アップロード' }));
    
    await waitFor(() => {
      expect(mockOnUploadError).toHaveBeenCalledWith('Network error');
    });
  });

  it('クエリエラーが正しく処理される', async () => {
    const mockOnQueryError = vi.fn();
    const user = userEvent.setup();
    
    (global.fetch as any).mockResolvedValueOnce({
      ok: false,
      status: 500,
      json: () => Promise.resolve({
        error: { message: 'Internal server error' }
      }),
    });
    
    render(<QueryInput sessionId="test" onQueryError={mockOnQueryError} />);
    
    const textarea = screen.getByRole('textbox');
    await user.type(textarea, 'テスト質問');
    await user.click(screen.getByRole('button', { name: '質問を送信' }));
    
    await waitFor(() => {
      expect(mockOnQueryError).toHaveBeenCalledWith('Internal server error');
    });
  });
});
