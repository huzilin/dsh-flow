// repo 包本文件：artifacts 表业务查询与 art-<ts> 业务 id 事务内取号（T6）。
package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/huzilin/wbflow/rd/pkg/errs"
)

// artifactIDPrefix 产物业务 id 前缀（spec §0：art-<ts>；ts 取 unix 毫秒，
// 取号单调递增——见 AllocArtifactIDTx）。
const artifactIDPrefix = "art-"

// GetArtifactByID 按业务 id 读产物行；未命中返回 (nil, nil)
// （由 service 层转 ErrNotFound）。
func (r *Repo) GetArtifactByID(ctx context.Context, artifactID string) (*Artifact, error) {
	row := &Artifact{}
	err := r.g.WithContext(ctx).
		Where("artifact_id = ?", artifactID).
		First(row).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// GetArtifactByKeyTx 事务内按幂等键（requirement×node×product_type×href）读产物行
// （加锁读：命中行锁至事务提交，并发重复登记序列化后走 no-op 分支）；
// 未命中返回 (nil, nil)。
func (r *Repo) GetArtifactByKeyTx(ctx context.Context, tx *gorm.DB, requirementID string, nodeID string, productType string, href string) (*Artifact, error) {
	row := &Artifact{}
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("requirement_id = ? AND node_id = ? AND product_type = ? AND href = ?",
			requirementID, nodeID, productType, href).
		First(row).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// ListArtifactsByRequirementAndNode 读需求下指定节点已登记产物（登记序）；
// my_task 产物侧上下文注入用。无登记返回空集。
func (r *Repo) ListArtifactsByRequirementAndNode(ctx context.Context, requirementID string, nodeID string) ([]*Artifact, error) {
	items := make([]*Artifact, 0)
	err := r.g.WithContext(ctx).
		Where("requirement_id = ? AND node_id = ?", requirementID, nodeID).
		Order("id").
		Find(&items).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	return items, nil
}

// AllocArtifactIDTx 事务内取下一个产物业务 id（art-<ts>，spec §0 示例 art-<ts>）。
//
// 取号 = MAX(数字后缀) FOR UPDATE 与当前 unix 毫秒取大者再 +1：
// 毫秒时间戳量级恒大于计数场景的 MAX 后缀，常态即 art-<now+1>（时间戳语义）；
// 同毫秒并发由 FOR UPDATE 范围锁串行化（后者读到前者已提交的更大后缀，取号仍单调）。
// 并发穿透（幻影）由 uk_artifacts_aid 兜底（1062），调用方整体重试事务。
func (r *Repo) AllocArtifactIDTx(ctx context.Context, tx *gorm.DB) (string, error) {
	maxSeq, err := maxIDNumberSuffixTx(ctx, tx, "artifacts", "artifact_id", artifactIDPrefix)
	if err != nil {
		return "", err
	}
	base := maxSeq
	if NowMillis() > base {
		base = NowMillis()
	}
	return fmt.Sprintf("%s%d", artifactIDPrefix, base+1), nil
}
