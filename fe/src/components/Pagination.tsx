/** 分页控件（后端强制分页：page 1 起，page_size 上限 100）。 */
export function Pagination({
  page,
  pageSize,
  total,
  onPageChange,
  onPageSizeChange,
}: {
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
}) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  return (
    <div className="pagination">
      <span className="muted">
        共 {total} 条 · 第 {page}/{totalPages} 页
      </span>
      <button type="button" disabled={page <= 1} onClick={() => onPageChange(page - 1)}>
        上一页
      </button>
      <button type="button" disabled={page >= totalPages} onClick={() => onPageChange(page + 1)}>
        下一页
      </button>
      <select
        aria-label="每页条数"
        value={pageSize}
        onChange={(e) => onPageSizeChange(Number(e.target.value))}
      >
        {[20, 50, 100].map((n) => (
          <option key={n} value={n}>
            {n} 条/页
          </option>
        ))}
      </select>
    </div>
  );
}
