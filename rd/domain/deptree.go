// Package domain 本文件：依赖树纯逻辑——环检测与树投影（flows.dependencies 消费口径）。
//
// 定稿出处（docs-tmp/data-tools-final.md「依赖树（plugin 层内置，2026-09-05）」+
// docs-tmp/ui-interaction-final.md §2「依赖树视图」）：依据 flows.dependencies 构建需求
// 依赖树，分析师按需拉取（动态加载，非常驻），内部分层不对用户显示；调度器就绪池规则
// （依赖全部 approved）为长期 effort，本文件只交付树只读投影。
package domain

// DependencyEdges 需求依赖图：requirement_id → 直接前置依赖 id 列表
// （flows.dependencies JSON 解析结果；悬挂 id 保留原样，由树投影标 found=false）。
type DependencyEdges map[string][]string

// FindDependencyCycle 从 root 出发做路径 DFS，返回首个环的路径
// （自 root 到重复节点；无环返回 nil）。edges 中 root 不可达的子图不参与判定。
// 同一节点的多条依赖按声明序遍历（确定性输出）。
func FindDependencyCycle(root string, edges DependencyEdges) []string {
	path := []string{root}
	onPath := map[string]bool{root: true}
	visited := map[string]bool{}
	var cycle []string
	var dfs func(string)
	dfs = func(current string) {
		if cycle != nil {
			return
		}
		for _, next := range edges[current] {
			if onPath[next] {
				cycle = append(append([]string{}, path...), next)
				return
			}
			if visited[next] {
				continue
			}
			visited[next] = true
			path = append(path, next)
			onPath[next] = true
			dfs(next)
			path = path[:len(path)-1]
			onPath[next] = false
			if cycle != nil {
				return
			}
		}
	}
	dfs(root)
	return cycle
}
