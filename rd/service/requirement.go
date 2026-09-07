// service 包本文件：wb_list / wb_create / wb_get / wb_update / wb_delete_requirement
// 五端点业务编排（spec §1.2 路由表）。DTO 直接以 protoc 生成的 api 包类型承载
// （proto 为契约真相源，spec §1.3 禁手写漂移）。
package service

import (
	"context"
	"fmt"

	gormio "gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/huzilin/wbflow/rd/api"
	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/repo"
)

// 分页默认值与上限（强制分页口径，spec §3：大需求池列表强制分页）。
const (
	defaultPageSize int64 = 20
	maxPageSize     int64 = 100
)

// auditByWhoManagement 管理面写路径审计操作者口径：本地部署为 User 直操作
// （audit_log.by_who 取值：role_id/user/system/plugin）。
const auditByWhoManagement = "user"

// 取号/插入竞态整体重试上限（唯一键 1062 与锁竞争 1213/1205 同等重试）。
const idAllocRetryLimit = 3

// RequirementService 需求管理面 service（requirements 五端点）。
type RequirementService struct {
	repo *repo.Repo
}

// NewRequirementService 构造需求 service。
func NewRequirementService(r *repo.Repo) *RequirementService {
	return &RequirementService{repo: r}
}

// List wb_list_requirements：version/priority/status 筛选 + 强制分页的
// 需求列表（id DESC 排序，新需求在前）。
//
// 入参：req.VersionId 精确版本筛选（空=不过滤）；req.Backlog=true 仅待分配池
// （version_id IS NULL，与 VersionId 同给时 Backlog 优先）；page/page_size
// 归一化（page<1 视为 1；page_size<1 视为 20，>100 截断 100）。
// 出参：筛选后 total + 当前页 items（分页基数=筛选后总数）。
//
// 错误集：无业务错误；底层 IO 错误经 Trace 上抛（HTTP 侧 INTERNAL）。
func (s *RequirementService) List(ctx context.Context, req *api.ListRequirementsReq) (*api.ListRequirementsResp, error) {
	filter := repo.ListRequirementFilter{
		Priority: req.Priority,
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	if req.Backlog {
		filter.Backlog = true
	} else if req.VersionId != "" {
		filter.VersionID = &req.VersionId
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = defaultPageSize
	}
	if filter.PageSize > maxPageSize {
		filter.PageSize = maxPageSize
	}

	rows, total, err := s.repo.ListRequirements(ctx, filter)
	if err != nil {
		return nil, errs.Trace(err)
	}
	items := make([]*api.RequirementItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, requirementItem(row))
	}
	return &api.ListRequirementsResp{
		Ok:       true,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Items:    items,
	}, nil
}

