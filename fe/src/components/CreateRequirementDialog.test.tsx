import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CreateRequirementDialog } from "./CreateRequirementDialog";
import type { CreateRequirementReq, RequirementItem, VersionItem } from "../api/proto/api";

/**
 * 创建需求对话框表驱动测试（T14 验收口径：类别→flow_defs_id 允许集显式选择 +
 * 标题为空保存禁用——PRD §5.2/异常场景）。
 */
function makeVersion(id: string, name: string, isBacklog = false): VersionItem {
  return { versionId: id, name, state: "待启动", isBacklog, createdAt: "1", updatedAt: "1" };
}

function renderDialog(overrides?: {
  onSubmit?: (req: CreateRequirementReq) => void;
  versions?: VersionItem[];
}) {
  const onSubmit = overrides?.onSubmit ?? vi.fn();
  const versions = overrides?.versions ?? [
    makeVersion("v-1", "V1.0"),
    makeVersion("v-bk", "待分配池", true),
  ];
  render(
    <CreateRequirementDialog
      open
      versions={versions}
      onClose={() => undefined}
      onSubmit={onSubmit}
    />,
  );
  return { onSubmit };
}

async function fillRequired(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText(/^标题/), "登录优化");
  await user.type(screen.getByLabelText(/创建人/), "user");
}

beforeEach(() => {
  // 表单状态随渲染重建，无需额外清理（每用例独立渲染）
});

const cases = [
  {
    name: "需求任务 → flowDefsId 允许集 business/tech/task，默认 business",
    category: "需求任务",
    wantDisabled: false,
    wantOptions: ["business · 业务流程", "tech · 技术流程", "task · 任务流程"],
    wantDefault: "business",
  },
  {
    name: "规划审批任务 → 允许集仅 plan（下拉锁定）",
    category: "规划审批任务",
    wantDisabled: true,
    wantOptions: ["plan · 规划流程"],
    wantDefault: "plan",
  },
  {
    name: "版本规划任务 → 允许集仅 plan（下拉锁定）",
    category: "版本规划任务",
    wantDisabled: true,
    wantOptions: ["plan · 规划流程"],
    wantDefault: "plan",
  },
];

describe("CreateRequirementDialog（表驱动）", () => {
  it.each(cases)("$name", async ({ category, wantDisabled, wantOptions, wantDefault }) => {
    const user = userEvent.setup();
    const { onSubmit } = renderDialog();

    await user.selectOptions(screen.getByLabelText(/^类别/), category);

    const flowSelect = screen.getByLabelText(/流程类型/);
    expect(flowSelect).toHaveValue(wantDefault);
    expect((flowSelect as HTMLSelectElement).disabled).toBe(wantDisabled);
    const options = Array.from((flowSelect as HTMLSelectElement).options).map((o) => o.text);
    expect(options).toEqual(wantOptions);

    await fillRequired(user);
    await user.click(screen.getByRole("button", { name: "保存" }));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ category, flowDefsId: wantDefault, title: "登录优化" }),
    );
  });

  it("需求任务显式选择 tech → 提交 flowDefsId=tech", async () => {
    const user = userEvent.setup();
    const { onSubmit } = renderDialog();
    await user.selectOptions(screen.getByLabelText(/^类别/), "需求任务");
    await user.selectOptions(screen.getByLabelText(/流程类型/), "tech");
    await fillRequired(user);
    await user.click(screen.getByRole("button", { name: "保存" }));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ category: "需求任务", flowDefsId: "tech" }),
    );
  });

  it("标题为空 → 保存禁用 + 红框提示（PRD 异常场景）", async () => {
    const user = userEvent.setup();
    const { onSubmit } = renderDialog();
    await user.selectOptions(screen.getByLabelText(/^类别/), "需求任务");
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled();
    expect(screen.getByText("标题不能为空")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "保存" }));
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("创建人为空 → 保存禁用（created_by 后端必填）", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.type(screen.getByLabelText(/^标题/), "只有标题");
    await user.selectOptions(screen.getByLabelText(/^类别/), "需求任务");
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled();
  });

  it("版本下拉不含待分配池标记行，默认待分配池（versionId 空串）", async () => {
    const user = userEvent.setup();
    const { onSubmit } = renderDialog();
    const versionSelect = screen.getByLabelText(/^版本/);
    const texts = Array.from((versionSelect as HTMLSelectElement).options).map((o) => o.text);
    expect(texts).toEqual(["待分配池（默认）", "V1.0（v-1）"]);
    await user.selectOptions(screen.getByLabelText(/^类别/), "需求任务");
    await fillRequired(user);
    await user.click(screen.getByRole("button", { name: "保存" }));
    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ versionId: "" }));
  });
});

