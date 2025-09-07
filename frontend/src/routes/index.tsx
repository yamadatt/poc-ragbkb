// ルーティング設定
// 現在のアプリケーションはSPA（Single Page Application）として
// App.tsx内でのビュー切り替えで実装していますが、
// 将来的なURL-based routingの拡張に備えたルーティング設定を提供します

import React from 'react';

// React Router（将来的な拡張用）
// import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
// import App from '../App';

// 現在のルーティング状態管理
export interface RouteState {
  currentPath: string;
  params: Record<string, string>;
  query: URLSearchParams;
}

// アプリケーションのルート定義
export interface AppRoute {
  path: string;
  name: string;
  component: React.ComponentType<any>;
  title: string;
  icon?: string;
  requiresData?: boolean;
}

// ビュー識別子
export type ViewType = 'upload' | 'query' | 'documents' | 'history' | 'settings';

// 内部ルーティングの設定
export const APP_ROUTES: Record<ViewType, AppRoute> = {
  upload: {
    path: '/upload',
    name: 'upload',
    component: () => <div>文書アップロード</div>,
    title: '文書アップロード',
    icon: '📤',
  },
  query: {
    path: '/query',
    name: 'query', 
    component: () => <div>質問・回答</div>,
    title: '質問・回答',
    icon: '💬',
  },
  documents: {
    path: '/documents',
    name: 'documents',
    component: () => <div>文書一覧</div>,
    title: '文書一覧',
    icon: '📂',
  },
  history: {
    path: '/history',
    name: 'history',
    component: () => <div>履歴</div>,
    title: '履歴',
    icon: '📚',
    requiresData: true,
  },
  settings: {
    path: '/settings',
    name: 'settings',
    component: () => <div>設定</div>,
    title: '設定',
    icon: '⚙️',
  },
};

// URLパラメータの管理
export class RouteManager {
  private currentRoute: RouteState;
  private listeners: Array<(route: RouteState) => void> = [];

  constructor() {
    this.currentRoute = this.parseCurrentURL();
    
    // ブラウザの戻る/進むボタンの対応
    window.addEventListener('popstate', this.handlePopState.bind(this));
  }

  private parseCurrentURL(): RouteState {
    const url = new URL(window.location.href);
    return {
      currentPath: url.pathname,
      params: this.extractParams(url.pathname),
      query: url.searchParams,
    };
  }

  private extractParams(path: string): Record<string, string> {
    // シンプルなパラメータ抽出（例: /documents/:id）
    const params: Record<string, string> = {};
    
    // 将来的な拡張のためのプレースホルダー
    // const pathSegments = path.split('/').filter(Boolean);
    // ルートパターンマッチングロジックをここに追加
    
    return params;
  }

  private handlePopState = () => {
    this.currentRoute = this.parseCurrentURL();
    this.notifyListeners();
  };

  // 現在のルート取得
  getCurrentRoute(): RouteState {
    return { ...this.currentRoute };
  }

  // プログラマティックナビゲーション
  navigate(path: string, options: { replace?: boolean; state?: any } = {}) {
    const url = new URL(window.location.origin + path);
    
    if (options.replace) {
      window.history.replaceState(options.state, '', url.toString());
    } else {
      window.history.pushState(options.state, '', url.toString());
    }
    
    this.currentRoute = this.parseCurrentURL();
    this.notifyListeners();
  }

  // クエリパラメータの更新
  updateQuery(updates: Record<string, string | null>) {
    const url = new URL(window.location.href);
    
    Object.entries(updates).forEach(([key, value]) => {
      if (value === null) {
        url.searchParams.delete(key);
      } else {
        url.searchParams.set(key, value);
      }
    });
    
    window.history.replaceState(null, '', url.toString());
    this.currentRoute = this.parseCurrentURL();
    this.notifyListeners();
  }

  // ルート変更のリスナー登録
  onRouteChange(callback: (route: RouteState) => void) {
    this.listeners.push(callback);
    
    // クリーンアップ関数を返す
    return () => {
      const index = this.listeners.indexOf(callback);
      if (index > -1) {
        this.listeners.splice(index, 1);
      }
    };
  }

