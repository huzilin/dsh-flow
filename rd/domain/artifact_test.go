// domain 产物受控词表单测（T6）：product_type 13 值定稿清单、case_design 移除、
// kind 词表与归一、MIME 推断。
package domain

import (
	"testing"

	"gotest.tools/v3/assert"
)

// TestUnit_ArtifactProductTypes_ControlledList 受控类型清单 = data-tools-final
// §1.2 定稿 13 值（无 case_design），越集一律拒绝。
func TestUnit_ArtifactProductTypes_ControlledList(t *testing.T) {
	want := []string{
		"brd", "product_doc", "rd_design", "fe_design", "architecture",
		"research", "plan", "alignment", "dev_notes", "code_mr",
		"test_result", "defect", "discussion",
	}
	assert.DeepEqual(t, ArtifactProductTypes, want)
	for _, productType := range want {
		assert.Assert(t, ValidProductType(productType), "受控值应放行: %s", productType)
	}
	assert.Assert(t, !ValidProductType("case_design"), "case_design 已移除（定稿明令）")
	assert.Assert(t, !ValidProductType(""), "空值非法")
	assert.Assert(t, !ValidProductType("BRD"), "大小写敏感（受控词表精确匹配）")
	assert.Assert(t, !ValidProductType("brd "), "尾随空格非法")
}

// TestUnit_ArtifactKind_NormalizeAndValid kind 词表：document | link，空=document。
func TestUnit_ArtifactKind_NormalizeAndValid(t *testing.T) {
	assert.Equal(t, NormalizeArtifactKind(""), ArtifactKindDocument, "空=按 DDL 默认 document")
	assert.Equal(t, NormalizeArtifactKind("link"), "link")
	assert.Assert(t, ValidArtifactKind(ArtifactKindDocument))
	assert.Assert(t, ValidArtifactKind(ArtifactKindLink))
	assert.Assert(t, !ValidArtifactKind(""))
	assert.Assert(t, !ValidArtifactKind("url"), "越出词表拒绝")
	assert.Assert(t, !ValidArtifactKind("Document"), "大小写敏感")
}

// TestUnit_ArtifactMimeByExt 扩展名 MIME 推断（大小写归一；未知空串）。
func TestUnit_ArtifactMimeByExt(t *testing.T) {
	assert.Equal(t, ArtifactMimeByExt("a.md"), "text/markdown")
	assert.Equal(t, ArtifactMimeByExt("docs/README.MD"), "text/markdown", "扩展名大小写归一")
	assert.Equal(t, ArtifactMimeByExt("x/y/z.json"), "application/json")
	assert.Equal(t, ArtifactMimeByExt("a.txt"), "text/plain")
	assert.Equal(t, ArtifactMimeByExt("a.unknownext"), "", "未知扩展名空串")
	assert.Equal(t, ArtifactMimeByExt("noext"), "", "无扩展名空串")
}
