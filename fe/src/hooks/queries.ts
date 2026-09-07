import { useQuery } from "@tanstack/react-query";
import type { ListRequirementsParams } from "../api/requirements";
import { getRequirement, listRequirements } from "../api/requirements";
import { listVersions } from "../api/versions";
import { api } from "../api/client";

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
