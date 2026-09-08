import { useQueries } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import type { DependencyNodeItem, RequirementItem } from "../api/proto/api";
import { DependencyTreeResp } from "../api/proto/api";
import { api } from "../api/client";

/**
 * 依赖树拉取（GET /api/requirements/:reqId/dependencies/tree，rd/handler/flow.go 路由同形）。
 * 注：api 层（flows.ts）由接口封装 owner 维护、本任务禁改，故此封装暂驻组件域，
 * 形状与 api/*.ts 既有 wrapper 一致（api.request + proto fromJSON），待收编时平移。
 */
function fetchDependencyTree(requirementId: string): Promise<DependencyTreeResp> {
  return api.request({
    method: "GET",
    path: `/api/requirements/${encodeURIComponent(requirementId)}/dependencies/tree`,
    decode: (raw) => DependencyTreeResp.fromJSON(raw),
  });
}

/** 森林节点：展示方向以「前置」为父（原型 P9 deproot：根=无前置，子行标「前置 父id」）。 */
export interface DepForestNode {
  requirementId: string;
  title: string;
  status: string;
  /** 本节点的前置需求 id 集（根节点为空集=无前置） */
  prereqIds: string[];
  /** 依赖本节点的下游（每条前置边一个出现位，菱形依赖会在多个父下重复出现） */
  children: DepForestNode[];
}

export interface DepForest {
  roots: DepForestNode[];
  /** 是否存在依赖边（false=整版本无任何前置边，呈现空态） */
  hasEdges: boolean;
}

/**
 * 由各需求的依赖树响应组装版本内森林缩进树（纯函数，单测直测）。
 * 口径（原型 P9 deproot + spec §1.2 注记-D4）：DependencyTreeResp.root=当前需求，
 * children=直接前置递归展开；把边反向即得「前置为父」的缩进树，根=无前置的需求。
 * found=false 的悬挂边（无对应需求行）不渲染不展开（proto 注记同口径）。
 */
export function buildDepForest(treeRoots: DependencyNodeItem[]): DepForest {
  const info = new Map<string, { title: string; status: string }>();
  const prereqIdsOf = new Map<string, string[]>();
  const dependentsOf = new Map<string, string[]>();

  const walk = (node: DependencyNodeItem) => {
    if (node.found && !info.has(node.requirementId)) {
      info.set(node.requirementId, { title: node.title, status: node.status });
    }
    const direct = node.children.filter((c) => c.found);
    prereqIdsOf.set(
      node.requirementId,
      direct.map((c) => c.requirementId),
    );
    for (const c of direct) {
      const list = dependentsOf.get(c.requirementId) ?? [];
      if (!list.includes(node.requirementId)) list.push(node.requirementId);
      dependentsOf.set(c.requirementId, list);
    }
    for (const c of node.children) walk(c);
  };
  for (const root of treeRoots) walk(root);

  const hasEdges = [...prereqIdsOf.values()].some((ids) => ids.length > 0);

  const makeNode = (id: string, prereqIds: string[], path: Set<string>): DepForestNode => {
    const meta = info.get(id);
    const children: DepForestNode[] = [];
    for (const depId of dependentsOf.get(id) ?? []) {
      if (path.has(depId)) continue; // 防环兜底（服务端已拒环 DEPENDENCY_CYCLE，防御性不展开）
      children.push(makeNode(depId, [id], new Set(path).add(id)));
    }
    return {
      requirementId: id,
      title: meta?.title ?? "",
      status: meta?.status ?? "",
      prereqIds,
      children,
    };
  };

  const roots: DepForestNode[] = [];
  for (const [id, prereqIds] of prereqIdsOf) {
    if (prereqIds.length === 0) roots.push(makeNode(id, [], new Set([id])));
  }
  return { roots, hasEdges };
}

/**
 * 版本依赖树（原型 P9：按需拉取——展开行内区域时对该版本全部需求并发调树接口，
 * 组装「前置为父」森林缩进树；↗ 流程树链接跳需求详情（流程树在详情页模块 II））。
 * 仅在展开态挂载（收起即卸载、缓存由 React Query 按 ["dependencyTree", reqId] 保留）。
 */
export function VersionDependencyForest({ reqs }: { reqs: RequirementItem[] }) {
  const navigate = useNavigate();
  const queries = useQueries({
    queries: reqs.map((r) => ({
      queryKey: ["dependencyTree", r.requirementId],
      queryFn: () => fetchDependencyTree(r.requirementId),
    })),
  });

  if (queries.some((q) => q.isPending)) {
    return <div className="dep-forest muted">依赖树加载中…</div>;
  }
  if (queries.some((q) => q.isError)) {
    return <div className="dep-forest muted">依赖树加载失败（可收起后重试）</div>;
  }
  const treeRoots = queries
    .map((q) => q.data?.root)
    .filter((r): r is DependencyNodeItem => r !== undefined);
  const forest = buildDepForest(treeRoots);
  if (!forest.hasEdges) {
    return <div className="dep-forest muted">该版本无依赖边</div>;
  }

  const renderNode = (node: DepForestNode, depth: number) => (
    <div key={`${depth}-${node.requirementId}`} className="dep-node" style={{ paddingLeft: depth * 22 }}>
      <span className="dep-node-mark muted">{depth === 0 ? "▸" : "└"}</span>
      <b>{node.requirementId}</b>
      <span>{node.title}</span>
      <span className="muted">
        {node.prereqIds.length === 0 ? "无前置" : `前置 ${node.prereqIds.join("、")}`}
      </span>
      <button
        type="button"
        className="dep-node-link"
        onClick={() => navigate(`/requirements/${node.requirementId}`)}
      >
        ↗ 流程树
      </button>
      {node.children.map((c) => renderNode(c, depth + 1))}
    </div>
  );

  return <div className="dep-forest">{forest.roots.map((r) => renderNode(r, 0))}</div>;
}
