import { handleAdmissionAction } from './review-action-admission'
import { handleReportAction } from './review-action-report'
import { handleReviewAction } from './review-action-review'
import type {
  AdmissionActionInput,
  ReportActionInput,
  ReviewActionInput,
  WorkItemActionActor,
  WorkItemActionDeps,
  WorkItemActionInput,
} from './review-action-types'

export type {
  AdmissionActionInput,
  ReportActionInput,
  ReviewActionInput,
  WorkItemActionActor,
  WorkItemActionDeps,
  WorkItemActionInput,
}

export async function handleWorkItemAction(
  deps: WorkItemActionDeps,
  input: WorkItemActionInput,
  actor: WorkItemActionActor,
) {
  switch (input.kind) {
    case 'review':
      return handleReviewAction(deps, input, actor)
    case 'admission':
      return handleAdmissionAction(deps, input, actor)
    case 'report':
      return handleReportAction(deps, input, actor)
  }
}
