import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CreateRequirementDialog } from "./CreateRequirementDialog";
import type { CreateRequirementReq, VersionItem } from "../api/proto/api";

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
