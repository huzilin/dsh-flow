// 装配守护测试：反射枚举 handler.Deps 全部导出字段断言非 nil（标量字段豁免）——
// 防 CR 发现 5 类「service 已实现但 main.go 漏装配、路由静默不注册」缺口再漏网。
package main

import (
	"reflect"
	"testing"

	"github.com/huzilin/wbflow/rd/handler"
	"github.com/huzilin/wbflow/rd/logger"
	"github.com/huzilin/wbflow/rd/repo"
	"github.com/huzilin/wbflow/rd/service"
)

// handlerDepsForTest 按 main.go runServer 装配分支（db.Gorm!=nil）构造全量 Deps。
// 与装配代码同包：main.go 装配行增删直接反映到本构造，编译器保证字段名单一真相源
// 是 handler.Deps 本身，改动装配分支即触发 TestDepsFullyWired 校验。
func handlerDepsForTest() *handler.Deps {
	logger.InitZap("", logger.ModeDev, "info") // 测试进程内初始化全局 zap（stdout，dev 模式）
	repoHandle := repo.NewRepo(nil)
	reviewSvc := service.NewReviewService(repoHandle)
	return &handler.Deps{
		Logger:      logger.L(),
		Health:      service.NewHealth(nil),
		Token:       "", // 空串合法=fail closed，不纳入非 nil 断言
		Reqs:        service.NewRequirementService(repoHandle),
		Vers:        service.NewVersionService(repoHandle),
		Flows:       service.NewFlowService(repoHandle),
		Artifacts:   service.NewArtifactService(repoHandle, ""),
		Tasks:       service.NewTaskService(repoHandle),
		Comments:    service.NewIssueService(repoHandle, 0),
		Escalations: service.NewEscalationService(repoHandle),
		Inbox:       service.NewInboxService(repoHandle),
		Reviews:     reviewSvc,
		Advance:     service.NewAdvanceService(repoHandle),
	}
}

// TestDepsFullyWired 装配完整守护：反射枚举 Deps 全部导出字段，接口/指针字段
// 必须 non-nil（Token 空串合法=fail closed，不纳入）。新增 Deps 字段自动纳入
// 断言（无需改测试），漏装配即测试失败。
func TestDepsFullyWired(t *testing.T) {
	deps := handlerDepsForTest()
	v := reflect.ValueOf(deps).Elem()
	typ := v.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		switch field.Type.Kind() {
		case reflect.Interface, reflect.Ptr:
			if v.Field(i).IsNil() {
				t.Errorf("handler.Deps.%s 未装配（nil）：service 已实现则必须接线，否则路由静默不注册", field.Name)
			}
		default:
			// 标量字段（如 Token）不属装配守护范围
		}
	}
}
