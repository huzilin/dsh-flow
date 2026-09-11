// service 包本文件：wb_add_artifact / wb_get_artifact_content（T6 产物登记与
// 正文阅读，spec §1.2 路由表两行）。产物是评审与评论的锚定对象。
// DTO 以 protoc 生成的 api 包类型承载（proto 为契约真相源）。
package service

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	gormio "gorm.io/gorm"

	"github.com/huzilin/wbflow/rd/api"
	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/repo"
)

// defaultDocsRoot docs root 配置缺省值（artifacts.docs_root 未配置时生效）：
// 仓库内专用目录（相对服务进程工作目录解析）。安全默认最小权限——
// 不取 "/" 或进程根，读取路径 resolve 后越出该目录一律拒绝。
const defaultDocsRoot = "docs"

// ArtifactService 产物登记与正文阅读 service（wb_add_artifact / wb_get_artifact_content）。
type ArtifactService struct {
	repo *repo.Repo
	// docsRoot document 类产物 href 解析根（构造时绝对化；symlink 语义
	// 在每次解析时经 EvalSymlinks 归一）
	docsRoot string
}

// NewArtifactService 构造产物 service。docsRoot 取配置 artifacts.docs_root；
// 空值回落 defaultDocsRoot。相对路径按服务进程工作目录解析。
func NewArtifactService(r *repo.Repo, docsRoot string) *ArtifactService {
	root := docsRoot
	if root == "" {
		root = defaultDocsRoot
	}
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	return &ArtifactService{repo: r, docsRoot: root}
}

// Add wb_add_artifact：产物登记——product_type 受控枚举校验（越集 400 + blocked）+
// 节点归属校验（requirement 下须存在该 node，否则 404）+ href 按 kind 分支校验
// （document=相对 docs root 可解析且文件存在；link=http(s) 外链）+
// artifact_id 取号（art-<ts>，事务内取号防重）+ 写路径单事务 + INSERT 审计
// （by_who=added_by）+ 幂等：同 requirement×node×product_type×href 重复登记
// no-op 返回既有行（不新取号、不重复审计）。
//
// 入参：requirement_id/node_id/product_type/label/href/added_by 必填；
// kind 空=按 document 处理（与 DDL 默认值一致），越出词表拦截。
// 出参：登记后的产物行（含生成的 artifact_id；幂等命中=既有行）。
//
// 错误集：ErrInvalidArgument（必填缺失/href 非法/docs root 不可用）；
// ErrPolicyRejected（product_type/kind 越集、href 路径越界——400 + blocked:true）；
// ErrNotFound（需求或节点不存在）；ErrIdempotentConflict（取号重试耗尽）。
func (s *ArtifactService) Add(ctx context.Context, req *api.AddArtifactReq) (*api.AddArtifactResp, error) {
	if req.RequirementId == "" {
		return nil, invalidArg("requirement_id 必填")
	}
	if req.NodeId == "" {
		return nil, invalidArg("node_id 必填")
	}
	if req.ProductType == "" {
		return nil, invalidArg("product_type 必填")
	}
	if !domain.ValidProductType(req.ProductType) {
		return nil, policyRejected("product_type %q 不在受控清单（允许：%s）",
			req.ProductType, strings.Join(domain.ArtifactProductTypes, "/"))
	}
	if req.Label == "" {
		return nil, invalidArg("label 必填")
	}
	if req.Href == "" {
		return nil, invalidArg("href 必填")
	}
	if req.AddedBy == "" {
		return nil, invalidArg("added_by 必填")
	}
	kind := domain.NormalizeArtifactKind(req.Kind)
	if !domain.ValidArtifactKind(kind) {
		return nil, policyRejected("kind %q 越出词表（允许：%s | %s）",
			kind, domain.ArtifactKindDocument, domain.ArtifactKindLink)
	}
	switch kind {
	case domain.ArtifactKindDocument:
		if _, exists, err := s.resolveDocsPath(req.Href); err != nil {
			return nil, err
		} else if !exists {
			return nil, invalidArg("产物文件在 docs root 下不存在: %q", req.Href)
		}
	case domain.ArtifactKindLink:
		if err := validateExternalHref(req.Href); err != nil {
			return nil, err
		}
	}

	requirement, err := s.repo.GetRequirementByID(ctx, req.RequirementId)
	if err != nil {
		return nil, errs.Trace(err)
	}
	if requirement == nil {
		return nil, notFound("需求 %q 不存在", req.RequirementId)
	}
	node, err := s.repo.GetFlowNodeByRequirementAndNodeID(ctx, req.RequirementId, req.NodeId)
	if err != nil {
		return nil, errs.Trace(err)
	}
	if node == nil {
		return nil, notFound("节点 %q 不存在于需求 %q", req.NodeId, req.RequirementId)
	}
	// 注记-L2（BUG-016）：done=离场终态，禁止挂载产物（防事后补挂绕过 exec 判据）；
	// 返工走 rejected 打回重入。
	if node.State == repo.PhaseDone {
		return nil, policyRejected("节点 %q 已完成离场（done），禁止挂载产物；如需返工请 rejected 打回重入", node.NodeID)
	}

	// BUG-015 守卫（产物入库约束，spec 注记-K）：节点 expected_outputs 对该
	// product_type 有期望且期望为入库文档（kind=document）时，不得以 link 登记——
	// 外链冒充入库文档会在阅读器断链（无正文/无锚定评论）。
	// 期望外 product_type（如 code_mr 补充外链）不受限。
	expected, err := defExpectedOutputs(ctx, s.repo, node.DefNodeID)
	if err != nil {
		return nil, err
	}
	for _, exp := range expected {
		if exp.ProductType == req.ProductType &&
			domain.NormalizeArtifactKind(exp.Kind) == domain.ArtifactKindDocument &&
			kind == domain.ArtifactKindLink {
			return nil, policyRejected("产物 %q 在节点期望中为入库文档（kind=document），不得以 link 登记", req.ProductType)
		}
	}

	row, err := withIDAllocRetry(func() (*repo.Artifact, bool, error) {
		return s.addOnce(ctx, req, kind)
	})
	if err != nil {
		return nil, err
	}
	return &api.AddArtifactResp{Ok: true, Artifact: artifactItem(row)}, nil
}

