import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { getArticleVersion, getArticleVersions, restoreArticleVersion } from '../../api/article';
import InlineNotice from '../../components/InlineNotice';
import MarkdownRenderer from '../../components/MarkdownRenderer';
import Pagination from '../../components/Pagination';
import TableSkeleton from '../../components/ui/TableSkeleton';
import { getArticleStatusLabel, formatText, getDateLocale, translate } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';
import { getErrorMessage } from '../../utils/error';
import { useConfirm } from '../../hooks/useConfirm';
import type { ArticleVersion } from '../../types';

const ArticleVersions: React.FC = () => {
  const { id } = useParams();
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  // 用 ref 持有最新 language，避免切换语言时重新拉取版本列表（数据与语言无关，
  // language 仅用于错误兜底文案）。
  const languageRef = useRef(language);
  languageRef.current = language;
  const confirm = useConfirm();
  const [versions, setVersions] = useState<ArticleVersion[]>([]);
  const [selectedVersion, setSelectedVersion] = useState<ArticleVersion | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);
  const [error, setError] = useState('');
  const [busyVersion, setBusyVersion] = useState<number | null>(null);
  const size = 10;
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const fetchVersions = useCallback(async () => {
    if (!id) {
      setLoading(false);
      return;
    }
    setLoading(true);
    setError('');
    try {
      const res = await getArticleVersions(id, { page, size });
      if (!mountedRef.current) {
        return;
      }
      setVersions(res.data.items || []);
      setTotal(res.data.total || 0);
    } catch (e) {
      if (!mountedRef.current) {
        return;
      }
      setError(getErrorMessage(e, translate(languageRef.current, 'articleVersion.historyLoadError')));
    } finally {
      if (mountedRef.current) {
        setLoading(false);
      }
    }
  }, [id, page]);

  useEffect(() => {
    fetchVersions();
  }, [fetchVersions]);

  const handleInspect = async (versionNo: number) => {
    if (!id) {
      return;
    }
    setDetailLoading(true);
    setError('');
    try {
      const res = await getArticleVersion(id, versionNo);
      if (!mountedRef.current) {
        return;
      }
      setSelectedVersion(res.data);
    } catch (e) {
      if (!mountedRef.current) {
        return;
      }
      setError(getErrorMessage(e, t('articleVersion.detailLoadError')));
    } finally {
      if (mountedRef.current) {
        setDetailLoading(false);
      }
    }
  };

  const handleRestore = async (versionNo: number) => {
    if (!id) {
      return;
    }
    const ok = await confirm({
      title: formatText(t('articleVersion.confirmRestore'), { versionNo }),
      confirmLabel: t('articleVersion.restore'),
      cancelLabel: t('common.cancel'),
      tone: 'danger',
    });
    if (!ok) {
      return;
    }
    setBusyVersion(versionNo);
    setError('');
    try {
      await restoreArticleVersion(id, versionNo);
      if (!mountedRef.current) {
        return;
      }
      await fetchVersions();
      if (!mountedRef.current) {
        return;
      }
      await handleInspect(versionNo);
    } catch (e) {
      if (!mountedRef.current) {
        return;
      }
      setError(getErrorMessage(e, t('articleVersion.restoreError')));
    } finally {
      if (mountedRef.current) {
        setBusyVersion(null);
      }
    }
  };

  return (
    <div>
      <div className="mb-8 flex flex-col gap-3 border-b border-mountain-grey pb-4 sm:flex-row sm:items-center sm:justify-between">
        <h3 className="text-2xl font-bold tracking-widest text-ink">{t('articleVersion.title')}</h3>
        {id && <Link to={`/admin/editor/${id}`} className="text-sm tracking-widest text-ochre">{t('articleVersion.backToEditor')}</Link>}
      </div>

      <InlineNotice message={error} className="mb-6" />

      {loading ? (
        <TableSkeleton rows={5} cols={5} />
      ) : (
        <>
          <table className="admin-responsive-table w-full border-collapse text-left text-sm">
            <thead>
              <tr className="border-b border-mountain-grey text-ink-light">
                <th className="py-3 font-normal">{t('articleVersion.version')}</th>
                <th className="py-3 font-normal">{t('articleVersion.articleTitle')}</th>
                <th className="py-3 font-normal">{t('articleVersion.status')}</th>
                <th className="py-3 font-normal">{t('articleVersion.savedAt')}</th>
                <th className="py-3 text-right font-normal">{t('articleVersion.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {versions.length === 0 && (
                <tr>
                  <td colSpan={5} className="py-10 text-center text-ink-light">{t('articleVersion.empty')}</td>
                </tr>
              )}
              {versions.map((version) => (
                <tr key={version.id} className="border-b border-mountain-grey/50">
                  <td data-label={t('articleVersion.version')} className="py-4">#{version.versionNo}</td>
                  <td data-label={t('articleVersion.articleTitle')} className="admin-card-title py-4 font-bold">{version.title}</td>
                  <td data-label={t('articleVersion.status')} className="py-4 text-ink-light">{getArticleStatusLabel(language, version.status)}</td>
                  <td data-label={t('articleVersion.savedAt')} className="py-4 text-ink-light">{new Date(version.createdAt).toLocaleString(getDateLocale(language))}</td>
                  <td data-label={t('articleVersion.actions')} className="admin-card-actions py-4 text-right">
                    <div className="admin-action-list">
                      <button
                        type="button"
                        disabled={detailLoading}
                        onClick={() => handleInspect(version.versionNo)}
                        className="text-ink-light hover:text-ochre disabled:opacity-50"
                      >
                        {t('articleVersion.inspect')}
                      </button>
                      <button
                        type="button"
                        disabled={busyVersion === version.versionNo}
                        onClick={() => handleRestore(version.versionNo)}
                        className="text-ochre disabled:opacity-50"
                      >
                        {t('articleVersion.restore')}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          <Pagination currentPage={page} total={total} pageSize={size} onPageChange={setPage} />
        </>
      )}

      {selectedVersion && (
        <section className="mt-10 border border-mountain-grey bg-[var(--paper-soft)] p-5">
          <div className="mb-5 flex flex-wrap items-start justify-between gap-3 border-b border-mountain-grey pb-4">
            <div>
              <p className="text-xs tracking-widest text-ink-light">{t('articleVersion.version')} #{selectedVersion.versionNo}</p>
              <h4 className="mt-2 text-xl font-bold text-ink">{selectedVersion.title}</h4>
              <p className="mt-2 text-sm text-ink-light">{selectedVersion.slug}</p>
            </div>
            <span className="border border-mountain-grey px-2 py-1 text-xs text-ink-light">{getArticleStatusLabel(language, selectedVersion.status)}</span>
          </div>

          <div className="grid gap-5 text-sm md:grid-cols-2">
            <div>
              <p className="mb-2 text-xs tracking-widest text-ink-light">{t('articleVersion.summary')}</p>
              <p className="leading-relaxed text-ink">{selectedVersion.summary || '-'}</p>
            </div>
            <div>
              <p className="mb-2 text-xs tracking-widest text-ink-light">SEO</p>
              <p className="leading-relaxed text-ink">{selectedVersion.seoTitle || '-'}</p>
              <p className="mt-1 leading-relaxed text-ink-light">{selectedVersion.seoDescription || '-'}</p>
              <p className="mt-1 text-xs text-ink-light">{selectedVersion.seoKeywords || '-'}</p>
            </div>
          </div>

          <div className="prose prose-stone mt-6 max-w-none border-t border-mountain-grey pt-5 font-serif">
            <MarkdownRenderer content={selectedVersion.content || ''} />
          </div>
        </section>
      )}
    </div>
  );
};

export default ArticleVersions;
