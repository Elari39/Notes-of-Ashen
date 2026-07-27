import mermaid from 'mermaid';
import { useCallback, useEffect, useId, useRef, useState, type PointerEvent } from 'react';
import { translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';
import { hasMermaidRenderError } from '../utils/mermaidRender';
import MarkdownCodeToolbar from './MarkdownCodeToolbar';
import { codeBlockStyle, codeTagStyle } from './MarkdownCodeStyles';

export type MermaidViewMode = 'diagram' | 'code';

type MermaidCodeBlockProps = {
  code: string;
};

type DiagramPoint = {
  x: number;
  y: number;
};

type DragState = DiagramPoint & {
  pointerId: number;
  originX: number;
  originY: number;
};

const MIN_ZOOM = 0.25;
const MAX_ZOOM = 4;
const ZOOM_FACTOR = 1.2;

const getMermaidTheme = (effectiveTheme: 'light' | 'dark') => {
  const colors = effectiveTheme === 'dark'
    ? {
        background: '#121210',
        primary: '#302a26',
        secondary: '#252320',
        tertiary: '#1f1e1b',
        text: '#f5f0e8',
        border: '#d68a70',
        line: '#c1b8ad',
      }
    : {
        background: '#181715',
        primary: '#302d29',
        secondary: '#252320',
        tertiary: '#1f1e1b',
        text: '#f5f0e8',
        border: '#cc785c',
        line: '#c1b8ad',
      };

  return {
    theme: 'base' as const,
    themeVariables: {
      background: colors.background,
      mainBkg: colors.background,
      primaryColor: colors.primary,
      primaryTextColor: colors.text,
      primaryBorderColor: colors.border,
      secondaryColor: colors.secondary,
      secondaryTextColor: colors.text,
      secondaryBorderColor: colors.border,
      tertiaryColor: colors.tertiary,
      tertiaryTextColor: colors.text,
      tertiaryBorderColor: colors.border,
      textColor: colors.text,
      nodeTextColor: colors.text,
      lineColor: colors.line,
      defaultLinkColor: colors.line,
      clusterBkg: colors.secondary,
      clusterBorder: colors.border,
      actorBkg: colors.primary,
      actorBorder: colors.border,
      actorTextColor: colors.text,
      actorLineColor: colors.border,
      signalColor: colors.line,
      signalTextColor: colors.text,
      labelBoxBkgColor: colors.secondary,
      labelBoxBorderColor: colors.border,
      labelTextColor: colors.text,
      loopTextColor: colors.text,
      noteBkgColor: colors.secondary,
      noteBorderColor: colors.border,
      noteTextColor: colors.text,
      edgeLabelBackground: colors.secondary,
      sequenceNumberColor: colors.text,
      fontFamily: 'Inter, "Noto Sans SC", system-ui, sans-serif',
    },
  };
};

const clampZoom = (value: number) => Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, value));

const getSvgSize = (svg: SVGSVGElement): DiagramPoint => {
  const viewBox = svg.viewBox.baseVal;
  if (viewBox.width > 0 && viewBox.height > 0) {
    return { x: viewBox.width, y: viewBox.height };
  }

  const width = Number.parseFloat(svg.getAttribute('width') || '');
  const height = Number.parseFloat(svg.getAttribute('height') || '');
  return {
    x: Number.isFinite(width) && width > 0 ? width : 1,
    y: Number.isFinite(height) && height > 0 ? height : 1,
  };
};

