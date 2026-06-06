import { useEffect } from 'react';
import { useSiteSettingsStore } from '../store/siteSettings';

const upsertMeta = (selector: string, createAttrs: Record<string, string>, content: string) => {
  let element = document.head.querySelector<HTMLMetaElement>(selector);
  if (!element) {
    element = document.createElement('meta');
    Object.entries(createAttrs).forEach(([key, value]) => element?.setAttribute(key, value));
    document.head.appendChild(element);
  }
  element.setAttribute('content', content);
};

export const useSEO = (title?: string, description?: string, keywords?: string, enabled = true) => {
  const siteTitle = useSiteSettingsStore((state) => state.siteTitle);
  const siteDescription = useSiteSettingsStore((state) => state.siteDescription);
  const siteKeywords = useSiteSettingsStore((state) => state.siteKeywords);

  useEffect(() => {
    if (!enabled) {
      return;
    }

    const finalTitle = title ? `${title} | ${siteTitle}` : siteTitle;
    const finalDescription = description || siteDescription;
    const finalKeywords = keywords || siteKeywords;

    document.title = finalTitle;
    upsertMeta('meta[name="description"]', { name: 'description' }, finalDescription);
    upsertMeta('meta[name="keywords"]', { name: 'keywords' }, finalKeywords);
    upsertMeta('meta[property="og:title"]', { property: 'og:title' }, finalTitle);
    upsertMeta('meta[property="og:description"]', { property: 'og:description' }, finalDescription);
  }, [description, enabled, keywords, siteDescription, siteKeywords, siteTitle, title]);
};
