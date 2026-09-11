import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ListRequirementsParams } from "../api/requirements";
import {
  getRequirement,
  listRequirementHistory,
  listRequirements,
} from "../api/requirements";
import { listVersions } from "../api/versions";
import { myTask } from "../api/flows";
import { getArtifactContent } from "../api/artifacts";
import {
  postClarified,
  postComment,
  postConfirmReview,
  postDeclareFixed,
  postQualify,
  postReopen,
  postRevise,
  postSettle,
} from "../api/comments";
import { listInbox, postAdjudicate, postEscalate } from "../api/inbox";
import type { ListInboxParams } from "../api/inbox";
import { postDecision } from "../api/reviews";
import { api } from "../api/client";
import type { CommentAnchor, IssueItem } from "../api/proto/api";
import { WEB_USER_ROLE } from "../api/constants";

export function useRequirementsQuery(params: ListRequirementsParams) {
  return useQuery({
    queryKey: ["requirements", params],
    queryFn: () => listRequirements(api, params),
  });
}

export function useVersionsQuery() {
  return useQuery({
    queryKey: ["versions"],
    queryFn: () => listVersions(api),
  });
}

export function useRequirementQuery(requirementId: string) {
  return useQuery({
    queryKey: ["requirement", requirementId],
    queryFn: () => getRequirement(api, requirementId),
  });
}

/** 需求字段变更历史（spec §1.2 注记-D3/N6：详情页补遗卡 histPanel 数据源）。 */
export function useRequirementHistoryQuery(requirementId: string, enabled: boolean) {
  return useQuery({
    queryKey: ["requirementHistory", requirementId],
    queryFn: () => listRequirementHistory(api, requirementId),
    enabled: enabled && requirementId !== "",
  });
}

/** wb_my_task（详情页以 role=user 消费节点摘要/动作/在途任务；节点为空=未进入编排）。 */
export function useMyTaskQuery(requirementId: string, nodeId: string | null) {
  return useQuery({
    queryKey: ["myTask", requirementId, nodeId ?? ""],
    queryFn: () => myTask(api, requirementId, WEB_USER_ROLE, nodeId ?? undefined),
    enabled: requirementId !== "" && nodeId !== null,
  });
}

/** 产物正文（阅读器打开时按需拉取；enabled 关断避免弹窗未开时请求）。 */
export function useArtifactContentQuery(requirementId: string, artifactId: string, enabled: boolean) {
  return useQuery({
    queryKey: ["artifactContent", requirementId, artifactId],
    queryFn: () => getArtifactContent(api, requirementId, artifactId),
    enabled: enabled && artifactId !== "",
  });
}

/** 评论写路径统一失效聚合快照（issues 集在快照内，单数据源不双写）。 */
function useInvalidateRequirement() {
  const queryClient = useQueryClient();
  return (requirementId: string) =>
    queryClient.invalidateQueries({ queryKey: ["requirement", requirementId] });
}

export interface CommentInput {
  nodeId: string;
  anchor: CommentAnchor;
  text: string;
  issueType?: string;
}

export function useCommentMutation(requirementId: string) {
  const invalidate = useInvalidateRequirement();
  return useMutation({
    mutationFn: (input: CommentInput) =>
      postComment(api, requirementId, {
        nodeId: input.nodeId,
        anchor: input.anchor,
        author: WEB_USER_ROLE,
        roleId: WEB_USER_ROLE,
        text: input.text,
        issueType: input.issueType ?? "issue",
        assignee: "",
        escalate: false,
      }),
    onSuccess: () => invalidate(requirementId),
  });
}

/** 回复申报（NAM-09 拆分路由）：澄清侧（open/pending-confirm）走 wb_clarified，
 *  修复侧（to-fix/fixed）走 wb_declare_fixed；服务端不校验 from-state，仅路由名承载意图。 */
export function useReplyMutation(requirementId: string) {
  const invalidate = useInvalidateRequirement();
  return useMutation({
    mutationFn: (input: { issue: IssueItem; text: string; voided?: boolean }) => {
      const req = {
        by: WEB_USER_ROLE,
        roleId: WEB_USER_ROLE,
        text: input.text,
        voided: input.voided ?? false,
      };
      const fixSide = input.issue.state === "to-fix" || input.issue.state === "fixed";
      return fixSide
        ? postDeclareFixed(api, input.issue.issueId, req)
        : postClarified(api, input.issue.issueId, req);
    },
    onSuccess: () => invalidate(requirementId),
  });
}

