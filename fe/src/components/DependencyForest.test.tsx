import { describe, expect, it } from "vitest";
import type { DependencyNodeItem } from "../api/proto/api";
import { buildDepForest } from "./DependencyForest";

/** buildDepForest 纯函数直测（原型 P9 deproot：前置为父、根=无前置、悬挂边不渲染、防环）。 */
function makeNode(
  requirementId: string,
  children: DependencyNodeItem[] = [],
  over: Partial<DependencyNodeItem> = {},
): DependencyNodeItem {
  return {
    requirementId,
    title: `标题-${requirementId}`,
    status: "进行中",
    versionId: "",
    found: true,
    children,
    ...over,
  };
}

describe("buildDepForest（原型 P9 依赖树森林装配）", () => {
  it("链式依赖：前置为父，根=无前置，子行标前置 id", () => {
    // req-a 前置 req-b；req-b 无前置 → 森林根=req-b，其子=req-a
    const roots = [
      makeNode("req-a", [makeNode("req-b")]),
      makeNode("req-b"),
    ];
    const forest = buildDepForest(roots);
    expect(forest.hasEdges).toBe(true);
    expect(forest.roots).toHaveLength(1);
    expect(forest.roots[0].requirementId).toBe("req-b");
    expect(forest.roots[0].prereqIds).toEqual([]);
    expect(forest.roots[0].children).toHaveLength(1);
    expect(forest.roots[0].children[0].requirementId).toBe("req-a");
    expect(forest.roots[0].children[0].prereqIds).toEqual(["req-b"]);
  });

  it("无依赖边：全部需求为根且 hasEdges=false（「该版本无依赖边」空态判据）", () => {
    const forest = buildDepForest([makeNode("req-a"), makeNode("req-b")]);
    expect(forest.hasEdges).toBe(false);
    expect(forest.roots).toHaveLength(2);
  });

  it("悬挂边（found=false 前置）不渲染不占根", () => {
    const forest = buildDepForest([makeNode("req-a", [makeNode("req-x", [], { found: false })])]);
    expect(forest.hasEdges).toBe(false);
    expect(forest.roots.map((r) => r.requirementId)).toEqual(["req-a"]);
    expect(forest.roots[0].children).toHaveLength(0);
  });

  it("菱形依赖在多个父下重复出现（每条前置边一个出现位）", () => {
    // top 的前置=l/r，l/r 共同前置 base——单棵树（children=前置递归展开）即含全部边
    const forest = buildDepForest([
      makeNode("req-top", [
        makeNode("req-l", [makeNode("req-base")]),
        makeNode("req-r", [makeNode("req-base")]),
      ]),
    ]);
    expect(forest.hasEdges).toBe(true);
    // 根=无前置者（req-base）；「前置为父」反向缩进：base → l/r → top
    const base = forest.roots.find((r) => r.requirementId === "req-base");
    expect(base).toBeDefined();
    expect(base!.children.map((c) => c.requirementId)).toEqual(["req-l", "req-r"]);
    // 菱形：req-base 在 req-l 与 req-r 两个父下各出现一次，各自下游为 req-top
    for (const child of base!.children) {
      expect(child.prereqIds).toEqual(["req-base"]);
      expect(child.children.map((c) => c.requirementId)).toEqual(["req-top"]);
    }
  });
});
