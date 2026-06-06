import React, { useEffect, useMemo, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import ReactMarkdown from 'react-markdown';
import { Components } from 'react-markdown';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vs } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { getArticleById, createArticle, updateArticle } from '../../api/article';
import { createCategory, getCategories } from '../../api/category';
import { createTag, getTags } from '../../api/tag';
import { Category, Tag } from '../../types';
import InlineNotice from '../../components/InlineNotice';
import { getErrorMessage } from '../../utils/error';
import { generateSlug } from '../../utils/slug';
import { isValidCoverUrl } from '../../utils/cover';
import { formatText, getArticleStatusLabel, translate } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';

const markdownComponents: Components = {
  code({ className, children, ...props }) {
    const match = /language-(\w+)/.exec(className || '');
    return match ? (
      <SyntaxHighlighter
        children={String(children).replace(/\n$/, '')}
        style={vs}
        language={match[1]}
        PreTag="div"
        className="rounded-sm border border-mountain-grey"
      />
    ) : (
      <code {...props} className="bg-mountain-grey bg-opacity-30 px-1 py-0.5 rounded-sm font-sans text-ink">
        {children}
      </code>
    );
  },
};

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
  const language = usePreferenceStore((state) => state.language);
  const isEdit = id && id !== 'new';
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  const [title, setTitle] = useState('');
  const [slug, setSlug] = useState('');
  const [summary, setSummary] = useState('');
  const [content, setContent] = useState('');
  const [coverUrl, setCoverUrl] = useState('');
  const [status, setStatus] = useState('draft');
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
      getArticleById(id)
        .then(res => {
          const article = res.data;
          setTitle(article.title);
          setSlug(article.slug);
          setSummary(article.summary);
          setContent(article.content || '');
          setCoverUrl(article.coverUrl || '');
          setStatus(article.status);
          setCategoryId(article.categoryId || '');
          if (article.tags) setTagIds(article.tags.map(tag => tag.id));
        })
        .catch(e => setError(getErrorMessage(e, translate(language, 'article.loadError'))));
    }
  }, [id, isEdit, language]);

  const handleSave = async () => {
    const trimmedCoverUrl = coverUrl.trim();
    if (!isValidCoverUrl(trimmedCoverUrl)) {
      setError(t('articleEditor.coverUrlError'));
      return;
    }

    const payload = {
      title, slug, summary, content, coverUrl: trimmedCoverUrl, status,
      categoryId: categoryId === '' ? 0 : Number(categoryId),
      tagIds,
    };

    setError('');
    setSubmitting(true);
    try {
      if (isEdit) {
        await updateArticle(id, payload);
      } else {
        await createArticle(payload);
      }
      navigate('/admin/articles');
    } catch (e: unknown) {
      setError(getErrorMessage(e, t('articleEditor.saveError')));
    } finally {
      setSubmitting(false);
    }
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
        <button onClick={handleSave} disabled={submitting} className="px-6 py-2 bg-ink text-paper tracking-widest text-sm hover:bg-opacity-80 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
          {submitting ? t('common.saving') : t('articleEditor.save')}
        </button>
      </div>

      <InlineNotice message={error} className="mb-6" />

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
        type="text" placeholder={t('articleEditor.summary')} value={summary} onChange={e => setSummary(e.target.value)}
        className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre mb-6"
      />

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
          <div className="prose prose-stone max-w-none
            prose-headings:font-serif prose-headings:font-bold prose-headings:text-ink
            prose-p:text-ink-light prose-p:leading-loose prose-p:tracking-wide
            prose-a:text-ochre
            prose-blockquote:border-l-4 prose-blockquote:border-mountain-grey prose-blockquote:pl-6 prose-blockquote:italic
          ">
            <ReactMarkdown components={markdownComponents}>
              {content}
            </ReactMarkdown>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ArticleEditor;
