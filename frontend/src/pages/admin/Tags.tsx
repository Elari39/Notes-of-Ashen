import React, { useCallback, useEffect, useRef, useState } from 'react';
import { getAdminTags, createTag, deleteTag, updateTag } from '../../api/tag';
import { Tag } from '../../types';
import Pagination from '../../components/Pagination';
import InlineNotice from '../../components/InlineNotice';
import PagePendingState from '../../components/RoutePending';
import TableSkeleton from '../../components/ui/TableSkeleton';
import EmptyState from '../../components/ui/EmptyState';
import Button from '../../components/ui/Button';
import { getErrorMessage } from '../../utils/error';
import { formatText, translate } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';
import { useConfirm } from '../../hooks/useConfirm';
import { MAX_TEXT_FIELD_BYTES, utf8ByteLength } from '../../utils/utf8';

const AdminTags: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const confirm = useConfirm();
  const [tags, setTags] = useState<Tag[]>([]);
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [description, setDescription] = useState('');
  const [editingId, setEditingId] = useState<number | null>(null);

  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [busyId, setBusyId] = useState<number | null>(null);
  const listRequestRef = useRef(0);
  const size = 10;
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  const fetchList = useCallback(async () => {
    const requestId = listRequestRef.current + 1;
    listRequestRef.current = requestId;
    setLoading(true);
    setError('');
    try {
      const res = await getAdminTags({ page, size });
      if (listRequestRef.current !== requestId) {
        return;
      }
      setTags(res.data.items || []);
      setTotal(res.data.total || 0);
    } catch (e) {
      if (listRequestRef.current === requestId) {
        setError(getErrorMessage(e, translate(language, 'taxonomy.listTagError')));
      }
    } finally {
      if (listRequestRef.current === requestId) {
        setLoading(false);
      }
    }
  }, [page, size, language]);

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (utf8ByteLength(description.trim()) > MAX_TEXT_FIELD_BYTES) {
      setError(formatText(t('common.textTooLarge'), { limit: MAX_TEXT_FIELD_BYTES }));
      return;
    }
    setError('');
    setSubmitting(true);
    try {
      if (editingId) {
        await updateTag(editingId, { name, slug, description });
      } else {
        await createTag({ name, slug, description });
      }
      handleCancel();
      await fetchList();
    } catch (e: unknown) {
      setError(getErrorMessage(e, t('taxonomy.submitError')));
    } finally {
      setSubmitting(false);
    }
  };

  const handleEdit = (tag: Tag) => {
    setEditingId(tag.id);
    setName(tag.name);
    setSlug(tag.slug);
    setDescription(tag.description);
  };

  const handleCancel = () => {
    setEditingId(null);
    setName('');
    setSlug('');
    setDescription('');
  };

  const handleDelete = async (id: number) => {
    const ok = await confirm({
      title: t('taxonomy.confirmDelete'),
      confirmLabel: t('common.delete'),
      cancelLabel: t('common.cancel'),
      tone: 'danger',
    });
    if (!ok) return;
    setError('');
    setBusyId(id);
    try {
      await deleteTag(id);
      await fetchList();
    } catch (e: unknown) {
      setError(getErrorMessage(e, t('taxonomy.deleteTagError')));
    } finally {
      setBusyId(null);
    }
  };

  return (
    <div>
      <div className="mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold text-ink tracking-widest">{t('admin.tags')}</h3>
      </div>

      <InlineNotice message={error} className="mb-6" />

      <form onSubmit={handleSubmit} className="mb-8 flex flex-col md:flex-row gap-4 md:items-end">
        <div className="flex-1">
          <input type="text" placeholder={t('common.name')} required value={name} onChange={e => setName(e.target.value)} className="w-full bg-transparent border-b border-mountain-grey py-2 focus:outline-hidden focus:border-ochre text-ink" />
        </div>
        <div className="flex-1">
          <input type="text" placeholder="Slug" required value={slug} onChange={e => setSlug(e.target.value)} className="w-full bg-transparent border-b border-mountain-grey py-2 focus:outline-hidden focus:border-ochre text-ink" />
        </div>
        <div className="flex-1">
          <input type="text" placeholder={t('common.description')} value={description} onChange={e => setDescription(e.target.value)} className="w-full bg-transparent border-b border-mountain-grey py-2 focus:outline-hidden focus:border-ochre text-ink" />
        </div>
        <div className="flex flex-wrap gap-2">
          <Button type="submit" variant="primary" size="sm" loading={submitting}>
            {submitting ? t('common.processing') : editingId ? t('common.save') : t('taxonomy.add')}
          </Button>
          {editingId && (
            <Button type="button" variant="ghost" size="sm" onClick={handleCancel}>
              {t('common.cancel')}
            </Button>
          )}
        </div>
      </form>

      {loading && tags.length === 0 && (
        <TableSkeleton rows={5} cols={4} />
      )}
      {loading && tags.length > 0 && (
        <PagePendingState variant="inline" label={t('common.loading')} />
      )}
      {!loading && tags.length === 0 ? (
        <EmptyState illustration="leaf" title={t('common.empty')} />
      ) : tags.length > 0 ? (
        <>
          <table className="admin-responsive-table w-full text-left border-collapse text-sm">
            <thead>
              <tr className="border-b border-mountain-grey text-ink-light opacity-80 tracking-widest">
                <th className="py-3 font-normal">{t('common.name')}</th>
                <th className="py-3 font-normal">Slug</th>
                <th className="py-3 font-normal">{t('taxonomy.articleCount')}</th>
                <th className="py-3 font-normal text-right">{t('common.action')}</th>
              </tr>
            </thead>
            <tbody>
              {tags.map(tag => (
                <tr key={tag.id} className="border-b border-mountain-grey/50 hover:bg-mountain-grey/20 transition-colors text-ink">
                  <td data-label={t('common.name')} className="admin-card-title py-4 font-bold relative before:content-['#'] before:mr-1 before:opacity-30">{tag.name}</td>
                  <td data-label="Slug" className="py-4 text-ink-light">{tag.slug}</td>
                  <td data-label={t('taxonomy.articleCount')} className="py-4 text-ink-light">{tag.articleCount}</td>
                  <td data-label={t('common.action')} className="admin-card-actions py-4 text-right">
                    <div className="admin-action-list">
                      <button onClick={() => handleEdit(tag)} className="text-ink opacity-80 hover:text-ochre hover:opacity-100 tracking-wider">{t('common.edit')}</button>
                      <button onClick={() => handleDelete(tag.id)} disabled={busyId === tag.id} className="text-ember opacity-80 hover:opacity-100 tracking-wider disabled:opacity-50 disabled:cursor-not-allowed transition-opacity duration-fast">{t('common.delete')}</button>
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
      ) : null}
    </div>
  );
};

export default AdminTags;
