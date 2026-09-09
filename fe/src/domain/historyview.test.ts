import { describe, expect, it } from "vitest";
import { describeHistory } from "./historyview";

/** 变更历史自然语句装配直测（§6 N6 histPanel 口径）。 */
describe("describeHistory（N6 自然语句）", () => {
  it("行级动作：INSERT/DELETE", () => {
    expect(describeHistory("INSERT", "", "", "")).toBe("创建了需求");
    expect(describeHistory("DELETE", "title", "旧", "")).toBe("删除了需求");
  });

  it("描述不做逐字 diff：填写/清空/修改了需求说明", () => {
    expect(describeHistory("UPDATE", "description", "", "新描述")).toBe("填写了需求说明");
    expect(describeHistory("UPDATE", "description", "旧描述", "")).toBe("清空了需求说明");
    expect(describeHistory("UPDATE", "description", "旧", "新")).toBe("修改了需求说明");
  });

  it("字段级变更语句：已知字段具名，空值用 —/待分配 兜底", () => {
    expect(describeHistory("UPDATE", "priority", "P2", "P1")).toBe('将「优先级」从「P2」变更为「P1」');
    expect(describeHistory("UPDATE", "owner", "", "rd")).toBe('将「负责人」从「—」变更为「rd」');
    expect(describeHistory("UPDATE", "version_id", "v-1", "")).toBe('将「版本」从「v-1」变更为「待分配」');
    expect(describeHistory("UPDATE", "custom_field", "a", "b")).toBe('将「custom_field」从「a」变更为「b」');
  });
});
