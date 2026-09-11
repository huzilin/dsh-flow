import { describe, expect, it } from "vitest";
import { parseInline, parseMarkdown } from "./mdview";

/** 轻量 Markdown 视图模型表驱动测试（BUG-012 描述卡渲染口径）。 */

describe("parseInline", () => {
  it("纯文本原样保真", () => {
    expect(parseInline("无任何标记的一行")).toEqual([{ t: "text", v: "无任何标记的一行" }]);
  });
  it("粗体 **…**", () => {
    expect(parseInline("前 **加粗** 后")).toEqual([
      { t: "text", v: "前 " },
      { t: "bold", v: "加粗" },
      { t: "text", v: " 后" },
    ]);
  });
  it("行内代码 `…`（内容原样不解析）", () => {
    expect(parseInline("用 `wb_comment` 接口")).toEqual([
      { t: "text", v: "用 " },
      { t: "code", v: "wb_comment" },
      { t: "text", v: " 接口" },
    ]);
  });
  it("链接 [文本](http(s)://…)", () => {
    expect(parseInline("见 [设计文档](https://example.com/a) 与 [仓库](http://x.dev/p)")).toEqual([
      { t: "text", v: "见 " },
      { t: "link", v: "设计文档", href: "https://example.com/a" },
      { t: "text", v: " 与 " },
      { t: "link", v: "仓库", href: "http://x.dev/p" },
    ]);
  });
  it("非 http(s) scheme 链接降级纯文本（防 javascript: 注入，文本往返保真）", () => {
    const src = "[点我](javascript:alert(1))";
    const tokens = parseInline(src);
    expect(tokens.some((tk) => tk.t === "link")).toBe(false);
    // 降级后拼接与原文一致（不丢字符）
    expect(tokens.map((tk) => tk.v).join("")).toBe(src);
  });
  it("混排先到先得", () => {
    expect(parseInline("**b** 与 `c` 与 [l](https://e.co)")).toEqual([
      { t: "bold", v: "b" },
      { t: "text", v: " 与 " },
      { t: "code", v: "c" },
      { t: "text", v: " 与 " },
      { t: "link", v: "l", href: "https://e.co" },
    ]);
  });
});

describe("parseMarkdown", () => {
  it("空串零块", () => {
    expect(parseMarkdown("")).toEqual([]);
  });
  it("标题 1-6 级与行内解析", () => {
    const blocks = parseMarkdown("## 目标：**降本**\n### `wbflow` v1.0");
    expect(blocks).toEqual([
      { kind: "heading", level: 2, text: [{ t: "text", v: "目标：" }, { t: "bold", v: "降本" }] },
      { kind: "heading", level: 3, text: [{ t: "code", v: "wbflow" }, { t: "text", v: " v1.0" }] },
    ]);
  });
  it("无序/有序列表连续段聚合，类型切换断段", () => {
    const blocks = parseMarkdown("- 甲\n- 乙\n1. 一\n2. 二\n");
    expect(blocks).toEqual([
      { kind: "list", ordered: false, items: [[{ t: "text", v: "甲" }], [{ t: "text", v: "乙" }]] },
      { kind: "list", ordered: true, items: [[{ t: "text", v: "一" }], [{ t: "text", v: "二" }]] },
    ]);
  });
  it("围栏代码块整块保留（lang 与内容原样，不做行内解析）", () => {
    const blocks = parseMarkdown("```sql\nSELECT **a** FROM t;\n```");
    expect(blocks).toEqual([{ kind: "code", lang: "sql", text: "SELECT **a** FROM t;" }]);
  });
  it("围栏未闭合按代码块收尾不丢内容", () => {
    const blocks = parseMarkdown("~~~\ncode line");
    expect(blocks).toEqual([{ kind: "code", lang: "", text: "code line" }]);
  });
  it("段落按空行断开，段内换行 \\n 保真", () => {
    const blocks = parseMarkdown("第一行\n第二行\n\n第三行");
    expect(blocks).toEqual([
      { kind: "paragraph", text: [{ t: "text", v: "第一行\n第二行" }] },
      { kind: "paragraph", text: [{ t: "text", v: "第三行" }] },
    ]);
  });
  it("综合：标题+段落+列表+代码块混排", () => {
    const src = [
      "# 需求说明",
      "",
      "目标是 **统一口径**，字段 `status` 单向。",
      "",
      "- 前置：A",
      "- 后置：B",
      "",
      "```ts",
      "const x = 1;",
      "```",
    ].join("\n");
    const blocks = parseMarkdown(src);
    expect(blocks.map((b) => b.kind)).toEqual(["heading", "paragraph", "list", "code"]);
  });
});

describe("OPT-005 扩容：表格与引用块", () => {
  it("管道表格：表头+分隔行+数据行 → table 块（单元格行内解析）", () => {
    const blocks = parseMarkdown(
      "| 断言维度 | 口径 |\n|---|---|\n| 流程位置 | `current_node_id` 前移 |\n| 状态联动 | **进行中**→已完成 |",
    );
    expect(blocks.length).toBe(1);
    const t = blocks[0]!;
    expect(t.kind).toBe("table");
    if (t.kind !== "table") return;
    expect(t.header.map((c) => c.map((tk) => tk.v).join(""))).toEqual(["断言维度", "口径"]);
    expect(t.rows.length).toBe(2);
    expect(t.rows[0]![1]!.some((tk) => tk.t === "code")).toBe(true);
    expect(t.rows[1]![1]!.some((tk) => tk.t === "bold")).toBe(true);
  });

  it("非表格竖线行（无分隔行）不误判为表格", () => {
    const blocks = parseMarkdown("a | b");
    expect(blocks[0]?.kind).toBe("paragraph");
  });

  it("引用块：连续 > 行聚合为单 blockquote（注记-L3 同批拍板）", () => {
    const blocks = parseMarkdown("> 复测口径：每步看 status。\n> 第二行延续。\n\n正文");
    expect(blocks.length).toBe(2);
    expect(blocks[0]?.kind).toBe("blockquote");
    if (blocks[0]?.kind !== "blockquote") return;
    expect(blocks[0]!.text.some((tk) => tk.t === "text" && tk.v.includes("第二行延续"))).toBe(true);
    expect(blocks[1]?.kind).toBe("paragraph");
  });
});
