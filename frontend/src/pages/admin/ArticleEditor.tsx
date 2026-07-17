import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useLocation, useParams, useNavigate } from 'react-router-dom';
import { assistArticle, getArticlePreview, createArticle, updateArticle } from '../../api/article';
import { createCategory, getCategories } from '../../api/category';
import { createTag, getTags } from '../../api/tag';
import type { Article, ArticleStatus, Category, Tag, MediaAsset } from '../../types';
import type { AIAssistAction } from '../../types/api';
import InlineNotice from '../../components/InlineNotice';
import MarkdownRenderer from '../../components/MarkdownRenderer';
import PagePendingState from '../../components/RoutePending';
import Button from '../../components/ui/Button';
import { getErrorMessage } from '../../utils/error';
import { generateSlug } from '../../utils/slug';
import { isValidCoverUrl } from '../../utils/cover';
import { notifyFromError } from '../../utils/notify';
import { notifyArticleCacheInvalid } from '../../utils/pwa';
import { formatText, getArticleStatusLabel, translate } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';
import { useSubmit } from '../../hooks/useSubmit';
import { useDebouncedValue } from '../../hooks/useDebouncedValue';
import {
  canOperateArticleEditor,
  executeArticleEditorOperation,
  resolveArticleEditorAccess,
  type ArticleBaselineStatus,
} from './editorAccessPolicy';
import {
  buildArticleCompletionPatch,
  readAITaxonomySuggestions,
  type AITaxonomySuggestions,
} from './articleAICompletionPolicy';
import { safeLocalStorage } from '../../utils/storage';
import MediaPicker from '../../components/admin/MediaPicker';
import { getReadingStats } from '../../utils/readingStats';
import { MAX_ARTICLE_CONTENT_BYTES, MAX_TEXT_FIELD_BYTES, utf8ByteLength } from '../../utils/utf8';

type TaxonomyOption = {
  id: number;
  name: string;
  slug: string;
};

type TaxonomyComboboxProps<T extends TaxonomyOption> = {
  label: string;
  placeholder: string;
  items: T[];
  selectedIds: number[];
  multiple?: boolean;
  error?: string;
  creating?: boolean;
  createLabel: (value: string) => string;
  noMatchesLabel: string;
  clearLabel: string;
  creatingLabel: string;
  onSelect: (item: T) => void;
  onRemove: (id: number) => void;
  onCreate: (name: string) => Promise<boolean>;
};

const normalizeSearch = (value: string) => value.trim().toLowerCase();

