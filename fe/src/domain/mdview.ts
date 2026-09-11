/**
 * 轻量 Markdown 视图模型（BUG-012 描述卡渲染源；纯函数，禁 IO）。
 * 覆盖集（与 ArtifactReader「不引重库」同口径）：标题/无序+有序列表/围栏代码块/段落；
 * 行内：粗体/行内代码/链接。产物阅读器不用此渲染（行号直映原文是锚点防漂移前提，
 * 见 ArtifactReader 头注）。
 * 安全性：输出结构化 token 交 React 渲染，不拼 HTML 字符串；链接仅放行 http(s)
 * （其余 scheme 的链接按纯文本呈现，防 javascript: 注入）。
 */

/** 行内 token：text=纯文本 / code=行内代码 / bold=粗体 / link=[文本](href)。 */
export type MdInline =
  | { t: "text"; v: string }
  | { t: "code"; v: string }
  | { t: "bold"; v: string }
  | { t: "link"; v: string; href: string };

/** 块级 token：heading（level 1-6）/ paragraph / list（items，每项=行内 token 序列）/ code（围栏块）
 *  / table（表头+行，单元格=行内 token 序列；OPT-005 对齐老 renderMd）/ blockquote（引用块，
 *  连续 > 行聚合，注记-L3 同批 User 拍板支持）。 */
export type MdBlock =
  | { kind: "heading"; level: number; text: MdInline[] }
  | { kind: "paragraph"; text: MdInline[] }
  | { kind: "list"; ordered: boolean; items: MdInline[][] }
  | { kind: "code"; lang: string; text: string }
  | { kind: "table"; header: MdInline[][]; rows: MdInline[][][] }
  | { kind: "blockquote"; text: MdInline[] };

/** http(s) 外才可点击（其余 scheme 原样文本呈现）。 */
function safeHref(href: string): string | null {
  return /^https?:\/\//.test(href) ? href : null;
}

/** 行内解析：单趟正则扫描（代码/粗体/链接，先到先得；源串原样保真）。 */
export function parseInline(text: string): MdInline[] {
  const out: MdInline[] = [];
  const re = /(`([^`]+)`)|(\*\*([^*]+)\*\*)|(\[([^\]]+)\]\(([^)\s]+)\))/g;
  let last = 0;
  for (let m = re.exec(text); m !== null; m = re.exec(text)) {
    if (m.index > last) out.push({ t: "text", v: text.slice(last, m.index) });
    if (m[2] !== undefined) {
      out.push({ t: "code", v: m[2] });
    } else if (m[4] !== undefined) {
      out.push({ t: "bold", v: m[4] });
    } else if (m[6] !== undefined && m[7] !== undefined) {
      const href = safeHref(m[7]);
      if (href !== null) out.push({ t: "link", v: m[6], href });
      else out.push({ t: "text", v: m[0] });
    }
    last = m.index + m[0].length;
  }
  if (last < text.length) out.push({ t: "text", v: text.slice(last) });
  return out;
}

/** 围栏开/闭行（``` 或 ~~~，起止标记须同款；与 flowview.scanFenceBlocks 同口径）。 */
function fenceLine(text: string): { marker: string; lang: string } | null {
  const m = /^\s*(```|~~~)\s*(\S*)\s*$/.exec(text);
  if (!m) return null;
  return { marker: m[1] ?? "", lang: m[2] ?? "" };
}

/** 块级解析：逐行扫描（标题/列表连续段聚合/围栏代码块/段落按空行断开；
 *  段内换行以 \n 保真，由渲染层 pre-wrap 呈现）。 */
export function parseMarkdown(src: string): MdBlock[] {
  const blocks: MdBlock[] = [];
  const lines = src === "" ? [] : src.split("\n");
  let para: string[] = [];
  let list: { ordered: boolean; items: MdInline[][] } | null = null;
  let code: { marker: string; lang: string; lines: string[] } | null = null;
  let quote: string[] | null = null;

  const flushPara = () => {
    if (para.length === 0) return;
    blocks.push({ kind: "paragraph", text: parseInline(para.join("\n")) });
    para = [];
  };
  const flushList = () => {
    if (list === null) return;
    blocks.push({ kind: "list", ordered: list.ordered, items: list.items });
    list = null;
  };
  const flushQuote = () => {
    if (quote === null) return;
    blocks.push({ kind: "blockquote", text: parseInline(quote.join("\n")) });
    quote = null;
  };
  const flushAll = () => {
    flushPara();
    flushList();
    flushQuote();
  };

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i] ?? "";
    if (code !== null) {
      const fence = fenceLine(line);
      if (fence && fence.marker === code.marker) {
        blocks.push({ kind: "code", lang: code.lang, text: code.lines.join("\n") });
        code = null;
      } else {
        code.lines.push(line);
      }
      continue;
    }
    const fence = fenceLine(line);
    if (fence) {
      flushAll();
      code = { marker: fence.marker, lang: fence.lang, lines: [] };
      continue;
    }
    if (line.trim() === "") {
      flushAll();
      continue;
    }
    // 引用块（OPT-005）：连续 > 行聚合为单 blockquote，行间 \n 保真
    const quoted = /^\s*>\s?(.*)$/.exec(line);
    if (quoted) {
      flushPara();
      flushList();
      if (quote === null) quote = [];
      quote.push(quoted[1] ?? "");
      continue;
    }
    flushQuote();
    // 管道表格（OPT-005 对齐老 renderMd）：首行 |…| + 次行分隔行 → 聚合至非 | 行
    if (/^\s*\|/.test(line)) {
      const sep = /^\s*\|?[\s:|-]+\|?\s*$/.test(lines[i + 1] ?? "") && /-/.test(lines[i + 1] ?? "");
      if (sep) {
        flushAll();
        const splitRow = (row: string): string[] =>
          row.trim().replace(/^\|/, "").replace(/\|$/, "").split("|").map((c) => c.trim());
        const header = splitRow(line).map((c) => parseInline(c));
        const rows: MdInline[][][] = [];
        let j = i + 2;
        while (j < lines.length && /^\s*\|/.test(lines[j] ?? "")) {
          rows.push(splitRow(lines[j] ?? "").map((c) => parseInline(c)));
          j++;
        }
        blocks.push({ kind: "table", header, rows });
        i = j - 1;
        continue;
      }
    }
    const heading = /^(#{1,6})\s+(.+?)\s*#*\s*$/.exec(line);
    if (heading) {
      flushAll();
      blocks.push({
        kind: "heading",
        level: (heading[1] ?? "#").length,
        text: parseInline(heading[2] ?? ""),
      });
      continue;
    }
    const unordered = /^\s*[-*+]\s+(.+)$/.exec(line);
    const ordered = /^\s*\d+[.)]\s+(.+)$/.exec(line);
    if (unordered || ordered) {
      flushPara();
      const isOrdered = ordered !== null;
      if (list === null || list.ordered !== isOrdered) {
        flushList();
        list = { ordered: isOrdered, items: [] };
      }
      list.items.push(parseInline((ordered ?? unordered)?.[1] ?? ""));
      continue;
    }
    flushList();
    para.push(line);
  }
  // 围栏未闭合：按代码块收尾（展示层降级，不丢内容）
  if (code !== null) {
    blocks.push({ kind: "code", lang: code.lang, text: code.lines.join("\n") });
  }
  flushAll();
  return blocks;
}
