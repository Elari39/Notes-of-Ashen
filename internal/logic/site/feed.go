package site

import (
	"context"
	"encoding/xml"
	"strconv"
	"strings"
	"time"

	"notes-of-ashen/internal/svc"
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
	baseURL := effectiveBaseURL(settings.SiteBaseURL, requestBaseURL)
	articles, err := svcCtx.Store.ListPublicArticles(ctx, 50)
	if err != nil {
		return nil, err
	}
	items := make([]rssItem, 0, len(articles))
	if settings.ResumePageEnabled {
		content, err := cachedResumePageContent(ctx, svcCtx)
		if err != nil {
			return nil, err
		}
		items = append(items, rssItem{
			Title:       content.Title,
			Link:        baseURL + "/resume",
			GUID:        baseURL + "/resume",
			Description: content.Subtitle,
		})
	}
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
	baseURL := effectiveBaseURL(settings.SiteBaseURL, requestBaseURL)
	articles, err := svcCtx.Store.ListPublicArticles(ctx, 100)
	if err != nil {
		return nil, err
	}
	urls := []sitemapURL{
		{Loc: baseURL + "/"},
		{Loc: baseURL + "/archive"},
		{Loc: baseURL + "/search"},
	}
	if settings.ResumePageEnabled {
		urls = append(urls, sitemapURL{Loc: baseURL + "/resume"})
	}
	if settings.ProjectsPageEnabled {
		urls = append(urls, sitemapURL{Loc: baseURL + "/projects"})
	}
	for _, article := range articles {
		urls = append(urls, sitemapURL{
			Loc:     articleURL(baseURL, article.ID),
			LastMod: article.UpdatedAt.Format("2006-01-02"),
		})
	}
	body, err := xml.MarshalIndent(sitemapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), body...), nil
}

func effectiveBaseURL(setting, requestBaseURL string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(setting), "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(strings.TrimSpace(requestBaseURL), "/")
	}
	return baseURL
}

func articleURL(baseURL string, id uint64) string {
	return baseURL + "/article/" + strconv.FormatUint(id, 10)
}