// Get wb_get_requirement：聚合快照基础版——需求行 + 归属版本信息 +
// flows/flow_nodes/artifacts/issues 集合字段（T4 实例化前为空集，结构先立住）
// + review_results 汇总 JSON 文本（TEXT 列同构，T4 前恒 "[]"）。
//
// 入参：req.RequirementId 需求业务 id。
// 出参：requirement 恒填充；version 仅归属版本存在时填充（待分配池为 nil，
// protojson 出 null）。
//
// 错误集：ErrNotFound（需求不存在）。
func (s *RequirementService) Get(ctx context.Context, req *api.GetRequirementReq) (*api.GetRequirementResp, error) {
	if req.RequirementId == "" {
		return nil, invalidArg("requirement_id 必填")
	}
	row, err := s.repo.GetRequirementByID(ctx, req.RequirementId)
	if err != nil {
		return nil, errs.Trace(err)
	}
	if row == nil {
		return nil, notFound("需求 %q 不存在", req.RequirementId)
	}

	var ver *api.VersionItem
	if row.VersionID != nil {
		verRow, err := s.repo.GetVersionByID(ctx, *row.VersionID)
		if err != nil {
			return nil, errs.Trace(err)
		}
		if verRow != nil {
			ver = versionItem(verRow)
		}
	}

	flowRows, err := s.repo.ListFlowsByRequirement(ctx, req.RequirementId)
	if err != nil {
		return nil, errs.Trace(err)
	}
	flows := make([]*api.FlowItem, 0, len(flowRows))
	for _, f := range flowRows {
		currentNode := ""
		if f.CurrentNodeID != nil {
			currentNode = *f.CurrentNodeID
		}
		flows = append(flows, &api.FlowItem{
			FlowDefsId:    f.FlowDefsID,
			Dependencies:  f.Dependencies,
			CurrentNodeId: currentNode,
			CreatedAt:     f.CreatedAt,
			UpdatedAt:     f.UpdatedAt,
		})
	}

	nodeRows, err := s.repo.ListFlowNodesByRequirement(ctx, req.RequirementId)
	if err != nil {
		return nil, errs.Trace(err)
	}
	nodes := make([]*api.FlowNodeItem, 0, len(nodeRows))
	for _, n := range nodeRows {
		featureBranch := ""
		if n.FeatureBranch != nil {
			featureBranch = *n.FeatureBranch
		}
		enteredAt := int64(0)
		if n.EnteredAt != nil {
			enteredAt = *n.EnteredAt
		}
		nodes = append(nodes, &api.FlowNodeItem{
			NodeId:          n.NodeID,
			DefNodeId:       n.DefNodeID,
			NodeOrder:       n.NodeOrder,
			NodeKey:         n.NodeKey,
			State:           n.State,
			ReviewResults:   n.ReviewResults,
			ReopenSource:    n.ReopenSource,
			RolledBackCount: n.RolledBackCount,
			FeatureBranch:   featureBranch,
			EnteredAt:       enteredAt,
		})
	}

	artifactRows, err := s.repo.ListArtifactsByRequirement(ctx, req.RequirementId)
	if err != nil {
		return nil, errs.Trace(err)
	}
	artifacts := make([]*api.ArtifactItem, 0, len(artifactRows))
	for _, a := range artifactRows {
		artifacts = append(artifacts, &api.ArtifactItem{
			ArtifactId:  a.ArtifactID,
			NodeId:      a.NodeID,
			ProductType: a.ProductType,
			Label:       a.Label,
			Href:        a.Href,
			Kind:        a.Kind,
			Note:        a.Note,
			AddedBy:     a.AddedBy,
			CreatedAt:   a.CreatedAt,
		})
	}

	issueRows, err := s.repo.ListIssuesByRequirement(ctx, req.RequirementId)
	if err != nil {
		return nil, errs.Trace(err)
	}
	issues := make([]*api.IssueItem, 0, len(issueRows))
	for _, i := range issueRows {
		issues = append(issues, &api.IssueItem{
			IssueId:    i.IssueID,
			NodeId:     i.NodeID,
			AnchorKind: i.AnchorKind,
			Author:     i.Author,
			RoleId:     i.RoleID,
			Text:       i.Text,
			State:      i.State,
			Escalate:   i.Escalate,
			LoopCount:  i.LoopCount,
			RoundCount: i.RoundCount,
			CreatedAt:  i.CreatedAt,
		})
	}

	return &api.GetRequirementResp{
		Ok:            true,
		Requirement:   requirementItem(row),
		Version:       ver,
		Flows:         flows,
		FlowNodes:     nodes,
		Artifacts:     artifacts,
		Issues:        issues,
		ReviewResults: "[]",
	}, nil
}

