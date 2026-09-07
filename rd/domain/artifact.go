// Package domain 本文件：产物受控词表（product_type / kind）与 MIME 推断——
// 纯逻辑，无 IO。
//
// 定稿出处：
//   - docs-tmp/data-tools-final.md §1.2 artifacts.product_type 受控枚举 13 值
//     （case_design 移除——定稿明令，登记侧越集一律拒绝）；
//   - .plan/wbflow-impl/spec.md §0 artifacts DDL 列注释同源；
//   - kind 取值 document | link（DDL 默认 document）。
package domain

import (
	"path/filepath"
	"strings"
)

// 产物 kind 词表（artifacts.kind）。
const (
	// ArtifactKindDocument 相对项目路径文档（docs root 下可解析文件）
	ArtifactKindDocument = "document"
	// ArtifactKindLink 外链（http/https URL）
	ArtifactKindLink = "link"
)

// ArtifactProductTypes 产物受控类型清单（词条顺序即定稿顺序；只增不改语义；
// 无 case_design）。
var ArtifactProductTypes = []string{
	"brd",
	"product_doc",
	"rd_design",
	"fe_design",
	"architecture",
	"research",
	"plan",
	"alignment",
	"dev_notes",
	"code_mr",
	"test_result",
	"defect",
	"discussion",
}

// artifactProductTypeSet 受控类型查询集（包初始化一次构建）。
var artifactProductTypeSet = func() map[string]bool {
	set := make(map[string]bool, len(ArtifactProductTypes))
	for _, productType := range ArtifactProductTypes {
		set[productType] = true
	}
	return set
}()

// ValidProductType 判定 product_type 是否在受控清单内。
// 越集值一律拒绝（wb_add_artifact 拦截口径：400 + blocked:true）。
func ValidProductType(productType string) bool {
	return artifactProductTypeSet[productType]
}

// NormalizeArtifactKind kind 归一：空=按 document 处理（与 DDL 默认值一致），
// 其余原样返回（合法性由 ValidArtifactKind 判定）。
func NormalizeArtifactKind(kind string) string {
	if kind == "" {
		return ArtifactKindDocument
	}
	return kind
}

// ValidArtifactKind 判定 kind 是否在词表内（document | link）。
func ValidArtifactKind(kind string) bool {
	return kind == ArtifactKindDocument || kind == ArtifactKindLink
}

// artifactMimeByExt 常见扩展名 → MIME（阅读器渲染模式选择；未命中空串=未知）。
var artifactMimeByExt = map[string]string{
	".md":       "text/markdown",
	".markdown": "text/markdown",
	".txt":      "text/plain",
	".html":     "text/html",
	".htm":      "text/html",
	".json":     "application/json",
	".xml":      "application/xml",
	".yaml":     "application/yaml",
	".yml":      "application/yaml",
	".csv":      "text/csv",
	".pdf":      "application/pdf",
	".png":      "image/png",
	".jpg":      "image/jpeg",
	".jpeg":     "image/jpeg",
	".gif":      "image/gif",
	".svg":      "image/svg+xml",
	".webp":     "image/webp",
}

// ArtifactMimeByExt 按文件扩展名推断 MIME；推断不出返回空串（出参 mime_type 空=
// 未知，阅读器按纯文本兜底）。
func ArtifactMimeByExt(path string) string {
	return artifactMimeByExt[strings.ToLower(filepath.Ext(path))]
}