  private notifyListeners() {
    this.listeners.forEach(callback => callback(this.currentRoute));
  }
}

// グローバルルートマネージャーのインスタンス
export const routeManager = new RouteManager();

// React Hookでルーティング情報を取得
export function useRouting() {
  const [currentRoute, setCurrentRoute] = React.useState(routeManager.getCurrentRoute());

  React.useEffect(() => {
    return routeManager.onRouteChange(setCurrentRoute);
  }, []);

  return {
    currentRoute,
    navigate: routeManager.navigate.bind(routeManager),
    updateQuery: routeManager.updateQuery.bind(routeManager),
  };
}

// セッション管理とルーティングの統合
export function useSessionRouting(sessionId: string) {
  const routing = useRouting();

  // セッションIDをクエリパラメータに保存
  React.useEffect(() => {
    const currentSessionId = routing.currentRoute.query.get('session');
    if (currentSessionId !== sessionId) {
      routing.updateQuery({ session: sessionId });
    }
  }, [sessionId, routing]);

  return routing;
}

// ブレッドクラム生成
export function generateBreadcrumb(route: RouteState): Array<{ name: string; path: string }> {
  const segments = route.currentPath.split('/').filter(Boolean);
  const breadcrumb: Array<{ name: string; path: string }> = [
    { name: 'ホーム', path: '/' }
  ];

  let currentPath = '';
  segments.forEach(segment => {
    currentPath += `/${segment}`;
    
    // ルート定義から名前を取得
    const matchedRoute = Object.values(APP_ROUTES).find(route => 
      route.path === currentPath
    );
    
    breadcrumb.push({
      name: matchedRoute?.title || segment,
      path: currentPath,
    });
  });

  return breadcrumb;
}

// パーマリンク生成（将来的なシェア機能用）
export function createPermalink(viewType: ViewType, params?: Record<string, string>): string {
  const route = APP_ROUTES[viewType];
  let path = route.path;
  
  // パラメータの置換
  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      path = path.replace(`:${key}`, value);
    });
  }
  
  return `${window.location.origin}${path}`;
}

// Deep linking対応（特定のクエリや文書への直接リンク）
export function parseDeepLink(): {
  view?: ViewType;
  documentId?: string;
  queryId?: string;
  sessionId?: string;
} {
  const route = routeManager.getCurrentRoute();
  const query = route.query;
  
  return {
    view: query.get('view') as ViewType || undefined,
    documentId: query.get('doc') || undefined,
    queryId: query.get('q') || undefined,
    sessionId: query.get('session') || undefined,
  };
}

// 将来的なReact Router統合のためのコンポーネント（現在は使用されていません）
export const AppRouter: React.FC = () => {
  // 現在はApp.tsx内でのビュー切り替えを使用
  // 将来的にURL-based routingが必要になった場合は以下のような実装を行います：
  
  /*
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Navigate to="/upload" replace />} />
        <Route path="/upload" element={<App />} />
        <Route path="/query" element={<App />} />
        <Route path="/documents" element={<App />} />
        <Route path="/documents/:id" element={<App />} />
        <Route path="/history" element={<App />} />
        <Route path="/settings" element={<App />} />
      </Routes>
    </BrowserRouter>
  );
  */
  
  return null; // 現在は使用しない
};

// URL状態の同期（App.tsxで使用）
export function useSyncUrlWithState(currentView: ViewType) {
  const routing = useRouting();
  
  React.useEffect(() => {
    const expectedPath = APP_ROUTES[currentView].path;
    if (routing.currentRoute.currentPath !== expectedPath) {
      // URLを現在のビューと同期（ブラウザ履歴には残さない）
      routing.navigate(expectedPath, { replace: true });
    }
  }, [currentView, routing]);
  
  return routing;
}

// エクスポート用の統合設定
export const routingConfig = {
  routes: APP_ROUTES,
  manager: routeManager,
  hooks: {
    useRouting,
    useSessionRouting,
    useSyncUrlWithState,
  },
  utils: {
    generateBreadcrumb,
    createPermalink,
    parseDeepLink,
  },
};