// Create wb_create_requirement：需求创建——category 派生流程类型绑定
// （domain.DeriveFlowDefsID；explicit flow_defs_id 须在类别允许集内）+
// requirement_id 取号（req-NNN 递增，事务内取号防重，竞态整体重试）+
// 默认 version_id NULL=待分配池、status=待启动、priority 空值默认 P2。
// 创建动作同事务写 audit_log（INSERT，by_who=user）。
//
// 入参：title/created_by 必填；category 必填且须为已知类别；
// version_id 提供时须为已存在版本。
// 出参：创建后的需求行（含生成的 requirement_id 与新基线 updated_at）。
//
// 错误集：ErrInvalidArgument（必填缺失/未知类别/流程越集/优先级枚举外）；
// ErrNotFound（version_id 未命中）；ErrIdempotentConflict（取号重试耗尽）。
func (s *RequirementService) Create(ctx context.Context, req *api.CreateRequirementReq) (*api.CreateRequirementResp, error) {
	if req.Title == "" {
		return nil, invalidArg("title 必填")
	}
	if req.CreatedBy == "" {
		return nil, invalidArg("created_by 必填")
	}
	if req.Category == "" {
		return nil, invalidArg("category 必填")
	}
	flowDefsID, err := domain.DeriveFlowDefsID(req.Category, req.FlowDefsId)
	if err != nil {
		return nil, invalidArg("%v", err)
	}
	priority := req.Priority
	if priority == "" {
		priority = domain.DefaultPriority
	}
	if !domain.ValidPriority(priority) {
		return nil, invalidArg("非法优先级 %q（允许 P0-P3）", priority)
	}
	if req.VersionId != "" {
		exists, err := s.versionExists(ctx, req.VersionId)
		if err != nil {
			return nil, errs.Trace(err)
		}
		if !exists {
			return nil, notFound("版本 %q 不存在", req.VersionId)
		}
	}

	for attempt := 0; attempt < idAllocRetryLimit; attempt++ {
		row, retryable, err := s.createOnce(ctx, req, flowDefsID, priority)
		if err != nil {
			if retryable {
				continue
			}
			return nil, err
		}
		return &api.CreateRequirementResp{Ok: true, Requirement: requirementItem(row)}, nil
	}
	return nil, errs.Trace(errs.ErrIdempotentConflict)
}

// createOnce 单次创建尝试（事务：取号 → 插入 → 审计）；返回 retryable=true
// 表示唯一键冲突/锁竞争可整体重试。
func (s *RequirementService) createOnce(ctx context.Context, req *api.CreateRequirementReq, flowDefsID string, priority string) (*repo.Requirement, bool, error) {
	row := &repo.Requirement{
		Title:       req.Title,
		Description: req.Description,
		Category:    req.Category,
		FlowDefsID:  flowDefsID,
		Priority:    priority,
		Status:      domain.ReqStatusPending,
		Owner:       req.Owner,
		CreatedBy:   req.CreatedBy,
	}
	if req.VersionId != "" {
		row.VersionID = strPtr(req.VersionId)
	}
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		requirementID, err := s.repo.AllocRequirementIDTx(ctx, tx)
		if err != nil {
			return err
		}
		row.RequirementID = requirementID
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		return s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
			Table:  "requirements",
			RowID:  requirementID,
			Action: "INSERT",
			ByWho:  auditByWhoManagement,
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

// Update wb_update_requirement：乐观锁更新（全字段替换语义）+ 自动 diff
// 变更历史——逐变更字段同事务写 audit_log（UPDATE，旧值/新值，by_who=user）。
// category 变更时 flow_defs_id 按绑定规则重派生（未变更则保留原绑定）；
// version_id 空串=移回待分配池（NULL）；无字段变更时 no-op 返回当前行
// （幂等无害，不写审计）。
//
// 入参：requirement_id 必填；updated_at=读取基线（0=以当前库值为基线，宽松模式）；
// title/created_by 类必填校验、priority/status 枚举校验同创建。
// 出参：更新后的需求行（新 updated_at 基线）。
//
// 错误集：ErrInvalidArgument（必填缺失/枚举外/流程派生越集）；
// ErrNotFound（需求或目标版本不存在）；ErrOptimisticLock（基线不匹配，409）。
func (s *RequirementService) Update(ctx context.Context, req *api.UpdateRequirementReq) (*api.UpdateRequirementResp, error) {
	if req.RequirementId == "" {
		return nil, invalidArg("requirement_id 必填")
	}
	if req.Title == "" {
		return nil, invalidArg("title 必填")
	}
	if !domain.ValidPriority(req.Priority) {
		return nil, invalidArg("非法优先级 %q（允许 P0-P3）", req.Priority)
	}
	if !domain.ValidRequirementStatus(req.Status) {
		return nil, invalidArg("非法状态 %q（允许：待启动/进行中/已完成/已取消）", req.Status)
	}
	if req.Category == "" {
		return nil, invalidArg("category 必填")
	}
	if req.VersionId != "" {
		exists, err := s.versionExists(ctx, req.VersionId)
		if err != nil {
			return nil, errs.Trace(err)
		}
		if !exists {
			return nil, notFound("版本 %q 不存在", req.VersionId)
		}
	}

	var updated *repo.Requirement
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		row, err := getRequirementTx(ctx, tx, req.RequirementId)
		if err != nil {
			return err
		}
		if row == nil {
			return notFound("需求 %q 不存在", req.RequirementId)
		}

		flowDefsID := row.FlowDefsID
		if req.Category != row.Category {
			rederived, err := domain.DeriveFlowDefsID(req.Category, "")
			if err != nil {
				return invalidArg("%v", err)
			}
			flowDefsID = rederived
		}

		newVersionID := ""
		if req.VersionId != "" {
			newVersionID = req.VersionId
		}

		changes := diffRequirement(row, &updatePatch{
			Title:       req.Title,
			Description: req.Description,
			Category:    req.Category,
			FlowDefsID:  flowDefsID,
			Priority:    req.Priority,
			Status:      req.Status,
			Owner:       req.Owner,
			VersionID:   newVersionID,
		})

		if len(changes) == 0 {
			updated = row
			return nil
		}

		updates := make(map[string]any, len(changes))
		for _, change := range changes {
			updates[change.column] = change.newVal
		}
		baseline := row.UpdatedAt
		if req.UpdatedAt != 0 {
			baseline = req.UpdatedAt
		}
		if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.Requirement{}, row.ID, baseline, updates); err != nil {
			return err
		}
		for field, change := range changes {
			if err := s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
				Table:    "requirements",
				RowID:    row.RequirementID,
				Action:   "UPDATE",
				Field:    strPtr(field),
				OldValue: change.oldText(),
				NewValue: change.newText(),
				ByWho:    auditByWhoManagement,
			}); err != nil {
				return err
			}
		}

		fresh, err := getRequirementTx(ctx, tx, req.RequirementId)
		if err != nil {
			return err
		}
		updated = fresh
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return &api.UpdateRequirementResp{Ok: true, Requirement: requirementItem(updated)}, nil
}

