/**
 * 轻量焦点陷阱工具，供对话框/灯箱等模态场景使用。
 * 不引入第三方依赖，仅实现 Tab/Shift+Tab 循环。
 */

const FOCUSABLE_SELECTOR = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled])',
  'textarea:not([disabled])',
  'select:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',');

export const getFocusableElements = (container: HTMLElement): HTMLElement[] => {
  return Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)).filter(
    (el) => el.offsetParent !== null || el === document.activeElement,
  );
};

/**
 * 在 keydown(Tab) 事件中调用，把焦点限制在 container 内循环。
 * 非自定义移动，仅处理默认 Tab 行为的边界回绕。
 */
export const trapFocus = (container: HTMLElement, event: KeyboardEvent) => {
  if (event.key !== 'Tab') {
    return;
  }
  const focusable = getFocusableElements(container);
  if (focusable.length === 0) {
    event.preventDefault();
    container.focus();
    return;
  }

  const first = focusable[0];
  const last = focusable[focusable.length - 1];
  const active = document.activeElement as HTMLElement | null;

  if (event.shiftKey) {
    if (active === first || !container.contains(active)) {
      event.preventDefault();
      last.focus();
    }
  } else {
    if (active === last) {
      event.preventDefault();
      first.focus();
    }
  }
};