// addOnce 单次登记尝试（事务：幂等查 → 取号 → 插入 → INSERT 审计）；
// 返回 retryable=true 表示唯一键冲突/锁竞争可整体重试（uk_artifacts_aid 兜底并发取号）。
func (s *ArtifactService) addOnce(ctx context.Context, req *api.AddArtifactReq, kind string) (*repo.Artifact, bool, error) {
	row := &repo.Artifact{
		RequirementID: req.RequirementId,
		NodeID:        req.NodeId,
		ProductType:   req.ProductType,
		Label:         req.Label,
		Href:          req.Href,
		Kind:          kind,
		Note:          req.Note,
		AddedBy:       req.AddedBy,
	}
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		// 幂等：同 requirement×node×product_type×href 已登记 → no-op 返回既有
		// （加锁读，与并发重复登记序列化；不做状态变更、不写审计）
		existing, err := s.repo.GetArtifactByKeyTx(ctx, tx, req.RequirementId, req.NodeId, req.ProductType, req.Href)
		if err != nil {
			return err
		}
		if existing != nil {
			row = existing
			return nil
		}
		artifactID, err := s.repo.AllocArtifactIDTx(ctx, tx)
		if err != nil {
			return err
		}
		row.ArtifactID = artifactID
		if err := tx.Create(row).Error; err != nil {
			return errs.Trace(err)
		}
		return s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
			Table:  "artifacts",
			RowID:  artifactID,
			Action: "INSERT",
			ByWho:  req.AddedBy,
		})
	})
	if txErr != nil {
		if repo.IsDuplicateEntry(txErr) || repo.IsLockRace(txErr) {
			return nil, true, errs.Trace(txErr)
		}
		return nil, false, errs.Trace(txErr)
	}
	return row, false, nil
}

// GetContent wb_get_artifact_content：按 artifact 读正文与锚点定位所需元数据。
// kind=document——读配置 docs root 下相对路径文件内容（路径穿越防护：resolve
// 含 symlink 展开后必须仍在 root 内，越界 400 拦截；文件缺失 404）；
// kind=link——返回 href 本身不读文件（content 恒空）。
//
// 入参：requirement_id/artifact_id 必填；artifact 须归属该 requirement。
// 出参：artifactId/nodeId/productType/label/kind/href 元数据 + content（仅
// document）+ mimeType（document 按扩展名推断，未知为空）。
//
// 错误集：ErrInvalidArgument（必填缺失/docs root 不可用）；
// ErrPolicyRejected（href resolve 后越出 docs root——400 + blocked:true）；
// ErrNotFound（产物不存在 / 不归属该需求 / 文档文件缺失）。
func (s *ArtifactService) GetContent(ctx context.Context, req *api.GetArtifactContentReq) (*api.GetArtifactContentResp, error) {
	if req.RequirementId == "" {
		return nil, invalidArg("requirement_id 必填")
	}
	if req.ArtifactId == "" {
		return nil, invalidArg("artifact_id 必填")
	}
	row, err := s.repo.GetArtifactByID(ctx, req.ArtifactId)
	if err != nil {
		return nil, errs.Trace(err)
	}
	if row == nil {
		return nil, notFound("产物 %q 不存在", req.ArtifactId)
	}
	if row.RequirementID != req.RequirementId {
		return nil, notFound("产物 %q 不属于需求 %q", req.ArtifactId, req.RequirementId)
	}

	resp := &api.GetArtifactContentResp{
		Ok:          true,
		ArtifactId:  row.ArtifactID,
		NodeId:      row.NodeID,
		ProductType: row.ProductType,
		Label:       row.Label,
		Kind:        row.Kind,
		Href:        row.Href,
	}
	if row.Kind == domain.ArtifactKindLink {
		// link：外链直出，不读文件（content 恒空、mime 不适用）
		return resp, nil
	}
	real, exists, err := s.resolveDocsPath(row.Href)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, notFound("产物文件缺失: %q（docs root %q 下不可解析）", row.Href, s.docsRoot)
	}
	data, err := os.ReadFile(real)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, notFound("产物文件缺失: %q", row.Href)
		}
		return nil, errs.Trace(err)
	}
	resp.Content = string(data)
	resp.MimeType = domain.ArtifactMimeByExt(row.Href)
	return resp, nil
}

