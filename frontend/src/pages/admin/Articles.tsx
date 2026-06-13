import React, { useCallback, useEffect, useRef, useState } from 'react';
import { deleteArticle, exportArticleMarkdown, getAdminArticles, importMarkdownArticle, updateArticleStatus } from '../../api/article';
import { getCategories } from '../../api/category';
import { getTags } from '../../api/tag';
import { Article, Category, Tag } from '../../types';
import { Link, useNavigate } from 'react-router-dom';
import Pagination from '../../components/Pagination';
import InlineNotice from '../../components/InlineNotice';
import { getErrorMessage } from '../../utils/error';
import { getArticleStatusLabel, getDateLocale, translate } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';

const AdminArticles: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const [articles, setArticles] = useState<Article[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busyId, setBusyId] = useState<number | null>(null);
  const [keyword, setKeyword] = useState('');
  const [appliedKeyword, setAppliedKeyword] = useState('');
  const [status, setStatus] = useState('');
  const [categoryId, setCategoryId] = useState(0);
  const [tagId, setTagId] = useState(0);
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [importing, setImporting] = useState(false);
  const size = 10;
  const navigate = useNavigate();
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const extraLabels = adminArticleExtraLabels(language);
  const pinLabels = articlePinLabels(language);
  const getDisplayStatus = (article: Article) => {
    if (article.status === 'published' && article.scheduledAt && new Date(article.scheduledAt).getTime() > Date.now()) {
      return 'scheduled';
    }
    return article.status;
  };
  const getDisplayStatusLabel = (article: Article) => {
    const displayStatus = getDisplayStatus(article);
    if (displayStatus === 'scheduled') {
      return extraLabels.scheduled;
    }
    return getArticleStatusLabel(language, displayStatus);
  };

  const fetchList = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await getAdminArticles({
        page,
        size,
        ...(appliedKeyword ? { q: appliedKeyword } : {}),
        ...(status ? { status } : {}),
        ...(categoryId ? { categoryId } : {}),
        ...(tagId ? { tagId } : {}),
      });
      setArticles(res.data.items || []);
      setTotal(res.data.total || 0);
    } catch (e) {
      setError(getErrorMessage(e, translate(language, 'adminArticles.loadError')));
    } finally {
      setLoading(false);
    }
  }, [page, size, appliedKeyword, status, categoryId, tagId, language]);

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  useEffect(() => {
    const fetchTaxonomies = async () => {
      try {
        const [categoryRes, tagRes] = await Promise.all([
          getCategories({ size: 100 }),
          getTags({ size: 100 }),
        ]);
        setCategories(categoryRes.data.items || []);
        setTags(tagRes.data.items || []);
      } catch (e) {
        setError(getErrorMessage(e, translate(language, 'archive.loadError')));
      }
    };
    fetchTaxonomies();
  }, [language]);

  const resetPage = () => {
    setPage(1);
  };

  const handleFilterSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setAppliedKeyword(keyword.trim());
    resetPage();
  };

  const handleClearFilters = () => {
    setKeyword('');
    setAppliedKeyword('');
    setStatus('');
    setCategoryId(0);
    setTagId(0);
    setPage(1);
  };

  const handleStatus = async (id: number, status: string) => {
    setError('');
    setBusyId(id);
    try {
      await updateArticleStatus(id, status);
      await fetchList();
    } catch (e) {
      setError(getErrorMessage(e, t('adminArticles.statusError')));
    } finally {
      setBusyId(null);
    }
  };

  const handleDelete = async (id: number) => {
    if (confirm(t('adminArticles.confirmDelete'))) {
      setError('');
      setBusyId(id);
      try {
        await deleteArticle(id);
        await fetchList();
      } catch (e) {
        setError(getErrorMessage(e, t('adminArticles.deleteError')));
      } finally {
        setBusyId(null);
      }
    }
  };

  const handleImport = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) {
      return;
    }
    setError('');
    setImporting(true);
    try {
      const res = await importMarkdownArticle(file);
      navigate(`/admin/editor/${res.data.id}`);
    } catch (e) {
      setError(getErrorMessage(e, extraLabels.importError));
    } finally {
      setImporting(false);
    }
  };

  const handleExport = async (id: number) => {
    setError('');
    setBusyId(id);
    try {
      const { blob, filename } = await exportArticleMarkdown(id);
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = filename;
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(url);
    } catch (e) {
      setError(getErrorMessage(e, extraLabels.exportError));
    } finally {
      setBusyId(null);
    }
  };

  return (
    <div>
      <div className="mb-8 flex flex-col gap-3 border-b border-mountain-grey pb-4 sm:flex-row sm:items-center sm:justify-between">
        <h3 className="text-2xl font-bold text-ink tracking-widest">{t('admin.articles')}</h3>
        <div className="flex flex-wrap gap-3">
          <input ref={fileInputRef} type="file" accept=".md,text/markdown,text/plain" onChange={handleImport} className="hidden" />
          <button
            onClick={() => fileInputRef.current?.click()}
            disabled={importing}
            className="px-4 py-2 border border-mountain-grey text-ink hover:border-ochre hover:text-ochre tracking-widest text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-50"
          >
            {importing ? extraLabels.importing : extraLabels.import}
          </button>
          <button onClick={() => navigate('/admin/editor/new')} className="px-4 py-2 border border-ink text-ink hover:bg-ink hover:text-paper tracking-widest text-sm transition-colors">
            {t('adminArticles.new')}
          </button>
        </div>
      </div>

      <InlineNotice message={error} className="mb-6" />

      <form onSubmit={handleFilterSubmit} className="grid grid-cols-1 gap-3 mb-8 md:grid-cols-5">
        <input
          value={keyword}
          onChange={(event) => setKeyword(event.target.value)}
          placeholder={t('home.searchPlaceholder')}
          className="bg-transparent border border-mountain-grey px-3 py-2 text-sm text-ink outline-none focus:border-ochre md:col-span-2"
        />
        <select
          value={status}
          onChange={(event) => {
            setStatus(event.target.value);
            resetPage();
          }}
          className="bg-paper border border-mountain-grey px-3 py-2 text-sm text-ink outline-none focus:border-ochre"
        >
          <option value="">{t('adminArticles.allStatus')}</option>
          <option value="draft">{getArticleStatusLabel(language, 'draft')}</option>
          <option value="published">{getArticleStatusLabel(language, 'published')}</option>
          <option value="scheduled">{extraLabels.scheduled}</option>
          <option value="archived">{getArticleStatusLabel(language, 'archived')}</option>
        </select>
        <select
          value={categoryId}
          onChange={(event) => {
            setCategoryId(Number(event.target.value));
            resetPage();
          }}
          className="bg-paper border border-mountain-grey px-3 py-2 text-sm text-ink outline-none focus:border-ochre"
        >
          <option value={0}>{t('adminArticles.allCategories')}</option>
          {categories.map((category) => (
            <option key={category.id} value={category.id}>{category.name}</option>
          ))}
        </select>
        <select
          value={tagId}
          onChange={(event) => {
            setTagId(Number(event.target.value));
            resetPage();
          }}
          className="bg-paper border border-mountain-grey px-3 py-2 text-sm text-ink outline-none focus:border-ochre"
        >
          <option value={0}>{t('adminArticles.allTags')}</option>
          {tags.map((tag) => (
            <option key={tag.id} value={tag.id}>#{tag.name}</option>
          ))}
        </select>
        <div className="flex flex-wrap gap-3 md:col-span-5">
          <button type="submit" className="px-4 py-2 border border-ink text-ink hover:bg-ink hover:text-paper tracking-widest text-sm transition-colors">
            {t('home.search')}
          </button>
          <button type="button" onClick={handleClearFilters} className="px-4 py-2 border border-mountain-grey text-ink-light hover:border-ochre hover:text-ochre tracking-widest text-sm transition-colors">
            {t('home.clearFilters')}
          </button>
        </div>
      </form>

      {loading ? (
        <div className="py-16 text-center tracking-widest text-ink-light">{t('common.loading')}</div>
      ) : articles.length === 0 ? (
        <div className="py-16 text-center tracking-widest text-ink-light">{t('common.empty')}</div>
      ) : (
        <>
          <table className="admin-responsive-table w-full text-left border-collapse text-sm">
            <thead>
              <tr className="border-b border-mountain-grey text-ink-light opacity-80 tracking-widest">
                <th className="py-3 font-normal">{t('adminArticles.title')}</th>
                <th className="py-3 font-normal">{t('adminArticles.taxonomy')}</th>
                <th className="w-[5.5rem] min-w-[5.5rem] py-3 font-normal">{t('common.status')}</th>
                <th className="py-3 font-normal">{t('common.time')}</th>
                <th className="py-3 font-normal text-right">{t('common.action')}</th>
              </tr>
            </thead>
            <tbody>
              {articles.map(a => (
                <tr key={a.id} className="border-b border-mountain-grey border-opacity-50 hover:bg-mountain-grey hover:bg-opacity-20 transition-colors text-ink">
                  <td data-label={t('adminArticles.title')} className="admin-card-title py-4">
                    <div className="font-bold">{a.title}</div>
                    {(a.isPinned || a.displayPriority > 0) && (
                      <div className="mt-2 flex flex-wrap gap-2 text-xs font-normal tracking-wider">
                        {a.isPinned && (
                          <span className="border border-ochre px-2 py-0.5 text-ochre">{pinLabels.pinned}</span>
                        )}
                        {a.displayPriority > 0 && (
                          <span className="border border-mountain-grey px-2 py-0.5 text-ink-light">
                            {pinLabels.priority} {a.displayPriority}
                          </span>
                        )}
                      </div>
                    )}
                  </td>
                  <td data-label={t('adminArticles.taxonomy')} className="py-4">
                    <div className="flex flex-col space-y-1">
                      {a.category && (
                        <span className="text-ochre text-xs">{a.category.name}</span>
                      )}
                      {a.tags && a.tags.length > 0 && (
                        <div className="flex flex-wrap gap-x-2 gap-y-1 text-xs text-ink-light opacity-80">
                          {a.tags.map(tg => <span key={tg.name}>#{tg.name}</span>)}
                        </div>
                      )}
                    </div>
                  </td>
                  <td data-label={t('common.status')} className="py-4 md:w-[5.5rem] md:min-w-[5.5rem]">
                    <span className={`inline-flex min-w-[3.75rem] items-center justify-center whitespace-nowrap px-2 py-1 text-xs border ${getDisplayStatus(a) === 'published' ? 'border-ochre text-ochre' : 'border-ink-light text-ink-light'}`}>
                      {getDisplayStatusLabel(a)}
                    </span>
                    {a.scheduledAt && (
                      <div className="mt-2 text-xs text-ink-light opacity-70">
                        {new Date(a.scheduledAt).toLocaleString(getDateLocale(language))}
                      </div>
                    )}
                  </td>
                  <td data-label={t('common.time')} className="py-4 text-ink-light opacity-80">
                    {new Date(a.createdAt).toLocaleDateString(getDateLocale(language))}
                  </td>
                  <td data-label={t('common.action')} className="admin-card-actions py-4 text-right tracking-wider">
                    <div className="admin-action-list">
                      <Link to={`/admin/editor/${a.id}`} className="hover:text-ochre">{t('common.edit')}</Link>
                      <Link to={`/admin/preview/${a.id}`} className="hover:text-ochre">{extraLabels.preview}</Link>
                      <Link to={`/admin/articles/${a.id}/versions`} className="hover:text-ochre">{extraLabels.versions}</Link>
                      <button onClick={() => handleExport(a.id)} disabled={busyId === a.id} className="hover:text-ochre disabled:opacity-50 disabled:cursor-not-allowed">{extraLabels.export}</button>
                      {a.status !== 'published' && (
                        <button onClick={() => handleStatus(a.id, 'published')} disabled={busyId === a.id} className="hover:text-ochre disabled:opacity-50 disabled:cursor-not-allowed">{t('adminArticles.publish')}</button>
                      )}
                      {a.status === 'published' && (
                        <button onClick={() => handleStatus(a.id, 'archived')} disabled={busyId === a.id} className="hover:text-ochre disabled:opacity-50 disabled:cursor-not-allowed">{t('adminArticles.archive')}</button>
                      )}
                      <button onClick={() => handleDelete(a.id)} disabled={busyId === a.id} className="text-ochre opacity-80 hover:opacity-100 disabled:opacity-50 disabled:cursor-not-allowed">{t('common.delete')}</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <Pagination
            currentPage={page}
            total={total}
            pageSize={size}
            onPageChange={setPage}
          />
        </>
      )}
    </div>
  );
};

const adminArticleExtraLabels = (language: string) => language === 'zh'
  ? {
      import: '导入 Markdown',
      importing: '导入中...',
      importError: 'Markdown 导入失败',
      export: '导出',
      exportError: 'Markdown 导出失败',
      preview: '预览',
      versions: '版本',
      scheduled: '定时发布',
    }
  : {
      import: 'Import Markdown',
      importing: 'Importing...',
      importError: 'Markdown import failed',
      export: 'Export',
      exportError: 'Markdown export failed',
      preview: 'Preview',
      versions: 'Versions',
      scheduled: 'Scheduled',
    };

const articlePinLabels = (language: string) => language === 'zh'
  ? {
      pinned: '\u7f6e\u9876',
      priority: '\u4f18\u5148\u7ea7',
    }
  : {
      pinned: 'Pinned',
      priority: 'Priority',
    };

export default AdminArticles;
