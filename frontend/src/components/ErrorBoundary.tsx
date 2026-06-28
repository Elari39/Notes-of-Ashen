import React from 'react';
import { translate } from '../i18n';
import { usePreferenceStore, type Language } from '../store/preferences';

interface ErrorBoundaryProps {
  children: React.ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
}

/**
 * 应用级错误边界：捕获子树渲染期异常（如 Markdown 渲染失败、图表初始化异常），
 * 渲染降级 UI 并提供刷新入口，避免整页白屏。
 *
 * 注意：React 错误边界不会捕获事件回调、异步代码与 setTimeout/Promise 中的错误，
 * 此处仅兜底渲染期异常，不替代页面级 Loading/Error/Empty 状态。
 */
class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  private unsubscribe?: () => void;
  private language: Language;

  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false };
    this.language = usePreferenceStore.getState().language;
  }

  static getDerivedStateFromError(): ErrorBoundaryState {
    return { hasError: true };
  }

  componentDidMount(): void {
    // 订阅语言变化，保证降级 UI 跟随用户语言切换更新。
    this.unsubscribe = usePreferenceStore.subscribe((state) => {
      if (state.language !== this.language) {
        this.language = state.language;
        if (this.state.hasError) {
          this.forceUpdate();
        }
      }
    });
  }

  componentWillUnmount(): void {
    this.unsubscribe?.();
  }

  handleReload = () => {
    window.location.reload();
  };

  render() {
    if (!this.state.hasError) {
      return this.props.children;
    }
    const language = this.language;
    return (
      <div
        role="alert"
        className="flex min-h-[60vh] flex-col items-center justify-center gap-4 px-6 py-16 text-center"
      >
        <h2 className="text-lg font-bold tracking-wide text-ink">
          {translate(language, 'error.boundaryTitle')}
        </h2>
        <p className="max-w-md text-sm text-ink-light">
          {translate(language, 'error.boundaryDesc')}
        </p>
        <button
          type="button"
          onClick={this.handleReload}
          className="border border-mountain-grey px-5 py-2 text-sm font-medium text-ink transition-colors hover:bg-paper"
        >
          {translate(language, 'error.boundaryReload')}
        </button>
      </div>
    );
  }
}

export default ErrorBoundary;
