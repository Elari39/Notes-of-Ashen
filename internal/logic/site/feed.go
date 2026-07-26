package site

import (
	"context"
	"encoding/xml"
	"strconv"
	"strings"
	"time"

	"notes-of-ashen/internal/svc"
	"notes-of-ashen/model"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	rssArticleLimit = 50
	maxSitemapURLs  = 50000
	baseSitemapURLs = 3
)

type rssDocument struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	Description string `xml:"description,omitempty"`
	PubDate     string `xml:"pubDate,omitempty"`
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

func RSS(ctx context.Context, svcCtx *svc.ServiceContext, requestBaseURL string) ([]byte, error) {
	settings, err := cachedSiteSettings(ctx, svcCtx)
	if err != nil {
		return nil, err
	}
	baseURL := effectiveBaseURL(ctx, "rss", settings.SiteBaseURL, requestBaseURL)
	articles, err := svcCtx.Store.ListPublicArticleEntries(ctx, rssArticleLimit)
	if err != nil {
		return nil, err
	}
	items := make([]rssItem, 0, len(articles))
	if settings.ProjectsPageEnabled {
		content, err := cachedProjectsPageContent(ctx, svcCtx)
		if err != nil {
			return nil, err
		}
		items = append(items, rssItem{
			Title:       content.Title,
			Link:        baseURL + "/projects",
			GUID:        baseURL + "/projects",
			Description: content.Subtitle,
		})
	}
	for _, article := range articles {
		link := articleURL(baseURL, article.ID)
		pubDate := article.CreatedAt
		if article.PublishedAt != nil {
			pubDate = *article.PublishedAt
		}
		items = append(items, rssItem{
			Title:       article.Title,
			Link:        link,
			GUID:        link,
			Description: article.Summary,
			PubDate:     pubDate.Format(time.RFC1123Z),
		})
	}
	body, err := xml.MarshalIndent(rssDocument{
		Version: "2.0",
		Channel: rssChannel{
			Title:       settings.SiteTitle,
			Link:        baseURL,
			Description: settings.SiteDescription,
			Items:       items,
		},
	}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), body...), nil
}

func Sitemap(ctx context.Context, svcCtx *svc.ServiceContext, requestBaseURL string) ([]byte, error) {
	settings, err := cachedSiteSettings(ctx, svcCtx)
	if err != nil {
		return nil, err
	}
	baseURL := effectiveBaseURL(ctx, "sitemap", settings.SiteBaseURL, requestBaseURL)
	articleLimit := maxSitemapURLs - baseSitemapURLs
	if settings.ProjectsPageEnabled {
		articleLimit--
	}
	articles, err := svcCtx.Store.ListPublicArticleEntries(ctx, articleLimit)
	if err != nil {
		return nil, err
	}
	urls := sitemapURLs(baseURL, settings.ProjectsPageEnabled, articles)
	body, err := xml.MarshalIndent(sitemapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), body...), nil
}

func sitemapURLs(baseURL string, projectsPageEnabled bool, articles []model.PublicArticleEntry) []sitemapURL {
	urls := []sitemapURL{
		{Loc: baseURL + "/"},
		{Loc: baseURL + "/archive"},
		{Loc: baseURL + "/search"},
	}
	if projectsPageEnabled {
		urls = append(urls, sitemapURL{Loc: baseURL + "/projects"})
	}
	for _, article := range articles {
		if len(urls) >= maxSitemapURLs {
			break
		}
		urls = append(urls, sitemapURL{
			Loc:     articleURL(baseURL, article.ID),
			LastMod: article.UpdatedAt.Format("2006-01-02"),
		})
	}
	return urls
}

func effectiveBaseURL(ctx context.Context, feature, setting, requestBaseURL string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(setting), "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(strings.TrimSpace(requestBaseURL), "/")
		// 站点基址未配置时回退请求 Host；生产环境应在站点设置中配置正式 HTTPS 域名，
		// 否则搜索引擎和订阅客户端可能收到不可公开访问的绝对链接。
		logx.WithContext(ctx).Infof("siteBaseUrl is empty, %s falls back to request base url %s; configure a public https siteBaseUrl for production", feature, baseURL)
	}
	return baseURL
}

func articleURL(baseURL string, id uint64) string {
	return baseURL + "/article/" + strconv.FormatUint(id, 10)
}
