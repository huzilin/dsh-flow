import { useMemo, type ReactNode } from "react";
import { parseMarkdown, type MdBlock, type MdInline } from "../domain/mdview";

/**
 * 轻量 Markdown 渲染（BUG-012 描述卡消费 mdview 结构化 token）：
 * 纯 React 元素拼装、不拼 HTML 字符串（无 dangerouslySetInnerHTML，天然免注入）；
 * 链接仅 http(s) 可点（mdview.safeHref 收口），统一新窗打开。
 */
export function MarkdownView({ source }: { source: string }) {
  const blocks = useMemo(() => parseMarkdown(source), [source]);
  return (
    <div className="md-view">
      {blocks.map((b, i) => (
        <MarkdownBlockView key={i} block={b} />
      ))}
    </div>
  );
}

function MarkdownBlockView({ block }: { block: MdBlock }) {
  switch (block.kind) {
    case "heading": {
      const level = Math.min(6, Math.max(1, block.level));
      return createElementWrap(level, <InlineTokens tokens={block.text} />);
    }
    case "paragraph":
      return (
        <p className="md-paragraph">
          <InlineTokens tokens={block.text} />
        </p>
      );
    case "list":
      return block.ordered ? (
        <ol className="md-list">
          {block.items.map((item, i) => (
            <li key={i}>
              <InlineTokens tokens={item} />
            </li>
          ))}
        </ol>
      ) : (
        <ul className="md-list">
          {block.items.map((item, i) => (
            <li key={i}>
              <InlineTokens tokens={item} />
            </li>
          ))}
        </ul>
      );
    case "code":
      return (
        <pre className="md-code">
          <code data-lang={block.lang}>{block.text}</code>
        </pre>
      );
    case "table":
      // OPT-005：管道表格（对齐老 renderMd——设计文档大量表格，缺渲染退化成散文）
      return (
        <table className="md-table">
          <thead>
            <tr>
              {block.header.map((cell, ci) => (
                <th key={ci}>
                  <InlineTokens tokens={cell} />
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {block.rows.map((row, ri) => (
              <tr key={ri}>
                {row.map((cell, ci) => (
                  <td key={ci}>
                    <InlineTokens tokens={cell} />
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      );
    case "blockquote":
      // OPT-005：引用块（注记-L3 同批拍板支持）
      return (
        <blockquote className="md-quote">
          <InlineTokens tokens={block.text} />
        </blockquote>
      );
  }
}

/** 标题层级 1-6（越界收敛，h1 自带视觉降级 class 便于描述卡内缩进排版）。 */
function createElementWrap(level: number, children: ReactNode) {
  switch (level) {
    case 1:
      return <h1 className="md-heading md-h1">{children}</h1>;
    case 2:
      return <h2 className="md-heading md-h2">{children}</h2>;
    case 3:
      return <h3 className="md-heading md-h3">{children}</h3>;
    case 4:
      return <h4 className="md-heading md-h4">{children}</h4>;
    case 5:
      return <h5 className="md-heading md-h5">{children}</h5>;
    default:
      return <h6 className="md-heading md-h6">{children}</h6>;
  }
}

function InlineTokens({ tokens }: { tokens: MdInline[] }) {
  return (
    <>
      {tokens.map((tk, i) => {
        switch (tk.t) {
          case "code":
            return <code key={i} className="md-inline-code">{tk.v}</code>;
          case "bold":
            return <strong key={i}>{tk.v}</strong>;
          case "link":
            return (
              <a key={i} href={tk.href} target="_blank" rel="noreferrer">
                {tk.v}
              </a>
            );
          default:
            return <span key={i}>{tk.v}</span>;
        }
      })}
    </>
  );
}
