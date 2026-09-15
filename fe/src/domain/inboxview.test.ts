/** C4 归位测试：TODO 五分类投影规则（spec -D5/N3）——分类归属与提醒类排除。 */
import { describe, expect, it } from "vitest";
import type { InboxItem } from "../api/proto/api";
import { projectTodoGroups } from "./inboxview";

function item(kind: string, targetId = ""): InboxItem {
  return {
    issueId: `cm-${kind}-${targetId}`,
    kind,
    targetId,
    state: "open",
  } as InboxItem;
}

describe("projectTodoGroups（TODO 五分类投影）", () => {
  it("五分类归属与子型拆分", () => {
    const groups = projectTodoGroups([
      item("gate_pending"),
      item("clarify_task"),
      item("re_review"), // 澄清子型（target_id 空）
      item("re_review", "cm-1"), // retest 子型（target_id 非空）
      item("escalation"),
    ]);
    expect(groups.map((g) => g.items.length)).toEqual([1, 1, 1, 1, 1]);
    expect(groups.map((g) => g.key)).toEqual(["gate", "clarify", "re-state", "retest", "conflict"]);
  });

  it("提醒类不产生 TODO", () => {
    const groups = projectTodoGroups([
      item("stall_followup"),
      item("need_intervene"),
      item("session_message"),
    ]);
    expect(groups.every((g) => g.items.length === 0)).toBe(true);
  });

  it("空输入=五组全空（dock 条计 0）", () => {
    const groups = projectTodoGroups([]);
    expect(groups.length).toBe(5);
    expect(groups.reduce((s, g) => s + g.items.length, 0)).toBe(0);
  });
});
