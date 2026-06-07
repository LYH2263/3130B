import { useEffect, useRef, useState, useCallback } from 'react';
import { reportProctorEvents, getProctorStudentStatus } from '../api/client';

const EVENT_TYPE_TAB_SWITCH = 'tab_switch';
const EVENT_TYPE_BLUR = 'blur';
const EVENT_TYPE_COPY = 'copy';
const EVENT_TYPE_PASTE = 'paste';
const EVENT_TYPE_FULLSCREEN_EXIT = 'fullscreen_exit';
const EVENT_TYPE_RECONNECT = 'reconnect';

const EVENT_LABELS = {
  [EVENT_TYPE_TAB_SWITCH]: '切屏',
  [EVENT_TYPE_BLUR]: '失焦',
  [EVENT_TYPE_COPY]: '复制',
  [EVENT_TYPE_PASTE]: '粘贴',
  [EVENT_TYPE_FULLSCREEN_EXIT]: '全屏退出',
  [EVENT_TYPE_RECONNECT]: '断线重连',
};

const BATCH_INTERVAL = 3000;
const MAX_BATCH_SIZE = 20;
const DEBOUNCE_MS = 500;

export function useProctor(examId, token, options = {}) {
  const { enabled = true, onStatusChange, onForceSubmit } = options;

  const [status, setStatus] = useState({
    totalEvents: 0,
    violationScore: 0,
    warningThreshold: 3,
    forceThreshold: 5,
    status: 'normal',
    remainingWarns: 3,
    eventBreakdown: {},
  });

  const [showWarning, setShowWarning] = useState(false);
  const [warningMessage, setWarningMessage] = useState('');

  const eventQueueRef = useRef([]);
  const batchTimerRef = useRef(null);
  const lastEventTimeRef = useRef({});
  const isMonitoringRef = useRef(false);
  const visibilityRef = useRef('visible');

  const flushQueue = useCallback(async () => {
    if (eventQueueRef.current.length === 0) return;

    const events = [...eventQueueRef.current];
    eventQueueRef.current = [];

    try {
      const result = await reportProctorEvents(examId, events, token);
      setStatus({
        totalEvents: status.totalEvents + events.length,
        violationScore: result.violationScore,
        warningThreshold: result.warningThreshold,
        forceThreshold: result.forceThreshold,
        status: result.status,
        remainingWarns: result.remainingWarns,
        eventBreakdown: status.eventBreakdown,
      });

      if (result.status === 'force_submitted') {
        onForceSubmit?.();
      } else if (result.status === 'suspicious' || result.status === 'warning') {
        if (status.status === 'normal') {
          setWarningMessage(`检测到违规行为，剩余警告次数：${result.remainingWarns}`);
          setShowWarning(true);
          setTimeout(() => setShowWarning(false), 3000);
        }
      }

      onStatusChange?.(result);
    } catch (error) {
      console.error('Failed to report proctor events:', error);
      eventQueueRef.current = [...events, ...eventQueueRef.current];
    }
  }, [examId, token, status, onStatusChange, onForceSubmit]);

  const queueEvent = useCallback((eventType, extraInfo = '') => {
    if (!enabled || !isMonitoringRef.current) return;

    const now = Date.now();
    const lastTime = lastEventTimeRef.current[eventType] || 0;

    if (now - lastTime < DEBOUNCE_MS) {
      return;
    }
    lastEventTimeRef.current[eventType] = now;

    eventQueueRef.current.push({
      eventType,
      eventTime: now,
      extraInfo,
    });

    setStatus((prev) => ({
      ...prev,
      totalEvents: prev.totalEvents + 1,
      eventBreakdown: {
        ...prev.eventBreakdown,
        [eventType]: (prev.eventBreakdown[eventType] || 0) + 1,
      },
    }));

    if (eventQueueRef.current.length >= MAX_BATCH_SIZE) {
      flushQueue();
    } else if (!batchTimerRef.current) {
      batchTimerRef.current = setTimeout(flushQueue, BATCH_INTERVAL);
    }
  }, [enabled, flushQueue]);

  const handleVisibilityChange = useCallback(() => {
    const isVisible = document.visibilityState === 'visible';
    const prevVisible = visibilityRef.current === 'visible';
    visibilityRef.current = document.visibilityState;

    if (prevVisible && !isVisible) {
      queueEvent(EVENT_TYPE_TAB_SWITCH, 'visibilitychange hidden');
    }
  }, [queueEvent]);

  const handleBlur = useCallback(() => {
    queueEvent(EVENT_TYPE_BLUR, 'window blur');
  }, [queueEvent]);

  const handleCopy = useCallback((e) => {
    const selectedText = window.getSelection()?.toString() || '';
    queueEvent(EVENT_TYPE_COPY, `copied ${selectedText.length} chars`);
  }, [queueEvent]);

  const handlePaste = useCallback((e) => {
    const pastedText = e.clipboardData?.getData('text') || '';
    queueEvent(EVENT_TYPE_PASTE, `pasted ${pastedText.length} chars`);
  }, [queueEvent]);

  const handleFullscreenChange = useCallback(() => {
    if (!document.fullscreenElement) {
      queueEvent(EVENT_TYPE_FULLSCREEN_EXIT, 'exited fullscreen');
    }
  }, [queueEvent]);

  const loadInitialStatus = useCallback(async () => {
    try {
      const data = await getProctorStudentStatus(examId, token);
      setStatus(data);
    } catch (error) {
      console.error('Failed to load proctor status:', error);
    }
  }, [examId, token]);

  const startMonitoring = useCallback(() => {
    if (!enabled || isMonitoringRef.current) return;

    isMonitoringRef.current = true;
    visibilityRef.current = document.visibilityState;

    document.addEventListener('visibilitychange', handleVisibilityChange);
    window.addEventListener('blur', handleBlur);
    document.addEventListener('copy', handleCopy);
    document.addEventListener('paste', handlePaste);
    document.addEventListener('fullscreenchange', handleFullscreenChange);

    loadInitialStatus();
  }, [enabled, handleVisibilityChange, handleBlur, handleCopy, handlePaste, handleFullscreenChange, loadInitialStatus]);

  const stopMonitoring = useCallback(() => {
    if (!isMonitoringRef.current) return;

    isMonitoringRef.current = false;

    document.removeEventListener('visibilitychange', handleVisibilityChange);
    window.removeEventListener('blur', handleBlur);
    document.removeEventListener('copy', handleCopy);
    document.removeEventListener('paste', handlePaste);
    document.removeEventListener('fullscreenchange', handleFullscreenChange);

    if (batchTimerRef.current) {
      clearTimeout(batchTimerRef.current);
      batchTimerRef.current = null;
    }

    if (eventQueueRef.current.length > 0) {
      flushQueue();
    }
  }, [handleVisibilityChange, handleBlur, handleCopy, handlePaste, handleFullscreenChange, flushQueue]);

  const requestFullscreen = useCallback(async (element) => {
    try {
      if (element?.requestFullscreen) {
        await element.requestFullscreen();
      }
    } catch (error) {
      console.warn('Failed to enter fullscreen:', error);
    }
  }, []);

  const reportReconnect = useCallback(() => {
    queueEvent(EVENT_TYPE_RECONNECT, 'page reloaded or reconnected');
  }, [queueEvent]);

  useEffect(() => {
    return () => {
      stopMonitoring();
    };
  }, [stopMonitoring]);

  return {
    status,
    showWarning,
    warningMessage,
    setShowWarning,
    startMonitoring,
    stopMonitoring,
    queueEvent,
    requestFullscreen,
    reportReconnect,
    EVENT_LABELS,
  };
}

export {
  EVENT_TYPE_TAB_SWITCH,
  EVENT_TYPE_BLUR,
  EVENT_TYPE_COPY,
  EVENT_TYPE_PASTE,
  EVENT_TYPE_FULLSCREEN_EXIT,
  EVENT_TYPE_RECONNECT,
  EVENT_LABELS,
};
