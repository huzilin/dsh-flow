import { useEffect, useMemo, useRef, useState } from "react";
import type { ArtifactItem, IssueItem } from "../api/proto/api";
import {
  inSelection,
  scanFenceBlocks,
  selectionLabel,
  splitContentLines,
} from "../domain/flowview";
import { assembleCommentAnchor, issueAnchorOrder } from "../api/comments";
import { useArtifactContentQuery, useCommentMutation } from "../hooks/queries";
import { AlertBanner } from "./AlertBanner";
import { CommentCard } from "./CommentCard";

/**
 * 产物阅读器（模块 II ③ 产物区点击弹出）：
 * - document：行号渲染（1-based）+ 懒加载分段（每段 200 行，滚动到底自动续载——
 *   选「懒加载分段」而非虚拟滚动：实现轻、DOM 保留完整选区高亮，seed 场景文档一次载完）；
 *   轻量 Markdown 渲染不做——行号直映原文是锚点防漂移的前提，渲染层转换会破坏 line↔内容对应。
 * - block 锚点：按围栏（```/~~~）扫描代码块，块首行「选代码块」钮整块选取
 *   （后端 ValidateIssueAnchor 要求 lang 非空，无围栏文档无 block 入口）。
 * - link：外链卡（不拉正文）。
 * - 底部固定输入框：document 默认态 / 选行（click）/ 选区间（click+shift-click）/ 选块。
 * - 锚定评论就地渲染：line=行下方、range/block=末行下方、document 级=正文末尾。
 */
const CHUNK_SIZE = 200;
const INITIAL_CHUNKS = 3;

