package model

import (
	"context"
	"errors"
	"strings"
)

// schemaTableRequirement 描述当前运行代码依赖的一张数据表及其字段。
// 这里只校验运行所需的表和字段，不将单纯的性能索引作为 readiness 条件。
type schemaTableRequirement struct {
	Name    string
	Columns []string
}

// runtimeSchemaManifest 是当前 Model 与运行期事件写入所需的最小数据库结构。
// 需要新增或移除查询字段时，应同步维护这里，避免连接可用但功能在运行期才因
// 缺少迁移而失败。
var runtimeSchemaManifest = [...]schemaTableRequirement{
	{
		Name: "users",
		Columns: []string{
			"id", "account", "password_hash", "email", "avatar_url", "nickname", "role", "status", "created_at", "updated_at",
		},
	},
	{
		Name:    "site_settings",
		Columns: []string{"setting_key", "setting_value", "created_at", "updated_at"},
	},
	{
		Name:    "categories",
		Columns: []string{"id", "name", "slug", "description", "created_by", "created_at", "updated_at"},
	},
	{
		Name:    "tags",
		Columns: []string{"id", "name", "slug", "description", "created_by", "created_at", "updated_at"},
	},
	{
		Name: "projects",
		Columns: []string{
			"id", "title", "summary", "role", "period", "cover_url", "demo_url", "repo_url", "content_markdown", "featured", "display_order",
		},
	},
	{
		Name:    "project_tags",
		Columns: []string{"project_id", "tag_id"},
	},
	{
		Name: "articles",
		Columns: []string{
			"id", "author_id", "category_id", "title", "slug", "summary", "content", "cover_url", "status", "view_count", "like_count", "scheduled_at", "published_at", "is_pinned", "display_priority", "seo_title", "seo_description", "seo_keywords", "created_at", "updated_at",
		},
	},
	{
		Name: "article_versions",
		Columns: []string{
			"id", "article_id", "version_no", "changed_by", "author_id", "category_id", "title", "slug", "summary", "content", "cover_url", "status", "view_count", "like_count", "scheduled_at", "published_at", "is_pinned", "display_priority", "seo_title", "seo_description", "seo_keywords", "tag_ids", "original_created_at", "original_updated_at", "created_at",
		},
	},
	{
		Name:    "article_tags",
		Columns: []string{"article_id", "tag_id"},
	},
	{
		Name:    "article_likes",
		Columns: []string{"article_id", "visitor_hash", "created_at"},
	},
	{
		Name:    "refresh_tokens",
		Columns: []string{"id", "user_id", "token_hash", "expires_at", "revoked_at", "created_at"},
	},
	{
		Name: "operation_logs",
		Columns: []string{
			"id", "user_id", "event_type", "resource_type", "resource_id", "metadata", "ip", "user_agent", "created_at",
		},
	},
	{
		Name:    "traffic_daily_stats",
		Columns: []string{"stat_date", "pv", "uv"},
	},
	{
		Name:    "traffic_daily_visitors",
		Columns: []string{"stat_date", "visitor_hash"},
	},
	{
		Name:    "traffic_referer_stats",
		Columns: []string{"stat_date", "article_id", "source_type", "source_name", "pv"},
	},
	{
		Name: "media_assets",
		Columns: []string{
			"id", "storage_key", "original_name", "mime_type", "size_bytes", "width", "height", "alt_text", "sha256", "created_by", "created_at", "updated_at",
		},
	},
	{
		Name:    "traffic_content_daily_stats",
		Columns: []string{"stat_date", "route_type", "article_id", "path", "pv", "uv"},
	},
	{
		Name:    "traffic_content_daily_visitors",
		Columns: []string{"stat_date", "route_type", "article_id", "path", "visitor_hash"},
	},
}

const runtimeSchemaColumnsQueryPrefix = `SELECT table_name, column_name FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name IN (`

// SchemaReady reports whether the current database includes every table and
// field required by the running application. DATABASE() keeps the check bound
// to the DSN-selected schema instead of a hard-coded database name.
func (s *Store) SchemaReady(ctx context.Context) (bool, error) {
	if s == nil || s.db == nil {
		return false, errors.New("database is not configured")
	}

	rows, err := s.db.QueryContext(ctx, runtimeSchemaColumnsQuery(), runtimeSchemaTableNames()...)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	available := make(map[string]map[string]struct{}, len(runtimeSchemaManifest))
	for rows.Next() {
		var tableName, columnName string
		if err := rows.Scan(&tableName, &columnName); err != nil {
			return false, err
		}
		columns := available[tableName]
		if columns == nil {
			columns = make(map[string]struct{})
			available[tableName] = columns
		}
		columns[columnName] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}

	for _, requirement := range runtimeSchemaManifest {
		columns := available[requirement.Name]
		for _, column := range requirement.Columns {
			if _, ok := columns[column]; !ok {
				return false, nil
			}
		}
	}
	return true, nil
}

func runtimeSchemaColumnsQuery() string {
	placeholders := make([]string, len(runtimeSchemaManifest))
	for index := range placeholders {
		placeholders[index] = "?"
	}
	return runtimeSchemaColumnsQueryPrefix + strings.Join(placeholders, ", ") + ")"
}

func runtimeSchemaTableNames() []any {
	names := make([]any, 0, len(runtimeSchemaManifest))
	for _, requirement := range runtimeSchemaManifest {
		names = append(names, requirement.Name)
	}
	return names
}

// BackupSchemaReady 保留备份调用方的兼容入口，并与应用 readiness 复用同一份
// 运行时结构清单，避免备份与公开/管理员健康检查的迁移判断发生漂移。
func (s *Store) BackupSchemaReady(ctx context.Context) (bool, error) {
	return s.SchemaReady(ctx)
}
