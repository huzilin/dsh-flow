// 能力装配共享 helper：capability 行集 → capability_id 投影信息映射。
//
// 原 4 处同形循环（flow.go loadCapabilityMaps / task.go loadCapabilityInfos /
// comment.go nodeExecRole 侧 / notifier.go 通知侧）收拢于此——单一真相源，
// 新增装配点一律复用本函数。
package service

import (
	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// buildCapabilityInfoMap 能力装配：capability_id → domain.CapabilityInfo
// （归属角色 + 动作词表 JSON 解析）。任一行动作 JSON 非法即整体报错
// （装配期失败优于运行期半装配）。
func buildCapabilityInfoMap(caps []*repo.Capability) (map[string]domain.CapabilityInfo, error) {
	infos := make(map[string]domain.CapabilityInfo, len(caps))
	for _, cap := range caps {
		var actions []repo.ActionDef
		if err := utils.UnmarshalJSONText(cap.Actions, &actions); err != nil {
			return nil, errs.Trace(err)
		}
		infos[cap.CapabilityID] = domain.CapabilityInfo{RoleID: cap.RoleID, Actions: actions}
	}
	return infos, nil
}
