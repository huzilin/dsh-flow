/**
 * 时间展示格式化（纯函数）。原散落 pages/RequirementPoolPage（被 InboxPage 跨页
 * import，C4 归位）；新增展示面统一从此消费。
 */

/** unix 毫秒（proto string 承载）→ 本地时间串；0/非法值=「—」（NULL 列出参口径）。 */
export function formatTime(unixMs: string): string {
  const n = Number(unixMs);
  if (!Number.isFinite(n) || n === 0) return "—";
  return new Date(n).toLocaleString("zh-CN", { hour12: false });
}