/** 编辑模式基线行（RequirementModal §6 N4：编辑复用同弹窗）。 */
function makeRequirement(): RequirementItem {
  return {
    requirementId: "req-001",
    title: "交易对账明细导出",
    description: "按日期范围导出",
    category: "需求任务",
    flowDefsId: "business",
    versionId: "v-1",
    priority: "P1",
    status: "进行中",
    owner: "rd",
    createdBy: "user",
    createdAt: "1",
    updatedAt: "100",
    dependencies: [],
  };
}

function renderEditDialog(overrides?: {
  onUpdate?: (req: { title: string; updatedAt: string; status: string }) => void;
  initial?: RequirementItem;
  initialScope?: string;
}) {
  const onUpdate = overrides?.onUpdate ?? vi.fn();
  const initial = overrides?.initial ?? makeRequirement();
  render(
    <CreateRequirementDialog
      open
      mode="edit"
      initial={initial}
      initialScope={overrides?.initialScope ?? "full"}
      versions={[makeVersion("v-1", "V1.0")]}
      onClose={() => undefined}
      onUpdate={onUpdate}
    />,
  );
  return { onUpdate, initial };
}

describe("RequirementModal 编辑模式（§6 N4）", () => {
  it("基线回填；提交载体含 updatedAt 乐观锁基线与改后字段", async () => {
    const user = userEvent.setup();
    const { onUpdate } = renderEditDialog();
    expect(screen.getByLabelText(/^标题/)).toHaveValue("交易对账明细导出");
    await user.clear(screen.getByLabelText(/^标题/));
    await user.type(screen.getByLabelText(/^标题/), "交易对账明细导出 v2");
    await user.click(screen.getByRole("button", { name: "保存" }));
    expect(onUpdate).toHaveBeenCalledWith(
      expect.objectContaining({ title: "交易对账明细导出 v2", updatedAt: "100", status: "进行中" }),
    );
  });

  it("类别/流程类型/scope 编辑模式只读（重投影与 scope 主动切换不支持口径）", () => {
    renderEditDialog();
    expect(screen.getByLabelText(/^类别/)).toBeDisabled();
    expect(screen.getByLabelText(/^流程类型/)).toBeDisabled();
    expect(screen.getByLabelText(/^scope/)).toBeDisabled();
    expect(screen.getByLabelText(/^scope/)).toHaveValue("full");
  });

  it("创建模式：scope 下拉随流程类型派生三值，空值=默认首值", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.selectOptions(screen.getByLabelText(/^类别/), "需求任务");
    const scopeSelect = screen.getByLabelText(/scope/);
    const texts = Array.from((scopeSelect as HTMLSelectElement).options).map((o) => o.text);
    expect(texts).toEqual(["默认（full）", "full", "rd", "fe"]);
  });

  it("创建模式：plan 流程 scope 禁用置 —（N4）", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.selectOptions(screen.getByLabelText(/^类别/), "规划审批任务");
    const scopeSelect = screen.getByLabelText(/scope/);
    expect(scopeSelect).toBeDisabled();
    expect(screen.getByText("—")).toBeInTheDocument();
  });
});