// Delete wb_delete_requirement ⚠️：删除类端点保持「需 User 明确指令」口径——
// confirm 未显式置 true 即 blocked（不执行）；status=进行中 拒绝删除（blocked）。
// 删除动作同事务写 audit_log（DELETE，by_who=user）；重复删除命中 ErrNotFound
// （删除幂等=资源已不存在）。
//
// 入参：requirement_id 必填；confirm 必须显式 true。
// 出参：被删 requirement_id。
//
// 错误集：ErrGateBlocked（未 confirm / 进行中需求）——HTTP 侧 blocked:true；
// ErrNotFound（需求不存在）；ErrInvalidArgument（id 缺失）。
func (s *RequirementService) Delete(ctx context.Context, req *api.DeleteRequirementReq) (*api.DeleteRequirementResp, error) {
	if req.RequirementId == "" {
		return nil, invalidArg("requirement_id 必填")
	}
	if !req.Confirm {
		return nil, gateBlocked("删除需求 %q 需 User 明确指令（confirm=true）", req.RequirementId)
	}
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		row, err := getRequirementTx(ctx, tx, req.RequirementId)
		if err != nil {
			return err
		}
		if row == nil {
			return notFound("需求 %q 不存在", req.RequirementId)
		}
		if row.Status == domain.ReqStatusRunning {
			return gateBlocked("需求 %q 进行中，拒绝删除（先取消或完成）", req.RequirementId)
		}
		if err := s.repo.DeleteRequirementByIDTx(tx, row.ID); err != nil {
			return err
		}
		return s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
			Table:  "requirements",
			RowID:  row.RequirementID,
			Action: "DELETE",
			ByWho:  auditByWhoManagement,
		})
	})
	if txErr != nil {
		return nil, txErr
	}
	return &api.DeleteRequirementResp{Ok: true, RequirementId: req.RequirementId}, nil
}

// versionExists 版本存在性（service 层保证逻辑引用完整性——无物理外键口径）。
func (s *RequirementService) versionExists(ctx context.Context, versionID string) (bool, error) {
	row, err := s.repo.GetVersionByID(ctx, versionID)
	if err != nil {
		return false, err
	}
	return row != nil, nil
}