const TaxonomyCombobox = <T extends TaxonomyOption>({
  label,
  placeholder,
  items,
  selectedIds,
  multiple = false,
  error,
  creating = false,
  createLabel,
  noMatchesLabel,
  clearLabel,
  creatingLabel,
  onSelect,
  onRemove,
  onCreate,
}: TaxonomyComboboxProps<T>) => {
  const [query, setQuery] = useState('');
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const activeOptionRef = useRef<HTMLButtonElement | null>(null);
  const listboxId = React.useId();
  const inputId = React.useId();
  const optionId = (index: number) => `${listboxId}-option-${index}`;
  const selectedIdSet = useMemo(() => new Set(selectedIds), [selectedIds]);
  const selectedItems = useMemo(
    () => items.filter(item => selectedIdSet.has(item.id)),
    [items, selectedIdSet],
  );
  const selectedCategory = !multiple ? selectedItems[0] : undefined;
  const selectedCategoryName = selectedCategory?.name;
  const trimmedQuery = query.trim();
  const normalizedQuery = normalizeSearch(query);

  useEffect(() => {
    if (!multiple && selectedCategoryName) {
      setQuery(selectedCategoryName);
    }
  }, [multiple, selectedCategoryName]);

  const availableItems = useMemo(
    () => multiple ? items.filter(item => !selectedIdSet.has(item.id)) : items,
    [items, multiple, selectedIdSet],
  );
  const filteredItems = useMemo(() => {
    if (!normalizedQuery) {
      return availableItems;
    }
    return availableItems.filter(item => {
      const name = item.name.toLowerCase();
      const slug = item.slug.toLowerCase();
      return name.includes(normalizedQuery) || slug.includes(normalizedQuery);
    });
  }, [availableItems, normalizedQuery]);
  const exactNameItem = items.find(item => item.name.trim().toLowerCase() === normalizedQuery);
  const hasSameName = Boolean(exactNameItem);
  const canCreate = trimmedQuery.length > 0 && !hasSameName;

  const handleSelect = (item: T) => {
    onSelect(item);
    setQuery(multiple ? '' : item.name);
    setOpen(false);
    setActiveIndex(-1);
  };

  const handleCreate = async () => {
    if (!canCreate || creating) {
      return;
    }
    const created = await onCreate(trimmedQuery);
    if (created) {
      setQuery('');
      setOpen(false);
      setActiveIndex(-1);
    }
  };

  const handleInputChange = (value: string) => {
    if (!multiple && selectedCategory) {
      onRemove(selectedCategory.id);
    }
    setQuery(value);
    setOpen(true);
    setActiveIndex(-1);
  };

  // 下拉可选条目总数：已过滤项 + 可创建项（创建项固定排在最后）。
  const optionCount = filteredItems.length + (canCreate ? 1 : 0);

  const handleKeyDown = async (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (!open) {
        setOpen(true);
      }
      if (optionCount > 0) {
        setActiveIndex(prev => (prev + 1) % optionCount);
      }
      return;
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (!open) {
        setOpen(true);
      }
      if (optionCount > 0) {
        setActiveIndex(prev => (prev <= 0 ? optionCount - 1 : prev - 1));
      }
      return;
    }
    if (e.key === 'Home') {
      if (open && optionCount > 0) {
        e.preventDefault();
        setActiveIndex(0);
      }
      return;
    }
    if (e.key === 'End') {
      if (open && optionCount > 0) {
        e.preventDefault();
        setActiveIndex(optionCount - 1);
      }
      return;
    }
    if (e.key === 'Enter') {
      e.preventDefault();
      if (activeIndex >= 0 && activeIndex < filteredItems.length) {
        handleSelect(filteredItems[activeIndex]);
        return;
      }
      if (activeIndex === filteredItems.length && canCreate) {
        await handleCreate();
        return;
      }
      if (exactNameItem) {
        handleSelect(exactNameItem);
        return;
      }
      await handleCreate();
    }
    if (e.key === 'Escape') {
      setOpen(false);
      setActiveIndex(-1);
    }
    if (multiple && e.key === 'Backspace' && query === '' && selectedIds.length > 0) {
      onRemove(selectedIds[selectedIds.length - 1]);
    }
  };

  // 方向键移动活跃项时，把它滚动进下拉可视区。
  useEffect(() => {
    if (activeIndex < 0 || !open) {
      return;
    }
    activeOptionRef.current?.scrollIntoView({ block: 'nearest' });
  }, [activeIndex, open]);

  // 选项列表变化（输入过滤）时重置活跃项，避免越界。
  useEffect(() => {
    setActiveIndex(-1);
  }, [normalizedQuery]);

  return (
    <div
      className="relative space-y-2"
      onBlur={e => {
        const nextTarget = e.relatedTarget as Node | null;
        if (!nextTarget || !e.currentTarget.contains(nextTarget)) {
          setOpen(false);
        }
      }}
    >
      <div className="flex items-center gap-2 border-b border-mountain-grey py-1 focus-within:border-ochre">
        <span className="text-ink-light text-sm whitespace-nowrap">{label}</span>
        {multiple && selectedItems.map(item => (
          <button
            key={item.id}
            type="button"
            onClick={() => onRemove(item.id)}
            className="shrink-0 px-2 py-1 text-xs border border-ochre text-ochre rounded-sm whitespace-nowrap hover:bg-mountain-grey hover:bg-opacity-20 transition-colors"
            title={clearLabel}
          >
            {item.name} <span aria-hidden="true">&times;</span>
          </button>
        ))}
        <input
          id={inputId}
          type="text"
          role="combobox"
          aria-expanded={open && (filteredItems.length > 0 || canCreate)}
          aria-controls={listboxId}
          aria-autocomplete="list"
          aria-activedescendant={activeIndex >= 0 ? optionId(activeIndex) : undefined}
          aria-label={label}
          value={query}
          placeholder={multiple || !selectedCategory ? placeholder : ''}
          onFocus={() => setOpen(true)}
          onChange={e => handleInputChange(e.target.value)}
          onKeyDown={handleKeyDown}
          className="min-w-[8rem] flex-1 bg-transparent py-2 text-ink focus:outline-none"
        />
        {!multiple && selectedCategory && (
          <button
            type="button"
            onClick={() => {
              onRemove(selectedCategory.id);
              setQuery('');
              setOpen(false);
            }}
            className="shrink-0 px-2 text-ink-light hover:text-ochre transition-colors"
            title={clearLabel}
          >
            <span aria-hidden="true">&times;</span>
          </button>
        )}
      </div>

      <InlineNotice message={error} />

      {open && (filteredItems.length > 0 || canCreate) && (
        <div
          id={listboxId}
          role="listbox"
          aria-label={label}
          className="absolute left-0 right-0 top-full z-20 mt-1 max-h-56 overflow-y-auto border border-mountain-grey bg-paper shadow-sm"
        >
          {filteredItems.map((item, index) => (
            <button
              key={item.id}
              ref={activeIndex === index ? activeOptionRef : undefined}
              type="button"
              role="option"
              id={optionId(index)}
              aria-selected={activeIndex === index}
              onMouseEnter={() => setActiveIndex(index)}
              onClick={() => handleSelect(item)}
              className={`block w-full px-3 py-2 text-left text-sm text-ink hover:bg-mountain-grey hover:bg-opacity-20 ${activeIndex === index ? 'bg-mountain-grey bg-opacity-20' : ''}`}
            >
              <span className="font-bold">{item.name}</span>
              <span className="ml-2 text-xs text-ink-light">{item.slug}</span>
            </button>
          ))}
          {canCreate && (
            <button
              ref={activeIndex === filteredItems.length ? activeOptionRef : undefined}
              type="button"
              role="option"
              id={optionId(filteredItems.length)}
              aria-selected={activeIndex === filteredItems.length}
              onMouseEnter={() => setActiveIndex(filteredItems.length)}
              onClick={handleCreate}
              disabled={creating}
              className={`block w-full px-3 py-2 text-left text-sm text-ochre hover:bg-mountain-grey hover:bg-opacity-20 disabled:opacity-50 disabled:cursor-not-allowed ${activeIndex === filteredItems.length ? 'bg-mountain-grey bg-opacity-20' : ''}`}
            >
              {creating ? creatingLabel : createLabel(trimmedQuery)}
            </button>
          )}
        </div>
      )}

      {open && filteredItems.length === 0 && !canCreate && (
        <div className="absolute left-0 right-0 top-full z-20 mt-1 border border-mountain-grey bg-paper px-3 py-2 text-sm text-ink-light shadow-sm">
          {noMatchesLabel}
        </div>
      )}
    </div>
  );
};