export function ArtifactReader({
  requirementId,
  nodeId,
  artifact,
  issues,
  onClose,
}: {
  requirementId: string;
  nodeId: string;
  artifact: ArtifactItem;
  /** 该节点全部评论（阅读器只取本文物锚点 + document 级） */
  issues: IssueItem[];
  onClose: () => void;
}) {
  const isDocument = artifact.kind !== "link";
  const content = useArtifactContentQuery(requirementId, artifact.artifactId, isDocument);
  const comment = useCommentMutation(requirementId);

  const [selection, setSelection] = useState<SelectionState | null>(null);
  const [text, setText] = useState("");
  const [visibleChunks, setVisibleChunks] = useState(INITIAL_CHUNKS);
  const sentinelRef = useRef<HTMLDivElement | null>(null);

  const lines = useMemo(() => splitContentLines(content.data?.content ?? ""), [content.data]);
  const fences = useMemo(() => scanFenceBlocks(lines), [lines]);
  const shownCount = Math.min(visibleChunks * CHUNK_SIZE, lines.length);

  // 本文物语境评论：锚定本文物的（按行就位）+ document 级（末尾整列）
  const readerIssues = useMemo(
    () => issues.filter((i) => i.anchorKind === "document" || i.anchorArtifactId === artifact.artifactId),
    [issues, artifact.artifactId],
  );
  const documentIssues = useMemo(
    () => readerIssues.filter((i) => i.anchorKind === "document"),
    [readerIssues],
  );

  // 滚动到底自动续载（IntersectionObserver；测试/旧环境无该 API 时兜底「加载后续」按钮）
  useEffect(() => {
    const el = sentinelRef.current;
    if (!el || !isDocument || typeof IntersectionObserver === "undefined") return;
    if (shownCount >= lines.length) return;
    const observer = new IntersectionObserver((entries) => {
      if (entries.some((e) => e.isIntersecting)) setVisibleChunks((n) => n + 1);
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, [isDocument, shownCount, lines.length]);

  if (!isDocument) {
    return (
      <div className="reader-overlay" onMouseDown={onClose}>
        <div
          role="dialog"
          aria-modal="true"
          aria-label={`产物：${artifact.label}`}
          className="reader-modal"
          onMouseDown={(e) => e.stopPropagation()}
        >
          <ReaderHeader artifact={artifact} onClose={onClose} />
          <div className="link-card">
            <div className="muted">外链产物（link）不提供站内正文与行号锚点</div>
            <a href={artifact.href} target="_blank" rel="noreferrer">
              {artifact.href}
            </a>
          </div>
          <CommentThread issues={documentIssues} requirementId={requirementId} />
        </div>
      </div>
    );
  }

  const allShown = shownCount >= lines.length;

  return (
    <div className="reader-overlay" onMouseDown={onClose}>
      <div
        role="dialog"
        aria-modal="true"
        aria-label={`产物：${artifact.label}`}
        className="reader-modal"
        onMouseDown={(e) => e.stopPropagation()}
      >
        <ReaderHeader artifact={artifact} onClose={onClose} />
        {content.isLoading && <div className="muted reader-loading">正文加载中…</div>}
        {content.error && (
          <AlertBanner error={content.error} onRetry={() => void content.refetch()} />
        )}
        <div className="reader-body">
          <div className="reader-lines">
            {lines.slice(0, shownCount).map((line) => {
              const fenceStart = fences.find((f) => f.startLine === line.no);
              return (
                <div key={line.no}>
                  <div
                    className={[
                      "reader-line",
                      inSelection(line.no, selection) ? "in-selection" : "",
                      fenceStart ? "fence-start" : "",
                    ].join(" ")}
                    data-line={line.no}
                    onClick={(e) =>
                      setSelection((prev) =>
                        e.shiftKey && prev && (prev.kind === "line" || prev.kind === "range")
                          ? {
                              kind: "range",
                              start: Math.min(prev.start, line.no),
                              end: Math.max(prev.start, line.no),
                            }
                          : { kind: "line", start: line.no, end: line.no },
                      )
                    }
                  >
                    <span className="line-no">{line.no}</span>
                    <span className="line-text">{line.text === "" ? " " : line.text}</span>
                  </div>
                  {fenceStart && (
                    <button
                      type="button"
                      className="pick-block"
                      title={`整块选取代码块（${fenceStart.lang || "未标注语言"}）发起 block 锚点评论`}
                      onClick={() =>
                        setSelection({
                          kind: "block",
                          start: fenceStart.startLine,
                          end: fenceStart.endLine,
                          lang: fenceStart.lang,
                        })
                      }
                    >
                      ⬚ 选代码块（{fenceStart.startLine}-{fenceStart.endLine}）
                    </button>
                  )}
                  <AnchorCommentsAt
                    position={line.no}
                    issues={readerIssues}
                    requirementId={requirementId}
                  />
                </div>
              );
            })}
            {!allShown && (
              <div className="reader-more">
                <button type="button" onClick={() => setVisibleChunks((n) => n + 1)}>
                  加载后续 {Math.min(CHUNK_SIZE, lines.length - shownCount)} 行
                </button>
                <span className="muted">
                  （已渲染 {shownCount}/{lines.length} 行 · 懒加载分段）
                </span>
              </div>
            )}
          </div>
          <CommentThread issues={documentIssues} requirementId={requirementId} />
          <div ref={sentinelRef} />
        </div>
        <div className="reader-footer">
          {selection && (
            <button type="button" onClick={() => setSelection(null)}>
              清除选区
            </button>
          )}
          <span className="reader-selection muted">{selectionLabel(selection)}</span>
          <textarea
            aria-label="评论内容"
            placeholder={
              selection ? "评论内容（必填）" : "未选行时默认全文评论（document 级锚点）"
            }
            value={text}
            onChange={(e) => setText(e.target.value)}
            rows={2}
          />
          <button
            type="button"
            className="primary"
            disabled={text.trim() === "" || comment.isPending}
            onClick={() =>
              comment.mutate(
                {
                  nodeId,
                  anchor: assembleCommentAnchor(
                    selection
                      ? {
                          kind: selection.kind,
                          artifactId: artifact.artifactId,
                          line: selection.start,
                          startLine: selection.start,
                          endLine: selection.end,
                          blockStart: selection.start,
                          blockEnd: selection.end,
                          lang: selection.lang ?? "",
                        }
                      : { kind: "document" },
                  ),
                  text,
                },
                {
                  onSuccess: () => {
                    setText("");
                    setSelection(null);
                  },
                },
              )
            }
          >
            发表评论
          </button>
          {comment.error && (
            <div role="alert" className="field-hint error">
              {comment.error instanceof Error ? comment.error.message : String(comment.error)}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

/** 选区状态（UI 态；发表时经 assembleCommentAnchor 转契约锚点）。 */
interface SelectionState {
  kind: "line" | "range" | "block";
  start: number;
  end: number;
  lang?: string;
}

function ReaderHeader({ artifact, onClose }: { artifact: ArtifactItem; onClose: () => void }) {
  return (
    <div className="reader-header">
      <b>{artifact.label}</b>
      <span className="muted">
        {artifact.productType} · {artifact.kind === "link" ? "外链" : "文档"}
        {artifact.href !== "" && artifact.kind !== "link" && ` · ${artifact.href}`}
      </span>
      <button type="button" aria-label="关闭阅读器" onClick={onClose}>
        ✕
      </button>
    </div>
  );
}

/** 行位锚定评论（line=行下、range/block=末行下）。 */
function AnchorCommentsAt({
  position,
  issues,
  requirementId,
}: {
  position: number;
  issues: IssueItem[];
  requirementId: string;
}) {
  const at = issues
    .filter((i) => issueAnchorOrder(i) === position)
    .sort((a, b) => Number(a.createdAt) - Number(b.createdAt));
  if (at.length === 0) return null;
  return (
    <div className="anchor-comments">
      {at.map((issue) => (
        <CommentCard
          key={issue.issueId}
          requirementId={requirementId}
          issue={issue}
          showAnchor={false}
        />
      ))}
    </div>
  );
}

/** document 级评论整列（阅读器末尾；意见区复用同列表形态）。 */
export function CommentThread({
  issues,
  requirementId,
  highlightIssueId,
}: {
  issues: IssueItem[];
  requirementId: string;
  /** 收件箱跳转锚点高亮（T16：focus=issue 定位） */
  highlightIssueId?: string;
}) {
  if (issues.length === 0) return null;
  return (
    <div className="comments-thread">
      {issues
        .slice()
        .sort((a, b) => Number(a.createdAt) - Number(b.createdAt))
        .map((issue) => (
          <CommentCard
            key={issue.issueId}
            requirementId={requirementId}
            issue={issue}
            highlight={highlightIssueId === issue.issueId}
          />
        ))}
    </div>
  );
}
