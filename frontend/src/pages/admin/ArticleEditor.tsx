import React, { useEffect, useMemo, useState } from 'react';
import { useLocation, useParams, useNavigate } from 'react-router-dom';
import { assistArticle, getArticlePreview, createArticle, updateArticle } from '../../api/article';
import { createCategory, getCategories } from '../../api/category';
import { createTag, getTags } from '../../api/tag';
import { Category, Tag } from '../../types';
import type { AIAssistAction } from '../../types/api';
import InlineNotice from '../../components/InlineNotice';
import MarkdownRenderer from '../../components/MarkdownRenderer';
import { getErrorMessage } from '../../utils/error';
import { generateSlug } from '../../utils/slug';
import { isValidCoverUrl } from '../../utils/cover';
import { formatText, getArticleStatusLabel, translate } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';

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
  };

  const handleCreate = async () => {
    if (!canCreate || creating) {
      return;
    }
    const created = await onCreate(trimmedQuery);
    if (created) {
      setQuery('');
      setOpen(false);
    }
  };

  const handleInputChange = (value: string) => {
    if (!multiple && selectedCategory) {
      onRemove(selectedCategory.id);
    }
    setQuery(value);
    setOpen(true);
  };

  const handleKeyDown = async (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      if (exactNameItem) {
        handleSelect(exactNameItem);
        return;
      }
      await handleCreate();
    }
    if (e.key === 'Escape') {
      setOpen(false);
    }
    if (multiple && e.key === 'Backspace' && query === '' && selectedIds.length > 0) {
      onRemove(selectedIds[selectedIds.length - 1]);
    }
  };

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
          type="text"
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
        <div className="absolute left-0 right-0 top-full z-20 mt-1 max-h-56 overflow-y-auto border border-mountain-grey bg-paper shadow-sm">
          {filteredItems.map(item => (
            <button
              key={item.id}
              type="button"
              onClick={() => handleSelect(item)}
              className="block w-full px-3 py-2 text-left text-sm text-ink hover:bg-mountain-grey hover:bg-opacity-20"
            >
              <span className="font-bold">{item.name}</span>
              <span className="ml-2 text-xs text-ink-light">{item.slug}</span>
            </button>
          ))}
          {canCreate && (
            <button
              type="button"
              onClick={handleCreate}
              disabled={creating}
              className="block w-full px-3 py-2 text-left text-sm text-ochre hover:bg-mountain-grey hover:bg-opacity-20 disabled:opacity-50 disabled:cursor-not-allowed"
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

  const [title, setTitle] = useState('');
  const [slug, setSlug] = useState('');
  const [summary, setSummary] = useState('');
  const [generateSummaryOnSave, setGenerateSummaryOnSave] = useState(true);
  const [content, setContent] = useState('');
  const [coverUrl, setCoverUrl] = useState('');
  const [status, setStatus] = useState('draft');
  const [scheduledAt, setScheduledAt] = useState('');
  const [isPinned, setIsPinned] = useState(false);
  const [displayPriority, setDisplayPriority] = useState(0);
  const [seoTitle, setSeoTitle] = useState('');
  const [seoDescription, setSeoDescription] = useState('');
  const [seoKeywords, setSeoKeywords] = useState('');
  const [categoryId, setCategoryId] = useState<number | ''>('');
  const [tagIds, setTagIds] = useState<number[]>([]);

  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [categoryError, setCategoryError] = useState('');
  const [categorySubmitting, setCategorySubmitting] = useState(false);
  const [tagError, setTagError] = useState('');
  const [tagSubmitting, setTagSubmitting] = useState(false);
  const [aiMenuOpen, setAiMenuOpen] = useState(false);
  const [aiAction, setAiAction] = useState<AIAssistAction | null>(null);
  const [aiNotice, setAiNotice] = useState('');
  const [aiDraft, setAiDraft] = useState<{
    action: AIAssistAction;
    revisedContent: string;
    suggestions: string[];
  } | null>(null);
  const aiText = getAIEditorLabels(language);
  const pinText = getPinPriorityLabels(language);

  useEffect(() => {
    const state = location.state as { aiNotice?: string } | null;
    if (!state?.aiNotice) {
      return;
    }
    setAiNotice(state.aiNotice);
    navigate(location.pathname, { replace: true, state: null });
  }, [location.pathname, location.state, navigate]);

  useEffect(() => {
    const fetchDeps = async () => {
      try {
        const [catRes, tagRes] = await Promise.all([
          getCategories({ size: 100 }),
          getTags({ size: 100 }),
        ]);
        setCategories(catRes.data.items || []);
        setTags(tagRes.data.items || []);
      } catch (e) {
        setError(getErrorMessage(e, translate(language, 'articleEditor.depsError')));
      }
    };
    fetchDeps();

    if (isEdit) {
      getArticlePreview(id)
        .then(res => {
          const article = res.data;
          const nextSummary = article.summary || '';
          setTitle(article.title);
          setSlug(article.slug);
          setSummary(nextSummary);
          setGenerateSummaryOnSave(nextSummary.trim() === '');
          setContent(article.content || '');
          setCoverUrl(article.coverUrl || '');
          setStatus(article.status);
          setScheduledAt(toDateTimeLocal(article.scheduledAt));
          setIsPinned(Boolean(article.isPinned));
          setDisplayPriority(article.displayPriority || 0);
          setSeoTitle(article.seoTitle || '');
          setSeoDescription(article.seoDescription || '');
          setSeoKeywords(article.seoKeywords || '');
          setCategoryId(article.categoryId || '');
          if (article.tags) setTagIds(article.tags.map(tag => tag.id));
        })
        .catch(e => setError(getErrorMessage(e, translate(language, 'article.loadError'))));
    }
  }, [id, isEdit, language]);

  const handleSummaryChange = (value: string) => {
    setSummary(value);
    setGenerateSummaryOnSave(value.trim() === '');
  };

  const handleSave = async () => {
    const trimmedCoverUrl = coverUrl.trim();
    if (!isValidCoverUrl(trimmedCoverUrl)) {
      setError(t('articleEditor.coverUrlError'));
      return;
    }

    setError('');
    setAiNotice('');
    setSubmitting(true);
    let nextSummary = summary;
    let summaryGenerationFailed = false;
    try {
      if (generateSummaryOnSave && content.trim()) {
        try {
          const aiRes = await assistArticle({ action: 'metadata', title, content });
          const generatedSummary = aiRes.data.summary?.trim();
          if (generatedSummary) {
            nextSummary = generatedSummary;
            setSummary(generatedSummary);
            setGenerateSummaryOnSave(false);
          } else {
            summaryGenerationFailed = true;
          }
        } catch {
          summaryGenerationFailed = true;
        }
      }

      const payload = {
        title, slug, summary: nextSummary, content, coverUrl: trimmedCoverUrl, status,
        scheduledAt: scheduledAt ? new Date(scheduledAt).toISOString() : undefined,
        isPinned,
        displayPriority,
        seoTitle: seoTitle.trim(),
        seoDescription: seoDescription.trim(),
        seoKeywords: seoKeywords.trim(),
        categoryId: categoryId === '' ? 0 : Number(categoryId),
        tagIds,
      };

      if (isEdit) {
        await updateArticle(id, payload);
      } else {
        const created = await createArticle(payload);
        if (summaryGenerationFailed) {
          navigate(`/admin/editor/${created.data.id}`, { replace: true, state: { aiNotice: aiText.saveSummaryFailed } });
          return;
        }
      }
      if (summaryGenerationFailed) {
        setAiNotice(aiText.saveSummaryFailed);
        return;
      }
      navigate('/admin/articles');
    } catch (e: unknown) {
      setError(getErrorMessage(e, t('articleEditor.saveError')));
    } finally {
      setSubmitting(false);
    }
  };

  const handleAIAssist = async (action: AIAssistAction) => {
    if (!content.trim()) {
      setError(aiText.contentRequired);
      setAiMenuOpen(false);
      return;
    }
    setError('');
    setAiNotice('');
    setAiDraft(null);
    setAiAction(action);
    setAiMenuOpen(false);
    try {
      const res = await assistArticle({ action, title, content });
      const data = res.data;
      if (action === 'metadata') {
        if (data.summary) setSummary(data.summary);
        if (data.seoDescription) setSeoDescription(data.seoDescription);
        if (data.seoKeywords) setSeoKeywords(data.seoKeywords);
        setAiNotice(aiText.metadataApplied);
        return;
      }
      setAiDraft({
        action,
        revisedContent: data.revisedContent || '',
        suggestions: data.suggestions || [],
      });
      setAiNotice(action === 'proofread' ? aiText.proofreadReady : aiText.polishReady);
    } catch (e: unknown) {
      setError(getErrorMessage(e, aiText.aiError));
    } finally {
      setAiAction(null);
    }
  };

  const applyAIContent = () => {
    if (!aiDraft?.revisedContent) {
      return;
    }
    setContent(aiDraft.revisedContent);
    setAiDraft(null);
    setAiNotice(aiText.contentApplied);
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
              {aiAction ? aiText.processing : aiText.magic}
            </button>
            {aiMenuOpen && (
              <div className="absolute right-0 top-full z-30 mt-2 w-52 border border-mountain-grey bg-paper shadow-sm">
                <button type="button" onClick={() => handleAIAssist('metadata')} className="block w-full px-3 py-2 text-left text-sm text-ink hover:bg-mountain-grey hover:bg-opacity-20">
                  {aiText.metadata}
                </button>
                <button type="button" onClick={() => handleAIAssist('proofread')} className="block w-full px-3 py-2 text-left text-sm text-ink hover:bg-mountain-grey hover:bg-opacity-20">
                  {aiText.proofread}
                </button>
                <button type="button" onClick={() => handleAIAssist('polish')} className="block w-full px-3 py-2 text-left text-sm text-ink hover:bg-mountain-grey hover:bg-opacity-20">
                  {aiText.polish}
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

      {aiDraft && (
        <section className="mb-6 border border-mountain-grey bg-[var(--paper-soft)] p-4">
          <div className="mb-3 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <h4 className="text-sm font-bold tracking-widest text-ink">
              {aiDraft.action === 'proofread' ? aiText.proofreadResult : aiText.polishResult}
            </h4>
            <div className="flex gap-3">
              <button type="button" onClick={applyAIContent} className="border border-ink px-3 py-1.5 text-sm text-ink transition-colors hover:bg-ink hover:text-paper">
                {aiText.apply}
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
        <input
          type="text" placeholder={t('articleEditor.coverPlaceholder')} value={coverUrl} onChange={e => setCoverUrl(e.target.value)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        />
        <select
          value={status} onChange={e => setStatus(e.target.value)}
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
        <label className="flex items-center gap-3 border-b border-mountain-grey py-2 text-sm text-ink focus-within:border-ochre">
          <input
            type="checkbox"
            checked={isPinned}
            onChange={e => setIsPinned(e.target.checked)}
            className="h-4 w-4 accent-ochre"
          />
          <span>{pinText.pinned}</span>
        </label>
        <input
          type="number"
          min={0}
          max={9999}
          step={1}
          value={displayPriority}
          aria-label={pinText.priority}
          placeholder={pinText.priority}
          onChange={e => setDisplayPriority(clampPriority(e.target.value))}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        />
        <input
          type="text"
          placeholder="SEO Title"
          value={seoTitle}
          onChange={e => setSeoTitle(e.target.value)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        />
        <input
          type="text"
          placeholder="SEO Description"
          value={seoDescription}
          onChange={e => setSeoDescription(e.target.value)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        />
        <input
          type="text"
          placeholder="SEO Keywords"
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

      <input
        type="text" placeholder={t('articleEditor.summary')} value={summary} onChange={e => handleSummaryChange(e.target.value)}
        className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre mb-6"
      />

      <label className="mb-6 flex flex-wrap items-center gap-3 border border-mountain-grey bg-[var(--paper-soft)] px-3 py-2 text-sm text-ink-light">
        <input
          type="checkbox"
          checked={generateSummaryOnSave}
          onChange={e => setGenerateSummaryOnSave(e.target.checked)}
          className="h-4 w-4 accent-ochre"
        />
        <span className="font-bold text-ink">{aiText.generateSummaryOnSave}</span>
        <span className="text-xs">{aiText.generateSummaryHint}</span>
      </label>

      <div className="flex-grow flex flex-col md:flex-row gap-6 overflow-hidden">
        <div className="w-full md:w-1/2 flex flex-col border border-mountain-grey p-4">
          <textarea
            value={content}
            onChange={e => setContent(e.target.value)}
            className="w-full h-full bg-transparent resize-none focus:outline-none text-ink-light font-serif leading-relaxed"
            placeholder={t('articleEditor.content')}
          ></textarea>
        </div>
        <div className="w-full md:w-1/2 border border-mountain-grey p-4 overflow-y-auto bg-[var(--paper-soft)]">
          <MarkdownRenderer content={content} />
        </div>
      </div>
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

const getPinPriorityLabels = (language: string) => language === 'zh'
  ? {
      pinned: '\u7f6e\u9876',
      priority: '\u663e\u793a\u4f18\u5148\u7ea7',
    }
  : {
      pinned: 'Pinned',
      priority: 'Display Priority',
    };

const getAIEditorLabels = (language: string) => language === 'zh'
  ? {
      magic: '✨ AI',
      metadata: '生成摘要与 SEO',
      proofread: '语法纠错',
      polish: '语气润色',
      processing: 'AI 处理中',
      contentRequired: '请先填写正文内容。',
      metadataApplied: 'AI 已填入摘要与 SEO 字段，请检查后再保存。',
      proofreadReady: 'AI 纠错结果已生成，请确认后应用。',
      polishReady: 'AI 润色结果已生成，请确认后应用。',
      proofreadResult: 'AI 纠错结果',
      polishResult: 'AI 润色结果',
      apply: '应用到正文',
      contentApplied: 'AI 结果已应用到正文，请检查后再保存。',
      aiError: 'AI 辅助失败',
      generateSummaryOnSave: '保存时生成核心摘要',
      generateSummaryHint: '默认仅在摘要为空时开启；AI 失败也会继续保存文章。',
      saveSummaryFailed: '文章已保存但摘要生成失败。',
    }
  : {
      magic: '✨ AI',
      metadata: 'Generate Summary / SEO',
      proofread: 'Proofread',
      polish: 'Polish Tone',
      processing: 'AI Working',
      contentRequired: 'Please write the article body first.',
      metadataApplied: 'AI filled summary and SEO fields. Please review before saving.',
      proofreadReady: 'Proofread result is ready. Review before applying.',
      polishReady: 'Polished result is ready. Review before applying.',
      proofreadResult: 'AI Proofread Result',
      polishResult: 'AI Polish Result',
      apply: 'Apply to Body',
      contentApplied: 'AI result was applied to the body. Please review before saving.',
      aiError: 'AI assist failed',
      generateSummaryOnSave: 'Generate summary on save',
      generateSummaryHint: 'Enabled by default only when summary is empty. The article still saves if AI fails.',
      saveSummaryFailed: 'Article saved, but summary generation failed.',
    };