// getRequirementTx 事务内按业务 id 读需求行（加锁读，防删除/更新竞态读脏）。
func getRequirementTx(ctx context.Context, tx *gormio.DB, requirementID string) (*repo.Requirement, error) {
	row := &repo.Requirement{}
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("requirement_id = ?", requirementID).
		First(row).Error
	if err != nil {
		if err == gormio.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// requirementChange 单字段变更（diff 变更历史载体）。
type requirementChange struct {
	// column 数据库列名（snake_case）
	column string
	// oldVal / newVal 旧值/新值（version_id 用 "" 表 NULL）
	oldVal string
	newVal string
}

// oldText / newText 审计出参文本（TEXT 列字符串口径）。
func (c requirementChange) oldText() *string { return strPtr(c.oldVal) }
func (c requirementChange) newText() *string { return strPtr(c.newVal) }

// updatePatch 更新目标值集合（diff 基准）。
type updatePatch struct {
	Title       string
	Description string
	Category    string
	FlowDefsID  string
	Priority    string
	Status      string
	Owner       string
	VersionID   string
}

// diffRequirement 逐字段 diff，返回 变更字段名 → 变更（含列名与新旧值）。
// 字段名与 proto 契约字段一致（camelCase，审计可读口径）。
func diffRequirement(row *repo.Requirement, patch *updatePatch) map[string]requirementChange {
	oldVersionID := ""
	if row.VersionID != nil {
		oldVersionID = *row.VersionID
	}
	candidates := []struct {
		field  string
		column string
		oldVal string
		newVal string
	}{
		{"title", "title", row.Title, patch.Title},
		{"description", "description", row.Description, patch.Description},
		{"category", "category", row.Category, patch.Category},
		{"flowDefsId", "flow_defs_id", row.FlowDefsID, patch.FlowDefsID},
		{"priority", "priority", row.Priority, patch.Priority},
		{"status", "status", row.Status, patch.Status},
		{"owner", "owner", row.Owner, patch.Owner},
		{"versionId", "version_id", oldVersionID, patch.VersionID},
	}
	changes := make(map[string]requirementChange)
	for _, c := range candidates {
		if c.oldVal != c.newVal {
			changes[c.field] = requirementChange{
				column: c.column,
				oldVal: c.oldVal,
				newVal: c.newVal,
			}
		}
	}
	return changes
}

// requirementItem repo 行转 proto 契约出参（version_id NULL → 空串=待分配池）。
func requirementItem(row *repo.Requirement) *api.RequirementItem {
	versionID := ""
	if row.VersionID != nil {
		versionID = *row.VersionID
	}
	return &api.RequirementItem{
		RequirementId: row.RequirementID,
		Title:         row.Title,
		Description:   row.Description,
		Category:      row.Category,
		FlowDefsId:    row.FlowDefsID,
		VersionId:     versionID,
		Priority:      row.Priority,
		Status:        row.Status,
		Owner:         row.Owner,
		CreatedBy:     row.CreatedBy,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

// versionItem repo 行转 proto 契约出参。
func versionItem(row *repo.Version) *api.VersionItem {
	return &api.VersionItem{
		VersionId: row.VersionID,
		Name:      row.Name,
		State:     row.State,
		IsBacklog: row.IsBacklog != 0,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

// invalidArg 入参非法错误（哨兵 errs.ErrInvalidArgument 为 cause）。
func invalidArg(format string, args ...any) error {
	return errs.Trace(fmt.Errorf("%w: %s", errs.ErrInvalidArgument, fmt.Sprintf(format, args...)))
}

// notFound 资源不存在错误（哨兵 errs.ErrNotFound 为 cause）。
func notFound(format string, args ...any) error {
	return errs.Trace(fmt.Errorf("%w: %s", errs.ErrNotFound, fmt.Sprintf(format, args...)))
}

// gateBlocked 门禁拦截错误（哨兵 errs.ErrGateBlocked 为 cause，HTTP 侧 blocked:true）。
func gateBlocked(format string, args ...any) error {
	return errs.Trace(fmt.Errorf("%w: %s", errs.ErrGateBlocked, fmt.Sprintf(format, args...)))
}
