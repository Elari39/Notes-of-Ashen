import React, { useEffect, useRef, useState } from 'react';
import Modal from '../ui/Modal';
import Button from '../ui/Button';
import TextField from '../ui/TextField';
import InlineNotice from '../InlineNotice';
import PagePendingState from '../RoutePending';
import { getMediaAssets, uploadMedia } from '../../api/media';
import type { MediaAsset } from '../../types';
import { usePreferenceStore } from '../../store/preferences';
import { translate } from '../../i18n';
import { getErrorMessage } from '../../utils/error';

type Props = {
  open: boolean;
  onOpenChange(next: boolean): void;
  onSelect(asset: MediaAsset): void;
};

const MediaPicker: React.FC<Props> = ({ open, onOpenChange, onSelect }) => {
  const language = usePreferenceStore((state) => state.language);
  const [items, setItems] = useState<MediaAsset[]>([]);
  const [query, setQuery] = useState('');
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  useEffect(() => {
    if (!open) return;
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      setLoading(true);
      setError('');
      getMediaAssets({ page: 1, size: 50, ...(query.trim() ? { q: query.trim() } : {}) }, controller.signal)
        .then((response) => setItems(response.data.items || []))
        .catch((err) => {
          if (!controller.signal.aborted) {
            setError(getErrorMessage(err, translate(language, 'media.loadError')));
          }
        })
        .finally(() => {
          if (!controller.signal.aborted) {
            setLoading(false);
          }
        });
    }, query ? 200 : 0);
    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [language, open, query]);

  const handleUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    setUploading(true);
    setError('');
    try {
      const response = await uploadMedia(file, file.name.replace(/\.[^.]+$/, ''));
      setItems((current) => [response.data, ...current.filter((item) => item.id !== response.data.id)]);
    } catch (err) {
      setError(getErrorMessage(err, t('media.uploadError')));
    } finally {
      setUploading(false);
      event.target.value = '';
    }
  };

  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      title={t('media.title')}
      description={t('media.subtitle')}
      size="lg"
      closeLabel={t('common.dismiss')}
    >
      <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0 flex-1">
          <TextField value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t('media.search')} />
        </div>
        <input
          data-testid="media-picker-upload-input"
          ref={inputRef}
          className="sr-only"
          type="file"
          accept="image/jpeg,image/png,image/gif,image/webp"
          onChange={handleUpload}
        />
        <Button type="button" size="sm" loading={uploading} onClick={() => inputRef.current?.click()}>
          {uploading ? t('media.uploading') : t('media.upload')}
        </Button>
      </div>

      <InlineNotice message={error} className="mb-4" />
      {loading ? (
        <PagePendingState variant="inline" label={t('common.loading')} />
      ) : items.length === 0 ? (
        <p className="py-12 text-center">{t('media.empty')}</p>
      ) : (
        <div className="grid max-h-[55vh] grid-cols-2 gap-3 overflow-y-auto pr-1 sm:grid-cols-3">
          {items.map((item) => (
            <button
              data-testid={`media-picker-item-${item.id}`}
              key={item.id}
              type="button"
              onClick={() => {
                onSelect(item);
                onOpenChange(false);
              }}
              className="group overflow-hidden rounded-lg border border-hairline bg-surface-soft text-left focus-visible:outline focus-visible:outline-2 focus-visible:outline-ochre"
            >
              <img src={item.url} alt={item.altText} className="aspect-video w-full object-cover" loading="lazy" />
              <span className="block truncate px-3 py-2 text-xs text-ink group-hover:text-ochre">{item.originalName}</span>
            </button>
          ))}
        </div>
      )}
    </Modal>
  );
};

export default MediaPicker;