/** 解决申报（NAM-09 拆分路由）：agree 意图走 wb_settle、fix 意图走 wb_qualify（path 参数取消）。 */
export function useResolveMutation(requirementId: string) {
  const invalidate = useInvalidateRequirement();
  return useMutation({
    mutationFn: (input: { issue: IssueItem; path: "fix" | "agree"; note?: string }) => {
      const req = {
        operator: WEB_USER_ROLE,
        note: input.note ?? "",
        escalate: false,
      };
      return input.path === "agree"
        ? postSettle(api, input.issue.issueId, req)
        : postQualify(api, input.issue.issueId, req);
    },
    onSuccess: () => invalidate(requirementId),
  });
}

export function useConfirmReviewMutation(requirementId: string) {
  const invalidate = useInvalidateRequirement();
  return useMutation({
    mutationFn: (issue: IssueItem) =>
      postConfirmReview(api, issue.issueId, { operator: WEB_USER_ROLE }),
    onSuccess: () => invalidate(requirementId),
  });
}

/** 复核退回（NAM-09 拆分路由）：澄清侧（pending-confirm）走 wb_reopen，
 *  修复侧（fixed）走 wb_revise。 */
export function useReopenMutation(requirementId: string) {
  const invalidate = useInvalidateRequirement();
  return useMutation({
    mutationFn: (input: { issue: IssueItem; text?: string }) => {
      const req = {
        operator: WEB_USER_ROLE,
        text: input.text ?? "",
      };
      return input.issue.state === "fixed"
        ? postRevise(api, input.issue.issueId, req)
        : postReopen(api, input.issue.issueId, req);
    },
    onSuccess: () => invalidate(requirementId),
  });
}

// ---------------------------------------------------------------------------
// 模块 III/IV（T16）：收件箱轮询 / 拍板 / 冲突上升裁决
// ---------------------------------------------------------------------------

/** MVP 轮询口径：30s，可经 VITE_INBOX_POLL_MS 覆盖（0/负值=关轮询）。 */
export const INBOX_POLL_INTERVAL_MS = (() => {
  const raw = Number(import.meta.env.VITE_INBOX_POLL_MS);
  return Number.isFinite(raw) && raw > 0 ? raw : 30_000;
})();

export type InboxFilter = ListInboxParams;

/** 收件箱查询（列表页消费完整出参；refetchInterval 由调用方传入实现轮询失效）。 */
export function useInboxQuery(filter: InboxFilter, refetchIntervalMs?: number) {
  return useQuery({
    queryKey: ["inbox", filter],
    queryFn: () => listInbox(api, filter),
    refetchInterval: refetchIntervalMs,
  });
}

/** 顶栏未读 badge：独立轻量查询（unread page_size=1 只取 total），与收件箱页同频率轮询。 */
export function useInboxUnreadBadge(refetchIntervalMs: number = INBOX_POLL_INTERVAL_MS) {
  return useQuery({
    queryKey: ["inbox", { unreadBadge: true }],
    queryFn: () => listInbox(api, { readFilter: "unread", pageSize: 1 }),
    refetchInterval: refetchIntervalMs,
    select: (resp) => Number(resp.total),
  });
}

/** 失效收件箱族（列表 + badge + 冲突聚合区同族 key 前缀）。 */
export function useInvalidateInbox() {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: ["inbox"] });
}

/** wb_decision：User 终审拍板（成功/失败均失效聚合快照——409 改判冲突时刷新已裁决态）。 */
export function useDecisionMutation(requirementId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { nodeId: string; verdict: "approved" | "rejected"; reason: string; branch?: string }) =>
      postDecision(api, requirementId, input.nodeId, {
        verdict: input.verdict,
        reason: input.reason,
        branch: input.branch ?? "",
      }),
    onSettled: () => queryClient.invalidateQueries({ queryKey: ["requirement", requirementId] }),
  });
}

/** wb_escalate：手动上升裁决（失效快照 + 收件箱——escalation 通知位随之落库）。 */
export function useEscalateMutation(requirementId: string) {
  const invalidateInbox = useInvalidateInbox();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { issue: IssueItem; reason?: string }) =>
      postEscalate(api, input.issue.issueId, {
        operator: WEB_USER_ROLE,
        reason: input.reason ?? "",
      }),
    onSettled: () => {
      invalidateInbox();
      queryClient.invalidateQueries({ queryKey: ["requirement", requirementId] });
    },
  });
}

export interface AdjudicateInput {
  issue: IssueItem;
  verdict: "uphold" | "override";
  conclusion: string;
}

/** wb_adjudicate：User 仲裁结论回填（失效快照 + 收件箱；reviewer 重表态由后端联动）。 */
export function useAdjudicateMutation(requirementId: string) {
  const invalidateInbox = useInvalidateInbox();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: AdjudicateInput) =>
      postAdjudicate(api, input.issue.issueId, {
        verdict: input.verdict,
        conclusion: input.conclusion,
        decidedBy: WEB_USER_ROLE,
      }),
    onSettled: () => {
      invalidateInbox();
      queryClient.invalidateQueries({ queryKey: ["requirement", requirementId] });
    },
  });
}
