import React, { useEffect, useRef, useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import { useUIStore } from '../store/ui';
import { usePreferenceStore } from '../store/preferences';
import { translate } from '../i18n';
import { toast } from '../utils/notify';

const SHOW_DELAY = 200;
// 持续 8s 仍未归零，给出 info toast 兜底；与 GET 默认 10s 超时错峰，
// 让用户在写操作（30s）等待中段先得到反馈。
const LONG_PENDING_MS = 8000;

/**
 * 请求级顶部进度条，由 http 拦截器的在途请求计数驱动。
 * 200ms 延迟显示，避免快速请求造成闪烁；归零后淡出。
 */
const RequestProgressBar: React.FC = () => {
  const globalLoading = useUIStore((state) => state.globalLoading);
  const language = usePreferenceStore((state) => state.language);
  const [visible, setVisible] = useState(false);
  const longTimerRef = useRef<number | null>(null);
  const longToastShownRef = useRef(false);

  useEffect(() => {
    if (globalLoading > 0) {
      // 仅当请求持续超过 SHOW_DELAY 才显示，短请求不打扰
      const timer = window.setTimeout(() => setVisible(true), SHOW_DELAY);
      return () => window.clearTimeout(timer);
    }
    setVisible(false);
    return undefined;
  }, [globalLoading]);

  useEffect(() => {
    if (globalLoading > 0) {
      if (longTimerRef.current == null && !longToastShownRef.current) {
        longTimerRef.current = window.setTimeout(() => {
          toast.info(translate(language, 'toast.requestPending'));
          longToastShownRef.current = true;
          longTimerRef.current = null;
        }, LONG_PENDING_MS);
      }
      return () => {
        if (longTimerRef.current != null) {
          window.clearTimeout(longTimerRef.current);
          longTimerRef.current = null;
        }
      };
    }
    // 归零，重置等待下一轮
    if (longTimerRef.current != null) {
      window.clearTimeout(longTimerRef.current);
      longTimerRef.current = null;
    }
    longToastShownRef.current = false;
    return undefined;
  }, [globalLoading, language]);

  return (
    <AnimatePresence>
      {visible && (
        <motion.div
          aria-hidden="true"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.2 }}
          className="fixed left-0 top-0 z-[95] h-[2px] w-full overflow-hidden"
        >
          <span className="block h-full w-2/5 bg-gradient-to-r from-transparent via-[var(--ochre)] to-transparent" style={{ animation: 'route-pending-slide 920ms cubic-bezier(0.22, 1, 0.36, 1) infinite' }} />
        </motion.div>
      )}
    </AnimatePresence>
  );
};

export default RequestProgressBar;
