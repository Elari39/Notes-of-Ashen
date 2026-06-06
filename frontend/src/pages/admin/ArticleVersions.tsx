import React, { useCallback, useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import ReactMarkdown from 'react-markdown';
import { getArticleVersion, getArticleVersions, restoreArticleVersion } from '../../api/article';
import InlineNotice from '../../components/InlineNotice';
import Pagination from '../../components/Pagination';
import { getErrorMessage } from '../../utils/error';
import type { ArticleVersion } from '../../types';

const ArticleVersions: React.FC = () => {
  const { id } = useParams();
  const [versions, setVersions] = useState<ArticleVersion[]>([]);
  const [selectedVersion, setSelectedVersion] = useState<ArticleVersion | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);
  const [error, setError] = useState('');
  const [busyVersion, setBusyVersion] = useState<number | null>(null);
  const size = 10;

  const fetchVersions = useCallback(async () => {
    if (!id) {
      setLoading(false);
      return;
    }
    setLoading(true);
    setError('');
    try {
      const res = await getArticleVersions(id, { page, size });
      setVersions(res.data.items || []);
      setTotal(res.data.total || 0);
    } catch (e) {
      setError(getErrorMessage(e, '版本历史加载失败'));
    } finally {
      setLoading(false);
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
      setSelectedVersion(res.data);
    } catch (e) {
      setError(getErrorMessage(e, '版本详情加载失败'));
    } finally {
      setDetailLoading(false);
    }
  };

  const handleRestore = async (versionNo: number) => {
    if (!id || !confirm(`确认恢复到版本 #${versionNo}？当前内容会先保存为新的历史版本。`)) {
      return;
    }
    setBusyVersion(versionNo);
    setError('');
    try {
      await restoreArticleVersion(id, versionNo);
      await fetchVersions();
      await handleInspect(versionNo);
    } catch (e) {
      setError(getErrorMessage(e, '版本恢复失败'));
    } finally {
      setBusyVersion(null);
    }
  };

  return (
    <div>
      <div className="mb-8 flex items-center justify-between border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold tracking-widest text-ink">版本历史</h3>
        {id && <Link to={`/admin/editor/${id}`} className="text-sm tracking-widest text-ochre">返回编辑</Link>}
      </div>

      <InlineNotice message={error} className="mb-6" />

      {loading ? (
        <div className="py-16 text-center tracking-widest text-ink-light">版本加载中...</div>
      ) : (
        <>
          <table className="w-full border-collapse text-left text-sm">
            <thead>
              <tr className="border-b border-mountain-grey text-ink-light">
                <th className="py-3 font-normal">版本</th>
                <th className="py-3 font-normal">标题</th>
                <th className="py-3 font-normal">状态</th>
                <th className="py-3 font-normal">保存时间</th>
                <th className="py-3 text-right font-normal">操作</th>
              </tr>
            </thead>
            <tbody>
              {versions.length === 0 && (
                <tr>
                  <td colSpan={5} className="py-10 text-center text-ink-light">暂无历史版本</td>
                </tr>
              )}
              {versions.map((version) => (
                <tr key={version.id} className="border-b border-mountain-grey border-opacity-50">
                  <td className="py-4">#{version.versionNo}</td>
                  <td className="py-4 font-bold">{version.title}</td>
                  <td className="py-4 text-ink-light">{version.status}</td>
                  <td className="py-4 text-ink-light">{new Date(version.createdAt).toLocaleString()}</td>
                  <td className="space-x-4 py-4 text-right">
                    <button
                      type="button"
                      disabled={detailLoading}
                      onClick={() => handleInspect(version.versionNo)}
                      className="text-ink-light hover:text-ochre disabled:opacity-50"
                    >
                      查看
                    </button>
                    <button
                      type="button"
                      disabled={busyVersion === version.versionNo}
                      onClick={() => handleRestore(version.versionNo)}
                      className="text-ochre disabled:opacity-50"
                    >
                      恢复
                    </button>
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
              <p className="text-xs tracking-widest text-ink-light">版本 #{selectedVersion.versionNo}</p>
              <h4 className="mt-2 text-xl font-bold text-ink">{selectedVersion.title}</h4>
              <p className="mt-2 text-sm text-ink-light">{selectedVersion.slug}</p>
            </div>
            <span className="border border-mountain-grey px-2 py-1 text-xs text-ink-light">{selectedVersion.status}</span>
          </div>

          <div className="grid gap-5 text-sm md:grid-cols-2">
            <div>
              <p className="mb-2 text-xs tracking-widest text-ink-light">摘要</p>
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
            <ReactMarkdown>{selectedVersion.content || ''}</ReactMarkdown>
          </div>
        </section>
      )}
    </div>
  );
};

export default ArticleVersions;