const ArticleEditor: React.FC = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const location = useLocation();
  const language = usePreferenceStore((state) => state.language);
  const isEdit = id && id !== 'new';
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const languageRef = useRef(language);
  languageRef.current = language;

  const [title, setTitle] = useState('');
  const [slug, setSlug] = useState('');
  const [summary, setSummary] = useState('');
  const [generateSummaryOnSave, setGenerateSummaryOnSave] = useState(true);
  const [content, setContent] = useState('');
  const [coverUrl, setCoverUrl] = useState('');
  const [mediaPickerMode, setMediaPickerMode] = useState<'cover' | 'content' | null>(null);
  const [status, setStatus] = useState<ArticleStatus>('draft');
  const [scheduledAt, setScheduledAt] = useState('');
  const [isPinned, setIsPinned] = useState(false);
  const [displayPriority, setDisplayPriority] = useState(0);
  const [seoTitle, setSeoTitle] = useState('');
  const [seoDescription, setSeoDescription] = useState('');
  const [seoKeywords, setSeoKeywords] = useState('');
  const [categoryId, setCategoryId] = useState<number | ''>('');
  const [tagIds, setTagIds] = useState<number[]>([]);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const liveReadingStats = useMemo(() => getReadingStats(content), [content]);
  const contentByteLength = useMemo(() => utf8ByteLength(content), [content]);
  const summaryByteLength = useMemo(() => utf8ByteLength(summary.trim()), [summary]);
  const [isEditorReady, setIsEditorReady] = useState(false);
  const [draftRecovery, setDraftRecovery] = useState<EditorDraft | null>(null);
  const draftRecoveryRef = useRef<EditorDraft | null>(null);
  draftRecoveryRef.current = draftRecovery;
  const editorBaselineRef = useRef<EditorDraft | null>(null);
  const [articleBaselineStatus, setArticleBaselineStatus] = useState<ArticleBaselineStatus>(isEdit ? 'loading' : 'ready');
  const [articleBaselineId, setArticleBaselineId] = useState<string | undefined>();
  const [articleBaselineError, setArticleBaselineError] = useState('');
  const [baselineRetryKey, setBaselineRetryKey] = useState(0);
  // discard 草稿后置位，跳过下一次自动保存写入，避免 effect 基于当前字段又把刚丢弃的草稿写回。
  const skipAutosaveOnceRef = useRef(false);

  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [error, setError] = useState('');
  const [categoryError, setCategoryError] = useState('');
  const [categorySubmitting, setCategorySubmitting] = useState(false);
  const [tagError, setTagError] = useState('');
  const [tagSubmitting, setTagSubmitting] = useState(false);
  const [aiMenuOpen, setAiMenuOpen] = useState(false);
  const [aiAction, setAiAction] = useState<AIAssistAction | null>(null);
  const [aiNotice, setAiNotice] = useState('');
  const [aiTaxonomySuggestions, setAiTaxonomySuggestions] = useState<AITaxonomySuggestions | null>(null);
  const [aiDraft, setAiDraft] = useState<{
    action: AIAssistAction;
    revisedContent: string;
    suggestions: string[];
    rangeStart: number;
    rangeEnd: number;
    sourceContent: string;
  } | null>(null);
  const aiActionLabelKey = (action: AIAssistAction) => translate(language, `aiAction.${action}`);
  const scheduledPublishHint = (() => {
    if (!scheduledAt || status !== 'published') {
      return '';
    }
    const scheduledTime = new Date(scheduledAt).getTime();
    if (Number.isNaN(scheduledTime) || scheduledTime <= Date.now()) {
      return '';
    }
    return t('articleEditor.scheduledPublishHint');
  })();
  // 预览防抖：每次按键只更新 content，预览用延迟值，避免全文重解析。
  const debouncedPreviewContent = useDebouncedValue(content, 250);
  const currentArticleBaselineStatus: ArticleBaselineStatus = isEdit && articleBaselineId !== id
    ? 'loading'
    : articleBaselineStatus;
  const articleEditorAccess = resolveArticleEditorAccess(Boolean(isEdit), currentArticleBaselineStatus);

  useEffect(() => {
    const state = location.state as { aiNotice?: string } | null;
    if (!state?.aiNotice) {
      return;
    }
    setAiNotice(state.aiNotice);
    navigate(location.pathname, { replace: true, state: null });
  }, [location.pathname, location.state, navigate]);

  useEffect(() => {
    let active = true;
    setIsEditorReady(false);
    setDraftRecovery(null);
    setAiDraft(null);
    setAiTaxonomySuggestions(null);
    setAiMenuOpen(false);
    editorBaselineRef.current = null;
    setError('');
    const fetchDeps = async () => {
      try {
        const [catRes, tagRes] = await Promise.all([
          getCategories({ size: 100 }),
          getTags({ size: 100 }),
        ]);
        if (!active) {
          return;
        }
        setCategories(catRes.data.items || []);
        setTags(tagRes.data.items || []);
      } catch (e) {
        if (active) {
          setError(getErrorMessage(e, translate(languageRef.current, 'articleEditor.depsError')));
        }
      }
    };
    void fetchDeps();

    if (isEdit && id) {
      setArticleBaselineStatus('loading');
      setArticleBaselineId(id);
      setArticleBaselineError('');
      void getArticlePreview(id)
        .then(res => {
          if (!active) {
            return;
          }
          const article = res.data;
          const baseline = articleToEditorDraft(article);
          const draftKey = editorDraftKey(id);
          editorBaselineRef.current = baseline;
          applyEditorDraft(baseline, {
            setTitle,
            setSlug,
            setSummary,
            setGenerateSummaryOnSave,
            setContent,
            setCoverUrl,
            setStatus,
            setScheduledAt,
            setIsPinned,
            setDisplayPriority,
            setSeoTitle,
            setSeoDescription,
            setSeoKeywords,
            setCategoryId,
            setTagIds,
          });
          const localDraft = readEditorDraft(draftKey);
          if (shouldRecoverEditorDraft(localDraft, baseline, article.updatedAt)) {
            setDraftRecovery(localDraft);
          } else if (localDraft) {
            removeEditorDraft(draftKey);
          }
          setArticleBaselineStatus('ready');
          setIsEditorReady(true);
        })
        .catch(e => {
          if (!active) {
            return;
          }
          setArticleBaselineStatus('error');
          setArticleBaselineError(getErrorMessage(e, translate(languageRef.current, 'article.loadError')));
        });
    } else {
      setArticleBaselineStatus('ready');
      setArticleBaselineId(undefined);
      setArticleBaselineError('');
      const baseline = emptyEditorDraft();
      const draftKey = editorDraftKey('new');
      editorBaselineRef.current = baseline;
      applyEditorDraft(baseline, {
        setTitle,
        setSlug,
        setSummary,
        setGenerateSummaryOnSave,
        setContent,
        setCoverUrl,
        setStatus,
        setScheduledAt,
        setIsPinned,
        setDisplayPriority,
        setSeoTitle,
        setSeoDescription,
        setSeoKeywords,
        setCategoryId,
        setTagIds,
      });
      const localDraft = readEditorDraft(draftKey);
      if (shouldRecoverEditorDraft(localDraft, baseline)) {
        setDraftRecovery(localDraft);
      } else if (localDraft) {
        removeEditorDraft(draftKey);
      }
      setIsEditorReady(true);
    }
    return () => {
      active = false;
    };
  }, [baselineRetryKey, id, isEdit]);

  const retryArticleBaseline = () => {
    setArticleBaselineStatus('loading');
    setArticleBaselineError('');
    setBaselineRetryKey((value) => value + 1);
  };

  const handleSummaryChange = (value: string) => {
    setSummary(value);
    setGenerateSummaryOnSave(value.trim() === '');
  };

  useEffect(() => {
    if (!isEditorReady) {
      return;
    }
    // discard 后跳过本次自动保存，避免把刚丢弃的草稿又写回。
    if (skipAutosaveOnceRef.current) {
      skipAutosaveOnceRef.current = false;
      return;
    }
    const key = editorDraftKey(isEdit ? id : 'new');
    const baseline = editorBaselineRef.current;
    if (!baseline) {
      return;
    }
    const draft = currentEditorDraft({
      title,
      slug,
      summary,
      generateSummaryOnSave,
      content,
      coverUrl,
      status,
      scheduledAt,
      isPinned,
      displayPriority,
      seoTitle,
      seoDescription,
      seoKeywords,
      categoryId,
      tagIds,
    });
    if (
      editorDraftEquals(draft, baseline) ||
      (!hasMeaningfulEditorDraft(draft) && !hasMeaningfulEditorDraft(baseline))
    ) {
      if (!draftRecoveryRef.current) {
        removeEditorDraft(key);
      }
      return;
    }
    const timer = window.setTimeout(() => {
      writeEditorDraft(key, { ...draft, savedAt: Date.now() });
    }, 800);
    return () => window.clearTimeout(timer);
  }, [
    categoryId,
    content,
    coverUrl,
    displayPriority,
    generateSummaryOnSave,
    id,
    isEdit,
    isEditorReady,
    isPinned,
    scheduledAt,
    seoDescription,
    seoKeywords,
    seoTitle,
    slug,
    status,
    summary,
    tagIds,
    title,
  ]);

  const restoreLocalDraft = () => {
    if (!draftRecovery) {
      return;
    }
    applyEditorDraft(draftRecovery, {
      setTitle,
      setSlug,
      setSummary,
      setGenerateSummaryOnSave,
      setContent,
      setCoverUrl,
      setStatus,
      setScheduledAt,
      setIsPinned,
      setDisplayPriority,
      setSeoTitle,
      setSeoDescription,
      setSeoKeywords,
      setCategoryId,
      setTagIds,
    });
    setDraftRecovery(null);
    setAiNotice(t('articleEditor.draftRestored'));
  };

  const discardLocalDraft = () => {
    removeEditorDraft(editorDraftKey(isEdit ? id : 'new'));
    setDraftRecovery(null);
    // 丢弃后表单字段未变，effect 重跑会基于当前字段再次写入草稿；置位跳过下一次自动保存。
    skipAutosaveOnceRef.current = true;
    setAiNotice(t('articleEditor.draftDiscarded'));
  };

  const { submit: submitSave, submitting } = useSubmit({
    handler: async () => {
      if (!canOperateArticleEditor(Boolean(isEdit), currentArticleBaselineStatus)) {
        throw new Error('Article baseline is not loaded yet');
      }
      const trimmedCoverUrl = coverUrl.trim();
      let nextSummary = summary;
      let nextSeoDescription = seoDescription;
      let nextSeoKeywords = seoKeywords;
      if (generateSummaryOnSave && content.trim()) {
        try {
          const aiRes = await executeArticleEditorOperation(
            Boolean(isEdit),
            currentArticleBaselineStatus,
            () => assistArticle({ action: 'metadata', title, content }),
          );
          const generatedSummary = aiRes.data.summary?.trim();
          if (generatedSummary) {
            nextSummary = generatedSummary;
            setSummary(generatedSummary);
            setGenerateSummaryOnSave(false);
          }
          if (aiRes.data.seoDescription?.trim() && !seoDescription.trim()) {
            nextSeoDescription = aiRes.data.seoDescription.trim();
            setSeoDescription(nextSeoDescription);
          }
          if (aiRes.data.seoKeywords?.trim() && !seoKeywords.trim()) {
            nextSeoKeywords = aiRes.data.seoKeywords.trim();
            setSeoKeywords(nextSeoKeywords);
          }
        } catch (e) {
          // 自动摘要只是保存时的辅助能力，失败不应阻断文章保存。
          notifyFromError(e, 'toast.summaryFailed');
        }
      }

      const payload = {
        title, slug, summary: nextSummary, content, coverUrl: trimmedCoverUrl, status,
        scheduledAt: scheduledAt ? new Date(scheduledAt).toISOString() : undefined,
        isPinned,
        displayPriority,
        seoTitle: seoTitle.trim(),
        seoDescription: nextSeoDescription.trim(),
        seoKeywords: nextSeoKeywords.trim(),
        categoryId: categoryId === '' ? 0 : Number(categoryId),
        tagIds,
      };

      await executeArticleEditorOperation(
        Boolean(isEdit),
        currentArticleBaselineStatus,
        async () => {
          if (isEdit) {
            await updateArticle(id, payload);
            removeEditorDraft(editorDraftKey(id));
          } else {
            await createArticle(payload);
            removeEditorDraft(editorDraftKey('new'));
          }
        },
      );
    },
    successMessage: t('toast.saveSuccess'),
    errorFallback: t('articleEditor.saveError'),
    onSuccess: () => {
      notifyArticleCacheInvalid();
      navigate('/admin/articles');
    },
    onError: (err) => {
      // 保留 InlineNotice 既有展示位
      setError(err.message);
    },
  });

  const handleSave = () => {
    if (!canOperateArticleEditor(Boolean(isEdit), currentArticleBaselineStatus)) {
      return;
    }
    if (!title.trim()) {
      setError(t('articleEditor.titleRequired'));
      return;
    }
    if (!slug.trim()) {
      setError(t('articleEditor.slugRequired'));
      return;
    }
    if (contentByteLength > MAX_ARTICLE_CONTENT_BYTES) {
      setError(formatText(t('articleEditor.contentTooLarge'), { limit: MAX_ARTICLE_CONTENT_BYTES }));
      return;
    }
    if (summaryByteLength > MAX_TEXT_FIELD_BYTES) {
      setError(formatText(t('articleEditor.summaryTooLarge'), { limit: MAX_TEXT_FIELD_BYTES }));
      return;
    }
    const trimmedCoverUrl = coverUrl.trim();
    if (!isValidCoverUrl(trimmedCoverUrl)) {
      setError(t('articleEditor.coverUrlError'));
      return;
    }
    setError('');
    setAiNotice('');
    void submitSave();
  };

  const handleAIAssist = async (action: AIAssistAction) => {
    if (!canOperateArticleEditor(Boolean(isEdit), currentArticleBaselineStatus)) {
      return;
    }
    const target = action === 'metadata' || action === 'complete'
      ? { text: content, start: 0, end: content.length }
      : getActiveMarkdownRange(textareaRef.current, content);
    if (!target.text.trim()) {
      setError(t('articleEditor.aiContentRequired'));
      setAiMenuOpen(false);
      return;
    }
    setError('');
    setAiNotice('');
    setAiDraft(null);
    if (action === 'complete') {
      setAiTaxonomySuggestions(null);
    }
    setAiAction(action);
    setAiMenuOpen(false);
    try {
      const res = await executeArticleEditorOperation(
        Boolean(isEdit),
        currentArticleBaselineStatus,
        () => assistArticle({ action, title, content: target.text }),
      );
      const data = res.data;
      if (action === 'metadata') {
        if (data.summary) setSummary(data.summary);
        if (data.seoDescription) setSeoDescription(data.seoDescription);
        if (data.seoKeywords) setSeoKeywords(data.seoKeywords);
        setAiNotice(t('articleEditor.aiMetadataApplied'));
        return;
      }
      if (action === 'complete') {
        const { patch, appliedCount } = buildArticleCompletionPatch({
          title,
          slug,
          summary,
          seoTitle,
          seoDescription,
          seoKeywords,
        }, data);
        if (patch.title) setTitle(patch.title);
        if (patch.slug) setSlug(patch.slug);
        if (patch.summary) setSummary(patch.summary);
        if (patch.seoTitle) setSeoTitle(patch.seoTitle);
        if (patch.seoDescription) setSeoDescription(patch.seoDescription);
        if (patch.seoKeywords) setSeoKeywords(patch.seoKeywords);
        setAiTaxonomySuggestions(readAITaxonomySuggestions(data));
        setAiNotice(formatText(t('articleEditor.aiCompletionApplied'), { count: appliedCount }));
        return;
      }
      setAiDraft({
        action,
        revisedContent: data.revisedContent || '',
        suggestions: data.suggestions || [],
        rangeStart: target.start,
        rangeEnd: target.end,
        sourceContent: content,
      });
      setAiNotice(formatText(t('articleEditor.aiResultReady'), { action: aiActionLabelKey(action) }));
    } catch (e: unknown) {
      setError(getErrorMessage(e, t('articleEditor.aiError')));
    } finally {
      setAiAction(null);
    }
  };

  const applyAIContent = () => {
    if (!canOperateArticleEditor(Boolean(isEdit), currentArticleBaselineStatus)) {
      return;
    }
    const draft = aiDraft;
    if (!draft?.revisedContent) {
      return;
    }
    if (draft.sourceContent !== content) {
      setError(t('articleEditor.aiContentChanged'));
      setAiDraft(null);
      return;
    }
    setContent((current) => `${current.slice(0, draft.rangeStart)}${draft.revisedContent}${current.slice(draft.rangeEnd)}`);
    setAiDraft(null);
    setAiNotice(t('articleEditor.aiContentApplied'));
    window.setTimeout(() => {
      textareaRef.current?.focus();
      const cursor = draft.rangeStart + draft.revisedContent.length;
      textareaRef.current?.setSelectionRange(cursor, cursor);
    }, 0);
  };

  const handleCreateCategory = async (name: string) => {
    setCategoryError('');
    setCategorySubmitting(true);
    try {
      const res = await createCategory({
        name,
        slug: generateSlug(name),
        description: '',
      });
      const createdCategory = res.data;
      setCategories(prev => [createdCategory, ...prev.filter(category => category.id !== createdCategory.id)]);
      setCategoryId(createdCategory.id);
      return true;
    } catch (e: unknown) {
      setCategoryError(getErrorMessage(e, t('articleEditor.createCategoryError')));
      return false;
    } finally {
      setCategorySubmitting(false);
    }
  };

  const handleCreateTag = async (name: string) => {
    setTagError('');
    setTagSubmitting(true);
    try {
      const res = await createTag({
        name,
        slug: generateSlug(name),
        description: '',
      });
      const createdTag = res.data;
      setTags(prev => [createdTag, ...prev.filter(tag => tag.id !== createdTag.id)]);
      setTagIds(prev => prev.includes(createdTag.id) ? prev : [...prev, createdTag.id]);
      return true;
    } catch (e: unknown) {
      setTagError(getErrorMessage(e, t('articleEditor.createTagError')));
      return false;
    } finally {
      setTagSubmitting(false);
    }
  };

  const handleMediaSelect = (asset: MediaAsset) => {
    if (mediaPickerMode === 'cover') {
      setCoverUrl(asset.url);
      return;
    }
    const textarea = textareaRef.current;
    const start = textarea?.selectionStart ?? content.length;
    const end = textarea?.selectionEnd ?? start;
    const markdown = `![${asset.altText || asset.originalName.replace(/\.[^.]+$/, '')}](${asset.url})`;
    setContent(`${content.slice(0, start)}${markdown}${content.slice(end)}`);
    window.requestAnimationFrame(() => {
      textarea?.focus();
      textarea?.setSelectionRange(start + markdown.length, start + markdown.length);
    });
  };

  if (articleEditorAccess !== 'editor') {
    return (
      <div className="flex h-[80vh] flex-col">
        {articleEditorAccess === 'loading' ? (
          <PagePendingState variant="admin" label={t('common.loading')} />
        ) : (
          <InlineNotice
            message={articleBaselineError || t('article.loadError')}
            action={(
              <Button type="button" size="sm" onClick={retryArticleBaseline}>
                {t('common.retry')}
              </Button>
            )}
          />
        )}
      </div>
    );
  }

  return (
    <div className="flex flex-col h-[80vh]">
      <div className="flex justify-between items-center mb-6 pb-4 border-b border-mountain-grey">
        <h3 className="text-2xl font-bold text-ink tracking-widest">{isEdit ? t('articleEditor.editTitle') : t('articleEditor.newTitle')}</h3>
        <div className="flex items-center gap-3">
          <div className="relative">
            <button
              type="button"
              onClick={() => setAiMenuOpen(prev => !prev)}
              disabled={Boolean(aiAction)}
              className="border border-ochre px-4 py-2 text-sm tracking-widest text-ochre transition-colors hover:bg-ochre hover:text-paper disabled:cursor-not-allowed disabled:opacity-50"
            >
              {aiAction ? t('articleEditor.aiProcessing') : t('articleEditor.aiMagic')}
            </button>
            {aiMenuOpen && (
              <div className="absolute right-0 top-full z-30 mt-2 w-52 border border-mountain-grey bg-paper shadow-sm">
                <button type="button" onClick={() => handleAIAssist('complete')} className="block w-full px-3 py-2 text-left text-sm text-ink hover:bg-mountain-grey hover:bg-opacity-20">
                  {t('articleEditor.aiMetadata')}
                </button>
                <button type="button" onClick={() => handleAIAssist('proofread')} className="block w-full px-3 py-2 text-left text-sm text-ink hover:bg-mountain-grey hover:bg-opacity-20">
                  {t('articleEditor.aiProofread')}
                </button>
                <button type="button" onClick={() => handleAIAssist('polish')} className="block w-full px-3 py-2 text-left text-sm text-ink hover:bg-mountain-grey hover:bg-opacity-20">
                  {t('articleEditor.aiPolish')}
                </button>
                <button type="button" onClick={() => handleAIAssist('expand')} className="block w-full px-3 py-2 text-left text-sm text-ink hover:bg-mountain-grey hover:bg-opacity-20">
                  {t('articleEditor.aiExpand')}
                </button>
                <button type="button" onClick={() => handleAIAssist('shorten')} className="block w-full px-3 py-2 text-left text-sm text-ink hover:bg-mountain-grey hover:bg-opacity-20">
                  {t('articleEditor.aiShorten')}
                </button>
                <button type="button" onClick={() => handleAIAssist('translate')} className="block w-full px-3 py-2 text-left text-sm text-ink hover:bg-mountain-grey hover:bg-opacity-20">
                  {t('articleEditor.aiTranslate')}
                </button>
              </div>
            )}
          </div>
          <button onClick={handleSave} disabled={submitting} className="px-6 py-2 bg-ink text-paper tracking-widest text-sm hover:bg-opacity-80 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
            {submitting ? t('common.saving') : t('articleEditor.save')}
          </button>
        </div>
      </div>

      <InlineNotice message={error} className="mb-6" />
      <InlineNotice message={aiNotice} className="mb-6" />

      {aiTaxonomySuggestions && (
        <section className="mb-6 border border-ochre bg-[var(--paper-soft)] p-4" aria-label={t('articleEditor.aiTaxonomyTitle')}>
          <h4 className="text-sm font-bold tracking-widest text-ink">{t('articleEditor.aiTaxonomyTitle')}</h4>
          <p className="mt-1 text-xs leading-relaxed text-ink-light">{t('articleEditor.aiTaxonomyHint')}</p>
          <div className="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
            {aiTaxonomySuggestions.category && (
              <div>
                <p className="text-xs tracking-widest text-ink-light">{t('articleEditor.aiCategorySuggestion')}</p>
                <p className="mt-2 text-sm font-medium text-ink">{aiTaxonomySuggestions.category}</p>
              </div>
            )}
            {aiTaxonomySuggestions.tags.length > 0 && (
              <div>
                <p className="text-xs tracking-widest text-ink-light">{t('articleEditor.aiTagSuggestions')}</p>
                <div className="mt-2 flex flex-wrap gap-2">
                  {aiTaxonomySuggestions.tags.map((tag) => (
                    <span key={tag} className="border border-mountain-grey px-2 py-1 text-xs text-ink">{tag}</span>
                  ))}
                </div>
              </div>
            )}
          </div>
        </section>
      )}

      {draftRecovery && (
        <section className="mb-6 border border-ochre bg-[var(--paper-soft)] p-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h4 className="text-sm font-bold tracking-widest text-ink">{t('articleEditor.localDraftTitle')}</h4>
              <p className="mt-1 text-xs text-ink-light">
                {formatText(t('articleEditor.localDraftDesc'), { time: new Date(draftRecovery.savedAt).toLocaleString() })}
              </p>
            </div>
            <div className="flex gap-3">
              <button type="button" onClick={restoreLocalDraft} className="border border-ink px-3 py-1.5 text-sm text-ink transition-colors hover:bg-ink hover:text-paper">
                {t('articleEditor.restoreDraft')}
              </button>
              <button type="button" onClick={discardLocalDraft} className="border border-mountain-grey px-3 py-1.5 text-sm text-ink-light transition-colors hover:border-ochre hover:text-ochre">
                {t('articleEditor.discardDraft')}
              </button>
            </div>
          </div>
        </section>
      )}

      {aiDraft && (
        <section className="mb-6 border border-mountain-grey bg-[var(--paper-soft)] p-4">
          <div className="mb-3 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <h4 className="text-sm font-bold tracking-widest text-ink">
              {formatText(t('articleEditor.aiResultTitle'), { action: aiActionLabelKey(aiDraft.action) })}
            </h4>
            <div className="flex gap-3">
              <button type="button" onClick={applyAIContent} className="border border-ink px-3 py-1.5 text-sm text-ink transition-colors hover:bg-ink hover:text-paper">
                {t('articleEditor.aiApply')}
              </button>
              <button type="button" onClick={() => setAiDraft(null)} className="border border-mountain-grey px-3 py-1.5 text-sm text-ink-light transition-colors hover:border-ochre hover:text-ochre">
                {t('common.cancel')}
              </button>
            </div>
          </div>
          {aiDraft.suggestions.length > 0 && (
            <ul className="mb-3 list-disc space-y-1 pl-5 text-sm text-ink-light">
              {aiDraft.suggestions.map((item, index) => (
                <li key={`${item}-${index}`}>{item}</li>
              ))}
            </ul>
          )}
          <pre className="max-h-56 overflow-auto whitespace-pre-wrap border border-mountain-grey bg-paper p-3 text-sm leading-relaxed text-ink-light">
            {aiDraft.revisedContent}
          </pre>
        </section>
      )}

      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-6">
        <input
          type="text" placeholder={t('articleEditor.titlePlaceholder')} value={title} onChange={e => setTitle(e.target.value)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre text-lg font-bold"
        />
        <input
          type="text" placeholder={t('articleEditor.slugPlaceholder')} value={slug} onChange={e => setSlug(e.target.value)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        />
        <div className="flex items-end gap-2">
          <input
            type="text"
            placeholder={t('articleEditor.coverPlaceholder')}
            value={coverUrl}
            onChange={(event) => setCoverUrl(event.target.value)}
            className="min-w-0 flex-1 border-b border-mountain-grey bg-transparent py-2 text-ink focus:border-ochre focus:outline-none"
          />
          <Button type="button" size="sm" variant="subtle" onClick={() => setMediaPickerMode('cover')}>
            {t('media.choose')}
          </Button>
        </div>
        <select
          value={status} onChange={e => setStatus(e.target.value as ArticleStatus)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        >
          <option value="draft">{getArticleStatusLabel(language, 'draft')}</option>
          <option value="published">{getArticleStatusLabel(language, 'published')}</option>
          <option value="archived">{getArticleStatusLabel(language, 'archived')}</option>
        </select>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-6">
        <input
          type="datetime-local"
          value={scheduledAt}
          onChange={e => setScheduledAt(e.target.value)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        />
        {scheduledPublishHint && (
          <div className="text-xs leading-relaxed text-ochre md:col-span-3">
            {scheduledPublishHint}
          </div>
        )}
        <label className="flex items-center gap-3 border-b border-mountain-grey py-2 text-sm text-ink focus-within:border-ochre">
          <input
            type="checkbox"
            checked={isPinned}
            onChange={e => setIsPinned(e.target.checked)}
            className="h-4 w-4 accent-ochre"
          />
          <span>{t('common.pinned')}</span>
        </label>
        <input
          type="number"
          min={0}
          max={9999}
          step={1}
          value={displayPriority}
          aria-label={t('articleEditor.priority')}
          placeholder={t('articleEditor.priority')}
          onChange={e => setDisplayPriority(clampPriority(e.target.value))}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        />
        <input
          type="text"
          placeholder={t('articleEditor.aiSeoTitle')}
          value={seoTitle}
          onChange={e => setSeoTitle(e.target.value)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        />
        <input
          type="text"
          placeholder={t('articleEditor.aiSeoDescription')}
          value={seoDescription}
          onChange={e => setSeoDescription(e.target.value)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        />
        <input
          type="text"
          placeholder={t('articleEditor.aiSeoKeywords')}
          value={seoKeywords}
          onChange={e => setSeoKeywords(e.target.value)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
        <TaxonomyCombobox
          label={t('articleEditor.categoryLabel')}
          placeholder={t('articleEditor.categorySearchPlaceholder')}
          items={categories}
          selectedIds={categoryId === '' ? [] : [Number(categoryId)]}
          error={categoryError}
          creating={categorySubmitting}
          createLabel={value => formatText(t('articleEditor.createCategoryOption'), { value })}
          noMatchesLabel={t('articleEditor.noCategoryMatches')}
          clearLabel={t('articleEditor.clearCategory')}
          creatingLabel={t('common.processing')}
          onSelect={category => {
            setCategoryId(category.id);
            setCategoryError('');
          }}
          onRemove={() => {
            setCategoryId('');
            setCategoryError('');
          }}
          onCreate={handleCreateCategory}
        />

        <TaxonomyCombobox
          label={t('articleEditor.tags')}
          placeholder={t('articleEditor.tagSearchPlaceholder')}
          items={tags}
          selectedIds={tagIds}
          multiple
          error={tagError}
          creating={tagSubmitting}
          createLabel={value => formatText(t('articleEditor.createTagOption'), { value })}
          noMatchesLabel={t('articleEditor.noTagMatches')}
          clearLabel={t('articleEditor.removeTag')}
          creatingLabel={t('common.processing')}
          onSelect={tag => {
            setTagIds(prev => prev.includes(tag.id) ? prev : [...prev, tag.id]);
            setTagError('');
          }}
          onRemove={tagId => {
            setTagIds(prev => prev.filter(prevId => prevId !== tagId));
            setTagError('');
          }}
          onCreate={handleCreateTag}
        />
      </div>

      <div className="mb-6">
        <input
          type="text" placeholder={t('articleEditor.summary')} value={summary} onChange={e => handleSummaryChange(e.target.value)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        />
        <p className="mt-1 text-right text-xs text-ink-light" aria-live="polite">
          {formatText(t('common.byteUsage'), { used: summaryByteLength, limit: MAX_TEXT_FIELD_BYTES })}
        </p>
      </div>

      <label className="mb-6 flex flex-wrap items-center gap-3 border border-mountain-grey bg-[var(--paper-soft)] px-3 py-2 text-sm text-ink-light">
        <input
          type="checkbox"
          checked={generateSummaryOnSave}
          onChange={e => setGenerateSummaryOnSave(e.target.checked)}
          className="h-4 w-4 accent-ochre"
        />
        <span className="font-bold text-ink">{t('articleEditor.generateSummaryOnSave')}</span>
        <span className="text-xs">{t('articleEditor.generateSummaryHint')}</span>
      </label>

      <div className="flex-grow flex flex-col md:flex-row gap-6 overflow-hidden">
        <div className="w-full md:w-1/2 flex flex-col border border-mountain-grey p-4">
          <div className="mb-3 flex flex-wrap items-center justify-between gap-2 border-b border-hairline pb-3 text-xs text-muted">
            <span>
              {formatText(t('reading.words'), { count: liveReadingStats.wordCount })}
              {' · '}
              {formatText(t('reading.minutes'), { count: liveReadingStats.readingTimeMinutes })}
              {' · '}
              {formatText(t('common.byteUsage'), { used: contentByteLength, limit: MAX_ARTICLE_CONTENT_BYTES })}
            </span>
            <Button type="button" size="sm" variant="subtle" onClick={() => setMediaPickerMode('content')}>
              {t('media.insert')}
            </Button>
          </div>
          <textarea
            ref={textareaRef}
            value={content}
            onChange={e => setContent(e.target.value)}
            className="w-full h-full bg-transparent resize-none focus:outline-none text-ink-light font-serif leading-relaxed"
            placeholder={t('articleEditor.content')}
          ></textarea>
        </div>
<div className="w-full md:w-1/2 border border-mountain-grey p-4 overflow-y-auto bg-[var(--paper-soft)]">
  <MarkdownRenderer content={debouncedPreviewContent} />
</div>
      </div>
      <MediaPicker
        open={mediaPickerMode !== null}
        onOpenChange={(open) => {
          if (!open) setMediaPickerMode(null);
        }}
        onSelect={handleMediaSelect}
      />
    </div>
  );
};

export default ArticleEditor;

const toDateTimeLocal = (value?: string) => {
  if (!value) {
    return '';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  const offsetMs = date.getTimezoneOffset() * 60 * 1000;
  return new Date(date.getTime() - offsetMs).toISOString().slice(0, 16);
};

const clampPriority = (value: string) => {
  const next = Number.parseInt(value, 10);
  if (!Number.isFinite(next)) {
    return 0;
  }
  return Math.min(9999, Math.max(0, next));
};

type EditorDraft = {
  title: string;
  slug: string;
  summary: string;
  generateSummaryOnSave: boolean;
  content: string;
  coverUrl: string;
  status: ArticleStatus;
  scheduledAt: string;
  isPinned: boolean;
  displayPriority: number;
  seoTitle: string;
  seoDescription: string;
  seoKeywords: string;
  categoryId: number | '';
  tagIds: number[];
  savedAt: number;
};

type DraftSetters = {
  setTitle: (value: string) => void;
  setSlug: (value: string) => void;
  setSummary: (value: string) => void;
  setGenerateSummaryOnSave: (value: boolean) => void;
  setContent: (value: string) => void;
  setCoverUrl: (value: string) => void;
  setStatus: (value: ArticleStatus) => void;
  setScheduledAt: (value: string) => void;
  setIsPinned: (value: boolean) => void;
  setDisplayPriority: (value: number) => void;
  setSeoTitle: (value: string) => void;
  setSeoDescription: (value: string) => void;
  setSeoKeywords: (value: string) => void;
  setCategoryId: (value: number | '') => void;
  setTagIds: (value: number[]) => void;
};

const emptyEditorDraft = (): EditorDraft => ({
  title: '',
  slug: '',
  summary: '',
  generateSummaryOnSave: true,
  content: '',
  coverUrl: '',
  status: 'draft',
  scheduledAt: '',
  isPinned: false,
  displayPriority: 0,
  seoTitle: '',
  seoDescription: '',
  seoKeywords: '',
  categoryId: '',
  tagIds: [],
  savedAt: 0,
});

const articleToEditorDraft = (article: Article): EditorDraft => ({
  title: article.title || '',
  slug: article.slug || '',
  summary: article.summary || '',
  generateSummaryOnSave: (article.summary || '').trim() === '',
  content: article.content || '',
  coverUrl: article.coverUrl || '',
  status: article.status || 'draft',
  scheduledAt: toDateTimeLocal(article.scheduledAt),
  isPinned: Boolean(article.isPinned),
  displayPriority: clampPriority(String(article.displayPriority || 0)),
  seoTitle: article.seoTitle || '',
  seoDescription: article.seoDescription || '',
  seoKeywords: article.seoKeywords || '',
  categoryId: article.categoryId || '',
  tagIds: Array.isArray(article.tags) ? article.tags.map(tag => tag.id) : [],
  savedAt: 0,
});

const currentEditorDraft = (draft: Omit<EditorDraft, 'savedAt'>): EditorDraft => ({
  ...draft,
  displayPriority: clampPriority(String(draft.displayPriority || 0)),
  categoryId: draft.categoryId || '',
  tagIds: Array.isArray(draft.tagIds) ? draft.tagIds : [],
  savedAt: Date.now(),
});

const editorDraftComparable = (draft: EditorDraft) => ({
  title: draft.title || '',
  slug: draft.slug || '',
  summary: draft.summary || '',
  generateSummaryOnSave: Boolean(draft.generateSummaryOnSave),
  content: draft.content || '',
  coverUrl: draft.coverUrl || '',
  status: draft.status || 'draft',
  scheduledAt: draft.scheduledAt || '',
  isPinned: Boolean(draft.isPinned),
  displayPriority: clampPriority(String(draft.displayPriority || 0)),
  seoTitle: draft.seoTitle || '',
  seoDescription: draft.seoDescription || '',
  seoKeywords: draft.seoKeywords || '',
  categoryId: draft.categoryId || '',
  tagIds: Array.isArray(draft.tagIds) ? [...draft.tagIds].sort((a, b) => a - b) : [],
});

const editorDraftEquals = (left: EditorDraft, right: EditorDraft) =>
  JSON.stringify(editorDraftComparable(left)) === JSON.stringify(editorDraftComparable(right));

const hasMeaningfulEditorDraft = (draft: EditorDraft) => {
  const normalized = editorDraftComparable(draft);
  return Boolean(
    normalized.title.trim() ||
    normalized.slug.trim() ||
    normalized.summary.trim() ||
    normalized.content.trim() ||
    normalized.coverUrl.trim() ||
    normalized.scheduledAt.trim() ||
    normalized.seoTitle.trim() ||
    normalized.seoDescription.trim() ||
    normalized.seoKeywords.trim() ||
    normalized.categoryId ||
    normalized.tagIds.length > 0 ||
    normalized.isPinned ||
    normalized.displayPriority > 0 ||
    normalized.status !== 'draft'
  );
};

const shouldRecoverEditorDraft = (draft: EditorDraft | null, baseline: EditorDraft, updatedAt?: string) => {
  if (!draft || !hasMeaningfulEditorDraft(draft) || editorDraftEquals(draft, baseline)) {
    return false;
  }
  if (!updatedAt) {
    return true;
  }
  const updatedAtMs = new Date(updatedAt).getTime();
  return !Number.isFinite(updatedAtMs) || draft.savedAt > updatedAtMs;
};

const editorDraftKey = (id?: string | number | false) => `article-editor:draft:${id || 'new'}`;

const readEditorDraft = (key: string): EditorDraft | null => {
  try {
    const raw = safeLocalStorage.getItem(key);
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw) as EditorDraft;
    if (!parsed || typeof parsed.savedAt !== 'number') {
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
};

const writeEditorDraft = (key: string, draft: EditorDraft) => {
  safeLocalStorage.setItem(key, JSON.stringify(draft));
};

const removeEditorDraft = (key: string) => {
  safeLocalStorage.removeItem(key);
};

const applyEditorDraft = (draft: EditorDraft, setters: DraftSetters) => {
  setters.setTitle(draft.title || '');
  setters.setSlug(draft.slug || '');
  setters.setSummary(draft.summary || '');
  setters.setGenerateSummaryOnSave(Boolean(draft.generateSummaryOnSave));
  setters.setContent(draft.content || '');
  setters.setCoverUrl(draft.coverUrl || '');
  setters.setStatus(draft.status || 'draft');
  setters.setScheduledAt(draft.scheduledAt || '');
  setters.setIsPinned(Boolean(draft.isPinned));
  setters.setDisplayPriority(clampPriority(String(draft.displayPriority || 0)));
  setters.setSeoTitle(draft.seoTitle || '');
  setters.setSeoDescription(draft.seoDescription || '');
  setters.setSeoKeywords(draft.seoKeywords || '');
  setters.setCategoryId(draft.categoryId || '');
  setters.setTagIds(Array.isArray(draft.tagIds) ? draft.tagIds : []);
};

const getActiveMarkdownRange = (textarea: HTMLTextAreaElement | null, value: string) => {
  const selectionStart = textarea?.selectionStart ?? 0;
  const selectionEnd = textarea?.selectionEnd ?? selectionStart;
  if (selectionEnd > selectionStart) {
    return {
      start: selectionStart,
      end: selectionEnd,
      text: value.slice(selectionStart, selectionEnd),
    };
  }
  let start = value.lastIndexOf('\n\n', Math.max(0, selectionStart - 1));
  start = start === -1 ? 0 : start + 2;
  let end = value.indexOf('\n\n', selectionStart);
  end = end === -1 ? value.length : end;
  return {
    start,
    end,
    text: value.slice(start, end),
  };
};