// resolveDocsPath document href 解析（路径穿越防护核心，登记校验与正文读取
// 共用同一函数——防线不打折）：
//
//  1. href 空 / 绝对路径 → 拦截（ErrPolicyRejected）；
//  2. docs root 不可用（不存在/非目录，构造配置问题）→ ErrInvalidArgument；
//  3. 目标路径 Join 清洗 + EvalSymlinks 全展开（含中间目录与叶子 symlink）；
//     叶子缺失时按「父目录 resolve + 叶名拼接」归一（缺失分支语义保留）；
//  4. 归一结果必须严格位于 root 真实路径内（前缀 + 分隔符），越界拦截。
//
// 返回归一化绝对路径与 exists 标记（调用方自行区分缺失分支）。
//
// 错误集：ErrPolicyRejected（越界）；ErrInvalidArgument（root 不可用/解析异常）。
func (s *ArtifactService) resolveDocsPath(href string) (string, bool, error) {
	if href == "" {
		return "", false, policyRejected("href 为空")
	}
	if filepath.IsAbs(href) {
		return "", false, policyRejected("href 不允许绝对路径: %q", href)
	}
	rootReal, err := filepath.EvalSymlinks(s.docsRoot)
	if err != nil {
		return "", false, invalidArg("docs root %q 不可用: %v", s.docsRoot, err)
	}
	rootInfo, err := os.Stat(rootReal)
	if err != nil || !rootInfo.IsDir() {
		return "", false, invalidArg("docs root %q 不可用（非目录或不可访问）", s.docsRoot)
	}

	target := filepath.Join(s.docsRoot, href)
	// 词法前置校验：Join 清洗后即越出 root（"../" 族）直接拦截——
	// 叶子缺失时 symlink 展开无从谈起，不能降级为 404（越界语义不变）
	if escaped := outsideRoot(target, s.docsRoot); escaped {
		return "", false, policyRejected("href resolve 后越出 docs root: %q", href)
	}
	real, err := filepath.EvalSymlinks(target)
	if err != nil {
		if !os.IsNotExist(err) {
			// 中间路径非目录等异常形态（如 "a.md/b"）——按入参非法拒绝
			return "", false, invalidArg("href 不可解析: %q (%v)", href, err)
		}
		// 叶子缺失：父目录 resolve 后拼接叶名，归一路径仍须过越界校验
		parentReal, parentErr := filepath.EvalSymlinks(filepath.Dir(target))
		if parentErr != nil {
			if os.IsNotExist(parentErr) {
				return "", false, nil
			}
			return "", false, invalidArg("href 不可解析: %q (%v)", href, parentErr)
		}
		real = filepath.Join(parentReal, filepath.Base(target))
		if escaped := outsideRoot(real, rootReal); escaped {
			return "", false, policyRejected("href resolve 后越出 docs root: %q", href)
		}
		return real, false, nil
	}
	if escaped := outsideRoot(real, rootReal); escaped {
		return "", false, policyRejected("href resolve 后越出 docs root: %q", href)
	}
	return real, true, nil
}

// outsideRoot 判定归一化路径是否越出 root 真实路径（root 本身命中不算越界）。
func outsideRoot(real string, rootReal string) bool {
	if real == rootReal {
		return false
	}
	return !strings.HasPrefix(real, rootReal+string(filepath.Separator))
}

// validateExternalHref link href 校验：须为带 host 的 http(s) 绝对 URL。
func validateExternalHref(href string) error {
	parsed, err := url.Parse(href)
	if err != nil {
		return invalidArg("href 非合法 URL: %q (%v)", href, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return invalidArg("link href 须为 http(s) 外链: %q", href)
	}
	if parsed.Host == "" {
		return invalidArg("link href 缺少 host: %q", href)
	}
	return nil
}

// policyRejected 受控词表越集/路径越界拦截错误（哨兵 errs.ErrPolicyRejected 为
// cause，HTTP 侧 400 + blocked:true）。
func policyRejected(format string, args ...any) error {
	return errs.Trace(fmt.Errorf("%w: %s", errs.ErrPolicyRejected, fmt.Sprintf(format, args...)))
}

// artifactItem repo 行转 proto 契约出参。
func artifactItem(row *repo.Artifact) *api.ArtifactItem {
	return &api.ArtifactItem{
		ArtifactId:  row.ArtifactID,
		NodeId:      row.NodeID,
		ProductType: row.ProductType,
		Label:       row.Label,
		Href:        row.Href,
		Kind:        row.Kind,
		Note:        row.Note,
		AddedBy:     row.AddedBy,
		CreatedAt:   row.CreatedAt,
	}
}