const MermaidCodeBlock: React.FC<MermaidCodeBlockProps> = ({ code }) => {
  const language = usePreferenceStore((state) => state.language);
  const effectiveTheme = usePreferenceStore((state) => state.effectiveTheme);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const renderId = `mermaid-diagram-${useId().replace(/:/g, '')}`;
  const diagramRef = useRef<HTMLDivElement>(null);
  const stageRef = useRef<HTMLDivElement>(null);
  const svgRef = useRef<SVGSVGElement | null>(null);
  const bindFunctionsRef = useRef<((element: HTMLElement) => void) | undefined>(undefined);
  const zoomRef = useRef(1);
  const panRef = useRef<DiagramPoint>({ x: 0, y: 0 });
  const dragRef = useRef<DragState | null>(null);
  const [mode, setMode] = useState<MermaidViewMode>('diagram');
  const [isRendering, setIsRendering] = useState(true);
  const [renderFailed, setRenderFailed] = useState(false);
  const [svgMarkup, setSvgMarkup] = useState('');
  const [zoom, setZoom] = useState(1);
  const [pan, setPan] = useState<DiagramPoint>({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);

  const applyView = useCallback((nextZoom: number, nextPan: DiagramPoint) => {
    zoomRef.current = nextZoom;
    panRef.current = nextPan;
    setZoom(nextZoom);
    setPan(nextPan);
  }, []);

  const fitDiagram = useCallback(() => {
    const panel = diagramRef.current;
    const svg = svgRef.current;
    if (!panel || !svg) {
      return;
    }

    const diagramSize = getSvgSize(svg);
    const availableWidth = Math.max(panel.clientWidth - 32, 1);
    const availableHeight = Math.max(panel.clientHeight - 32, 1);
    const nextZoom = clampZoom(Math.min(
      availableWidth / diagramSize.x,
      availableHeight / diagramSize.y,
    ));
    const nextPan = {
      x: (panel.clientWidth - diagramSize.x * nextZoom) / 2,
      y: (panel.clientHeight - diagramSize.y * nextZoom) / 2,
    };

    applyView(nextZoom, nextPan);
  }, [applyView]);

  const zoomAround = useCallback((nextZoom: number, focalPoint: DiagramPoint) => {
    const currentZoom = zoomRef.current;
    const currentPan = panRef.current;
    const scale = nextZoom / currentZoom;
    applyView(nextZoom, {
      x: focalPoint.x - (focalPoint.x - currentPan.x) * scale,
      y: focalPoint.y - (focalPoint.y - currentPan.y) * scale,
    });
  }, [applyView]);

  const changeZoom = useCallback((factor: number) => {
    const panel = diagramRef.current;
    if (!panel) {
      return;
    }

    zoomAround(
      clampZoom(zoomRef.current * factor),
      { x: panel.clientWidth / 2, y: panel.clientHeight / 2 },
    );
  }, [zoomAround]);

  const handleWheel = (event: React.WheelEvent<HTMLDivElement>) => {
    event.preventDefault();
    const panel = event.currentTarget;
    const rect = panel.getBoundingClientRect();
    const nextZoom = clampZoom(zoomRef.current * (event.deltaY < 0 ? ZOOM_FACTOR : 1 / ZOOM_FACTOR));
    zoomAround(nextZoom, {
      x: event.clientX - rect.left,
      y: event.clientY - rect.top,
    });
  };

  const handlePointerDown = (event: PointerEvent<HTMLDivElement>) => {
    if (event.target instanceof Element && event.target.closest('.article-mermaid-controls')) {
      return;
    }

    event.preventDefault();
    event.currentTarget.setPointerCapture(event.pointerId);
    dragRef.current = {
      pointerId: event.pointerId,
      x: event.clientX,
      y: event.clientY,
      originX: panRef.current.x,
      originY: panRef.current.y,
    };
    setIsDragging(true);
  };

  const handlePointerMove = (event: PointerEvent<HTMLDivElement>) => {
    const drag = dragRef.current;
    if (!drag || drag.pointerId !== event.pointerId) {
      return;
    }

    applyView(zoomRef.current, {
      x: drag.originX + event.clientX - drag.x,
      y: drag.originY + event.clientY - drag.y,
    });
  };

  const stopDragging = (event: PointerEvent<HTMLDivElement>) => {
    if (dragRef.current?.pointerId !== event.pointerId) {
      return;
    }

    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
    dragRef.current = null;
    setIsDragging(false);
  };

  useEffect(() => {
    let active = true;
    setMode('diagram');
    setIsRendering(true);
    setRenderFailed(false);
    bindFunctionsRef.current = undefined;
    setSvgMarkup('');

    const renderDiagram = async () => {
      try {
        mermaid.initialize({
          startOnLoad: false,
          securityLevel: 'strict',
          ...getMermaidTheme(effectiveTheme),
        });
        const result = await mermaid.render(renderId, code);
        if (!result.svg.trim()) {
          throw new Error('Mermaid returned an empty diagram');
        }
        if (hasMermaidRenderError(result.svg)) {
          throw new Error('Mermaid returned an error diagram');
        }
        if (!active) {
          return;
        }

        bindFunctionsRef.current = result.bindFunctions;
        setSvgMarkup(result.svg);
        setIsRendering(false);
      } catch {
        if (active) {
          setMode('code');
          setIsRendering(false);
          setRenderFailed(true);
          setSvgMarkup('');
        }
      }
    };

    void renderDiagram();
    return () => {
      active = false;
      bindFunctionsRef.current = undefined;
    };
  }, [code, effectiveTheme, renderId]);

  useEffect(() => {
    const diagramNode = diagramRef.current;
    const stage = stageRef.current;
    if (mode !== 'diagram' || !diagramNode || !stage) {
      return;
    }

    stage.innerHTML = svgMarkup;
    const svg = stage.querySelector('svg');
    if (!svg) {
      return () => {
        stage.innerHTML = '';
      };
    }

    svgRef.current = svg;
    const diagramSize = getSvgSize(svg);
    svg.style.width = `${diagramSize.x}px`;
    svg.style.height = `${diagramSize.y}px`;
    if (svgMarkup) {
      bindFunctionsRef.current?.(stage);
    }
    fitDiagram();

    return () => {
      stage.innerHTML = '';
      svgRef.current = null;
    };
  }, [fitDiagram, mode, svgMarkup]);

  useEffect(() => {
    const svg = svgRef.current;
    if (!svg) {
      return;
    }

    svg.style.transform = `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`;
  }, [pan, zoom]);

  const modeActions = renderFailed ? null : (
    <div className="article-code-mode" role="group" aria-label={t('markdownCode.mermaidMode')}>
      <button
        type="button"
        className={mode === 'diagram' ? 'is-active' : ''}
        aria-pressed={mode === 'diagram'}
        onClick={() => setMode('diagram')}
      >
        {t('markdownCode.mermaidDiagram')}
      </button>
      <button
        type="button"
        className={mode === 'code' ? 'is-active' : ''}
        aria-pressed={mode === 'code'}
        onClick={() => setMode('code')}
      >
        {t('markdownCode.mermaidCode')}
      </button>
    </div>
  );

  return (
    <div className="article-code-shell article-mermaid-shell">
      <MarkdownCodeToolbar code={code} language="mermaid" actions={modeActions} />
      {mode === 'diagram' && !renderFailed ? (
        <div
          ref={diagramRef}
          className={`article-mermaid-panel${isDragging ? ' is-dragging' : ''}`}
          role="region"
          aria-label={t('markdownCode.mermaidDiagramLabel')}
          aria-busy={isRendering}
          onWheel={handleWheel}
          onPointerDown={handlePointerDown}
          onPointerMove={handlePointerMove}
          onPointerUp={stopDragging}
          onPointerCancel={stopDragging}
          onDoubleClick={fitDiagram}
        >
          <div ref={stageRef} className="article-mermaid-stage" aria-hidden="true" />
          {!isRendering && (
            <div className="article-mermaid-controls" role="group" aria-label={t('markdownCode.mermaidZoom')}>
              <button
                type="button"
                aria-label={t('markdownCode.mermaidZoomOut')}
                title={t('markdownCode.mermaidZoomOut')}
                disabled={zoom <= MIN_ZOOM}
                onClick={() => changeZoom(1 / ZOOM_FACTOR)}
              >
                −
              </button>
              <output className="article-mermaid-zoom-value" aria-live="polite">{Math.round(zoom * 100)}%</output>
              <button
                type="button"
                aria-label={t('markdownCode.mermaidZoomIn')}
                title={t('markdownCode.mermaidZoomIn')}
                disabled={zoom >= MAX_ZOOM}
                onClick={() => changeZoom(ZOOM_FACTOR)}
              >
                +
              </button>
              <button
                type="button"
                aria-label={t('markdownCode.mermaidResetZoom')}
                title={t('markdownCode.mermaidResetZoom')}
                onClick={fitDiagram}
              >
                ↺
              </button>
            </div>
          )}
        </div>
      ) : (
        <pre className="article-code-block" style={codeBlockStyle}>
          <code style={codeTagStyle}>{code}</code>
        </pre>
      )}
    </div>
  );
};

export default MermaidCodeBlock;
