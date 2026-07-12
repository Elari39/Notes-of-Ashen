export type ArticleBaselineStatus = 'loading' | 'error' | 'ready';

export type ArticleEditorAccess = 'editor' | 'loading' | 'error';

export const canSaveAISettings = (hasLoaded: boolean): boolean => hasLoaded;

export const canOperateArticleEditor = (
  isEditingExistingArticle: boolean,
  baselineStatus: ArticleBaselineStatus,
): boolean => !isEditingExistingArticle || baselineStatus === 'ready';

export const resolveArticleEditorAccess = (
  isEditingExistingArticle: boolean,
  baselineStatus: ArticleBaselineStatus,
): ArticleEditorAccess => {
  if (canOperateArticleEditor(isEditingExistingArticle, baselineStatus)) {
    return 'editor';
  }

  return baselineStatus === 'error' ? 'error' : 'loading';
};

export const executeAISettingsUpdate = async <T>(
  hasLoaded: boolean,
  request: () => Promise<T>,
): Promise<T> => {
  if (!canSaveAISettings(hasLoaded)) {
    throw new Error('AI settings are not loaded yet');
  }

  return request();
};

export const executeArticleEditorOperation = async <T>(
  isEditingExistingArticle: boolean,
  baselineStatus: ArticleBaselineStatus,
  request: () => Promise<T>,
): Promise<T> => {
  if (!canOperateArticleEditor(isEditingExistingArticle, baselineStatus)) {
    throw new Error('Article baseline is not loaded yet');
  }

  return request();
};
