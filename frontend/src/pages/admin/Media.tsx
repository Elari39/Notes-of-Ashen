import React, { useCallback, useEffect, useRef, useState } from 'react';
import { deleteMedia, getMediaAssets, updateMediaAlt, uploadMedia } from '../../api/media';
import type { MediaAsset } from '../../types';
import { usePreferenceStore } from '../../store/preferences';
import { useAuthStore } from '../../store/auth';
import { translate } from '../../i18n';
import { getErrorMessage } from '../../utils/error';
import { useConfirm } from '../../hooks/useConfirm';
import Button from '../../components/ui/Button';
import TextField from '../../components/ui/TextField';
import InlineNotice from '../../components/InlineNotice';
import PagePendingState from '../../components/RoutePending';
import Pagination from '../../components/Pagination';
import EmptyState from '../../components/ui/EmptyState';
import { toast } from '../../utils/notify';

const pageSize = 18;

const AdminMedia: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const user = useAuthStore((state) => state.user);
  const confirm = useConfirm();
  const [items, setItems] = useState<MediaAsset[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [query, setQuery] = useState('');
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<number | 'upload' | null>(null);
  const [error, setError] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  const requestRef = useRef(0);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  const load = useCallback(async () => {
    const requestID = requestRef.current + 1;
    requestRef.current = requestID;
    setLoading(true);
    setError('');
    try {
      const response = await getMediaAssets({
        page,
        size: pageSize,
        ...(query.trim() ? { q: query.trim() } : {}),
      });
      if (requestRef.current !== requestID) return;
      setItems(response.data.items || []);
      setTotal(response.data.total || 0);
    } catch (err) {
      if (requestRef.current === requestID) {
        setError(getErrorMessage(err, translate(language, 'media.loadError')));
      }
    } finally {
      if (requestRef.current === requestID) {
        setLoading(false);
      }
    }
  }, [language, page, query]);

  useEffect(() => {
    void load();
  }, [load]);

  const handleUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    setBusy('upload');
    setError('');
    try {
      await uploadMedia(file, file.name.replace(/\.[^.]+$/, ''));
      await load();
    } catch (err) {
      setError(getErrorMessage(err, t('media.uploadError')));
    } finally {
      setBusy(null);
      event.target.value = '';
    }
  };

  const saveAlt = async (item: MediaAsset, altText: string) => {
    setBusy(item.id);
    setError('');
    try {
      const response = await updateMediaAlt(item.id, altText);
      setItems((current) => current.map((value) => value.id === item.id ? response.data : value));
    } catch (err) {
      setError(getErrorMessage(err, t('media.updateError')));
    } finally {
      setBusy(null);
    }
  };

  const remove = async (item: MediaAsset) => {
    const accepted = await confirm({ title: t('media.confirmDelete'), tone: 'danger' });
    if (!accepted) return;
    setBusy(item.id);
    setError('');
    try {
      await deleteMedia(item.id);
      await load();
    } catch (err) {
      setError(getErrorMessage(err, t('media.deleteError')));
    } finally {
      setBusy(null);
    }
  };

  return (
    <div>
      <header className="mb-8 flex flex-col gap-4 border-b border-hairline pb-5 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="editorial-kicker">{t('admin.media')}</p>
          <h3 className="mt-3 text-4xl text-ink">{t('media.title')}</h3>
          <p className="mt-2 text-sm text-muted">{t('media.subtitle')}</p>
        </div>
        <div>
          <input
            ref={inputRef}
            className="sr-only"
            type="file"
            accept="image/jpeg,image/png,image/gif,image/webp"
            onChange={handleUpload}
          />
          <Button loading={busy === 'upload'} onClick={() => inputRef.current?.click()}>
            {busy === 'upload' ? t('media.uploading') : t('media.upload')}
          </Button>
        </div>
      </header>

      <div className="mb-6 max-w-lg">
        <TextField
          value={query}
          onChange={(event) => {
            setQuery(event.target.value);
            setPage(1);
          }}
          placeholder={t('media.search')}
        />
      </div>
      <InlineNotice message={error} className="mb-5" />

      {loading && items.length === 0 ? (
        <PagePendingState variant="admin" label={t('common.loading')} />
      ) : items.length === 0 ? (
        <EmptyState illustration="leaf" title={t('media.empty')} />
      ) : (
        <>
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {items.map((item) => (
              <MediaCard
                key={item.id}
                item={item}
                busy={busy === item.id}
                canDelete={user?.role === 'admin'}
                t={t}
                onSave={saveAlt}
                onDelete={remove}
              />
            ))}
          </div>
          <Pagination currentPage={page} total={total} pageSize={pageSize} onPageChange={setPage} />
        </>
      )}
    </div>
  );
};

const MediaCard = ({ item, busy, canDelete, t, onSave, onDelete }: {
  item: MediaAsset;
  busy: boolean;
  canDelete: boolean;
  t: (key: Parameters<typeof translate>[1]) => string;
  onSave(item: MediaAsset, alt: string): Promise<void>;
  onDelete(item: MediaAsset): Promise<void>;
}) => {
  const [alt, setAlt] = useState(item.altText);

  useEffect(() => {
    setAlt(item.altText);
  }, [item.altText]);

  return (
    <article className="overflow-hidden rounded-lg border border-hairline bg-paper shadow-xs">
      <img src={item.url} alt={item.altText} className="aspect-video w-full object-cover" loading="lazy" />
      <div className="space-y-3 p-4">
        <p className="truncate font-medium text-ink">{item.originalName}</p>
        <p className="text-xs text-muted">{Math.ceil(item.sizeBytes / 1024)} KB · {item.width}×{item.height}</p>
        <TextField value={alt} onChange={(event) => setAlt(event.target.value)} placeholder={t('media.alt')} />
        <div className="flex flex-wrap gap-2">
          <Button size="sm" variant="ghost" onClick={() => void copyMediaURL(item.url)}>{t('media.copy')}</Button>
          <Button size="sm" disabled={alt === item.altText} loading={busy} onClick={() => void onSave(item, alt)}>{t('common.save')}</Button>
          {canDelete && (
            <Button size="sm" variant="danger" disabled={busy} onClick={() => void onDelete(item)}>{t('common.delete')}</Button>
          )}
        </div>
      </div>
    </article>
  );
};

const copyMediaURL = async (url: string) => {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(url);
    } else {
      const input = document.createElement('textarea');
      input.value = url;
      input.style.position = 'fixed';
      input.style.opacity = '0';
      document.body.appendChild(input);
      input.select();
      const copied = document.execCommand('copy');
      input.remove();
      if (!copied) throw new Error('copy failed');
    }
    toast.success('share.copied');
  } catch {
    toast.error('share.failed');
  }
};

export default AdminMedia;
