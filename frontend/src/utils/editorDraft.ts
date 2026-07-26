import { safeLocalStorage } from './storage.ts';

const editorDraftPrefix = 'article-editor:draft:user:';

export const editorDraftKey = (id: string | number | false | undefined, userId?: number): string | null => {
  if (!userId || userId <= 0) {
    return null;
  }
  return `${editorDraftPrefix}${userId}:${id || 'new'}`;
};

export const clearEditorDraftsForUser = (userId?: number): void => {
  if (!userId || userId <= 0) {
    return;
  }
  safeLocalStorage.removeByPrefix(`${editorDraftPrefix}${userId}:`);
};
