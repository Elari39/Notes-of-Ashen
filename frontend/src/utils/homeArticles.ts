/** Preserve server ordering and pagination; promote only on the unfiltered first page. */
export function selectHomeArticles<T extends { id: number }>(articles: T[], promote: boolean) {
  const featuredArticle = promote ? articles[0] : undefined;
  return {
    featuredArticle,
    listArticles: featuredArticle ? articles.filter((article) => article.id !== featuredArticle.id) : articles,
  };
}
