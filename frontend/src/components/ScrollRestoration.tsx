import { useEffect, useLayoutEffect, useRef } from 'react';
import { useLocation, useNavigationType } from 'react-router-dom';

type ScrollPosition = {
  x: number;
  y: number;
};

type ScrollPositions = Record<string, ScrollPosition>;

const storageKey = 'notes-of-ashen:scroll-positions:v1';

const isScrollPosition = (value: unknown): value is ScrollPosition => {
  if (typeof value !== 'object' || value === null) {
    return false;
  }

  const position = value as Record<string, unknown>;
  return typeof position.x === 'number' && typeof position.y === 'number';
};

const readScrollPositions = (): ScrollPositions => {
  try {
    const value = window.sessionStorage.getItem(storageKey);
    if (!value) {
      return {};
    }

    const parsed: unknown = JSON.parse(value);
    if (typeof parsed !== 'object' || parsed === null) {
      return {};
    }

    return Object.entries(parsed).reduce<ScrollPositions>((positions, [key, position]) => {
      if (isScrollPosition(position)) {
        positions[key] = position;
      }
      return positions;
    }, {});
  } catch {
    return {};
  }
};

const persistScrollPosition = (positions: ScrollPositions, key: string) => {
  positions[key] = { x: window.scrollX, y: window.scrollY };

  try {
    window.sessionStorage.setItem(storageKey, JSON.stringify(positions));
  } catch {
    // 隐私模式或存储空间不足时，当前会话仍保留内存中的恢复能力。
  }
};

const scrollToHash = (hash: string) => {
  const encodedTargetId = hash.slice(1);
  let targetId = encodedTargetId;

  try {
    targetId = decodeURIComponent(encodedTargetId);
  } catch {
    // 非法百分号编码仍按原始值查找，避免 URL 输入导致页面渲染失败。
  }

  if (!targetId) {
    return false;
  }

  const target = document.getElementById(targetId) ?? document.getElementsByName(targetId)[0];

  if (!target) {
    return false;
  }

  target.scrollIntoView();
  return true;
};

const ScrollRestoration = () => {
  const location = useLocation();
  const navigationType = useNavigationType();
  const positionsRef = useRef<ScrollPositions | null>(null);
  const activeKeyRef = useRef('');
  const locationKey = `${location.key}:${location.pathname}${location.search}${location.hash}`;

  if (positionsRef.current === null) {
    positionsRef.current = readScrollPositions();
  }

  useLayoutEffect(() => {
    const previousScrollRestoration = window.history.scrollRestoration;
    window.history.scrollRestoration = 'manual';

    return () => {
      window.history.scrollRestoration = previousScrollRestoration;
    };
  }, []);

  useLayoutEffect(() => {
    activeKeyRef.current = locationKey;

    return () => {
      persistScrollPosition(positionsRef.current ?? {}, locationKey);
    };
  }, [locationKey]);

  useEffect(() => {
    let frame: number | undefined;

    const saveCurrentPosition = () => {
      persistScrollPosition(positionsRef.current ?? {}, activeKeyRef.current);
    };

    const handleScroll = () => {
      if (frame !== undefined) {
        return;
      }

      frame = window.requestAnimationFrame(() => {
        frame = undefined;
        saveCurrentPosition();
      });
    };

    const handlePageHide = () => {
      if (frame !== undefined) {
        window.cancelAnimationFrame(frame);
        frame = undefined;
      }
      saveCurrentPosition();
    };

    window.addEventListener('scroll', handleScroll, { passive: true });
    window.addEventListener('pagehide', handlePageHide);

    return () => {
      if (frame !== undefined) {
        window.cancelAnimationFrame(frame);
      }
      saveCurrentPosition();
      window.removeEventListener('scroll', handleScroll);
      window.removeEventListener('pagehide', handlePageHide);
    };
  }, []);

  useLayoutEffect(() => {
    let observer: MutationObserver | undefined;
    let timeout: number | undefined;
    const frame = window.requestAnimationFrame(() => {
      if (location.hash) {
        if (scrollToHash(location.hash)) {
          return;
        }

        observer = new MutationObserver(() => {
          if (scrollToHash(location.hash)) {
            observer?.disconnect();
          }
        });
        observer.observe(document.getElementById('main-content') ?? document.body, {
          childList: true,
          subtree: true,
        });
        timeout = window.setTimeout(() => observer?.disconnect(), 1200);
        return;
      }

      if (navigationType === 'POP') {
        const savedPosition = (positionsRef.current ?? {})[locationKey];
        if (savedPosition) {
          window.scrollTo(savedPosition.x, savedPosition.y);
          return;
        }
      }

      window.scrollTo(0, 0);
    });

    return () => {
      window.cancelAnimationFrame(frame);
      observer?.disconnect();
      if (timeout !== undefined) {
        window.clearTimeout(timeout);
      }
    };
  }, [location.hash, locationKey, navigationType]);

  return null;
};

export default ScrollRestoration;
