import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ListRequirementsParams } from "../api/requirements";
import { getRequirement, listRequirements } from "../api/requirements";
import { listVersions } from "../api/versions";
import { myTask } from "../api/flows";
import { getArtifactContent } from "../api/artifacts";
import {
  postComment,
  postConfirmReview,
  postReopen,
  postReply,
  postResolve,
} from "../api/comments";
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

export function useReplyMutation(requirementId: string) {
  const invalidate = useInvalidateRequirement();
  return useMutation({
    mutationFn: (input: { issue: IssueItem; text: string; voided?: boolean }) =>
      postReply(api, input.issue.issueId, {
        by: WEB_USER_ROLE,
        roleId: WEB_USER_ROLE,
        text: input.text,
        voided: input.voided ?? false,
      }),
    onSuccess: () => invalidate(requirementId),
  });
}

export function useResolveMutation(requirementId: string) {
  const invalidate = useInvalidateRequirement();
  return useMutation({
    mutationFn: (input: { issue: IssueItem; path: "fix" | "agree"; note?: string }) =>
      postResolve(api, input.issue.issueId, {
        path: input.path,
        operator: WEB_USER_ROLE,
        note: input.note ?? "",
        escalate: false,
      }),
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

export function useReopenMutation(requirementId: string) {
  const invalidate = useInvalidateRequirement();
  return useMutation({
    mutationFn: (input: { issue: IssueItem; text?: string }) =>
      postReopen(api, input.issue.issueId, {
        operator: WEB_USER_ROLE,
        text: input.text ?? "",
      }),
    onSuccess: () => invalidate(requirementId),
  });
}
