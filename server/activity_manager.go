package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/webhook"
	"go.uber.org/zap"
)

// ActivityManager is the activity manager.
type ActivityManager struct {
	s               *Server
	activityService api.ActivityService
}

// ActivityMeta is the activity metadata.
type ActivityMeta struct {
	issue *api.Issue
}

// NewActivityManager creates an activity manager.
func NewActivityManager(server *Server, activityService api.ActivityService) *ActivityManager {
	return &ActivityManager{
		s:               server,
		activityService: activityService,
	}
}

// CreateActivity creates an activity.
func (m *ActivityManager) CreateActivity(ctx context.Context, create *api.ActivityCreate, meta *ActivityMeta) (*api.Activity, error) {
	activity, err := m.activityService.CreateActivity(ctx, create)
	if err != nil {
		return nil, err
	}

	if meta.issue == nil {
		return activity, nil
	}
	postInbox, err := shouldPostInbox(activity, create.Type)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to post webhook event after changing the issue task status: %s", meta.issue.Name)
	}
	if postInbox {
		if err := m.s.postInboxIssueActivity(ctx, meta.issue, activity.ID); err != nil {
			return nil, err
		}
	}

	hookFind := &api.ProjectWebhookFind{
		ProjectID:    &meta.issue.ProjectID,
		ActivityType: &create.Type,
	}
	hookList, err := m.s.ProjectWebhookService.FindProjectWebhookList(ctx, hookFind)
	if err != nil {
		return nil, fmt.Errorf("failed to find project webhook after changing the issue status: %v, error: %w", meta.issue.Name, err)
	}
	if len(hookList) == 0 {
		return activity, nil
	}

	// If we need to post webhook event, then we need to make sure the project info exists since we will include
	// the project name in the webhook event.
	if meta.issue.Project == nil {
		projectFind := &api.ProjectFind{
			ID: &meta.issue.ProjectID,
		}
		meta.issue.Project, err = m.s.ProjectService.FindProject(ctx, projectFind)
		if err != nil {
			return nil, fmt.Errorf("failed to find project for posting webhook event after changing the issue status: %v, error: %w", meta.issue.Name, err)
		}
	}

	principalFind := &api.PrincipalFind{
		ID: &create.CreatorID,
	}
	updater, err := m.s.PrincipalService.FindPrincipal(ctx, principalFind)
	if err != nil {
		return nil, fmt.Errorf("failed to find updater for posting webhook event after changing the issue status: %v, error: %w", meta.issue.Name, err)
	}

	// Call external webhook endpoint in Go routine to avoid blocking web serveing thread.
	go func() {
		webhookCtx, err := m.getWebhookContext(ctx, activity, meta, updater)
		if err != nil {
			return
		}

		for _, hook := range hookList {
			webhookCtx.URL = hook.URL
			webhookCtx.CreatedTs = time.Now().Unix()
			if err := webhook.Post(hook.Type, webhookCtx); err != nil {
				// The external webhook endpoint might be invalid which is out of our code control, so we just emit a warning
				m.s.l.Warn("Failed to post webhook event after changing the issue status",
					zap.String("webhook_type", hook.Type),
					zap.String("webhook_name", hook.Name),
					zap.String("issue_name", meta.issue.Name),
					zap.String("status", string(meta.issue.Status)),
					zap.Error(err))
			}
		}
	}()

	return activity, nil
}

func (m *ActivityManager) getWebhookContext(ctx context.Context, activity *api.Activity, meta *ActivityMeta, updater *api.Principal) (webhook.Context, error) {
	var webhookCtx webhook.Context
	level := webhook.WebhookInfo
	title := ""
	link := fmt.Sprintf("%s:%d/issue/%s", m.s.frontendHost, m.s.frontendPort, api.IssueSlug(meta.issue))
	switch activity.Type {
	case api.ActivityIssueCreate:
		title = "Issue created - " + meta.issue.Name
	case api.ActivityIssueStatusUpdate:
		switch meta.issue.Status {
		case "OPEN":
			title = "Issue reopened - " + meta.issue.Name
		case "DONE":
			level = webhook.WebhookSuccess
			title = "Issue resolved - " + meta.issue.Name
		case "CANCELED":
			title = "Issue canceled - " + meta.issue.Name
		}
	case api.ActivityIssueCommentCreate:
		title = "Comment created"
		link += fmt.Sprintf("#activity%d", activity.ID)
	case api.ActivityIssueFieldUpdate:
		update := new(api.ActivityIssueFieldUpdatePayload)
		if err := json.Unmarshal([]byte(activity.Payload), update); err != nil {
			m.s.l.Warn("Failed to post webhook event after changing the issue field, failed to unmarshal payload",
				zap.String("issue_name", meta.issue.Name),
				zap.Error(err))
			return webhookCtx, err
		}
		switch update.FieldID {
		case api.IssueFieldAssignee:
			{
				var oldAssignee, newAssignee *api.Principal
				if update.OldValue != "" {
					oldID, err := strconv.Atoi(update.OldValue)
					if err != nil {
						m.s.l.Warn("Failed to post webhook event after changing the issue assignee, old assignee id is not number",
							zap.String("issue_name", meta.issue.Name),
							zap.String("old_assignee_id", update.OldValue),
							zap.Error(err))
						return webhookCtx, err
					}
					principalFind := &api.PrincipalFind{
						ID: &oldID,
					}
					oldAssignee, err = m.s.PrincipalService.FindPrincipal(ctx, principalFind)
					if err != nil {
						m.s.l.Warn("Failed to post webhook event after changing the issue assignee, failed to find old assignee",
							zap.String("issue_name", meta.issue.Name),
							zap.String("old_assignee_id", update.OldValue),
							zap.Error(err))
						return webhookCtx, err
					}
				}

				if update.NewValue != "" {
					newID, err := strconv.Atoi(update.NewValue)
					if err != nil {
						m.s.l.Warn("Failed to post webhook event after changing the issue assignee, new assignee id is not number",
							zap.String("issue_name", meta.issue.Name),
							zap.String("new_assignee_id", update.NewValue),
							zap.Error(err))
						return webhookCtx, err
					}
					principalFind := &api.PrincipalFind{
						ID: &newID,
					}
					newAssignee, err = m.s.PrincipalService.FindPrincipal(ctx, principalFind)
					if err != nil {
						m.s.l.Warn("Failed to post webhook event after changing the issue assignee, failed to find new assignee",
							zap.String("issue_name", meta.issue.Name),
							zap.String("new_assignee_id", update.NewValue),
							zap.Error(err))
						return webhookCtx, err
					}

					if oldAssignee != nil && newAssignee != nil {
						title = fmt.Sprintf("Reassigned issue from %s to %s", oldAssignee.Name, newAssignee.Name)
					} else if newAssignee != nil {
						title = fmt.Sprintf("Assigned issue to %s", newAssignee.Name)
					} else if oldAssignee != nil {
						title = fmt.Sprintf("Unassigned issue from %s", newAssignee.Name)
					}
				}
			}
		case api.IssueFieldDescription:
			title = "Changed issue description"
		case api.IssueFieldName:
			title = "Changed issue name"
		default:
			title = "Updated issue"
		}
	case api.ActivityPipelineTaskStatusUpdate:
		update := &api.ActivityPipelineTaskStatusUpdatePayload{}
		if err := json.Unmarshal([]byte(activity.Payload), update); err != nil {
			m.s.l.Warn("Failed to post webhook event after changing the issue task status, failed to unmarshal paylaod",
				zap.String("issue_name", meta.issue.Name),
				zap.Error(err))
			return webhookCtx, err
		}

		taskFind := &api.TaskFind{
			ID: &update.TaskID,
		}
		task, err := m.s.TaskService.FindTask(ctx, taskFind)
		if err != nil {
			m.s.l.Warn("Failed to post webhook event after changing the issue task status, failed to find task",
				zap.String("issue_name", meta.issue.Name),
				zap.Int("task_id", update.TaskID),
				zap.Error(err))
			return webhookCtx, err
		}

		title = "Task changed - " + task.Name
		switch update.NewStatus {
		case api.TaskPending:
			if update.OldStatus == api.TaskRunning {
				title = "Task canceled - " + task.Name
			} else if update.OldStatus == api.TaskPendingApproval {
				title = "Task approved - " + task.Name
			}
		case api.TaskRunning:
			title = "Task started - " + task.Name
		case api.TaskDone:
			level = webhook.WebhookSuccess
			title = "Task completed - " + task.Name
		case api.TaskFailed:
			level = webhook.WebhookError
			title = "Task failed - " + task.Name
		}
	}

	metaList := []webhook.Meta{
		{
			Name:  "Issue",
			Value: meta.issue.Name,
		},
		{
			Name:  "Project",
			Value: meta.issue.Project.Name,
		},
	}
	webhookCtx = webhook.Context{
		Level:        level,
		Title:        title,
		Description:  activity.Comment,
		Link:         link,
		CreatorName:  updater.Name,
		CreatorEmail: updater.Email,
		MetaList:     metaList,
	}
	return webhookCtx, nil
}

func shouldPostInbox(activity *api.Activity, createType api.ActivityType) (bool, error) {
	switch createType {
	case api.ActivityIssueCreate:
		return true, nil
	case api.ActivityIssueStatusUpdate:
		return true, nil
	case api.ActivityIssueCommentCreate:
		return true, nil
	case api.ActivityIssueFieldUpdate:
		return true, nil
	case api.ActivityPipelineTaskStatementUpdate:
		return true, nil
	case api.ActivityPipelineTaskStatusUpdate:
		update := new(api.ActivityPipelineTaskStatusUpdatePayload)
		if err := json.Unmarshal([]byte(activity.Payload), update); err != nil {
			return false, err
		}
		// To reduce noise, for now we only post status update to inbox upon task failure.
		if update.NewStatus == api.TaskFailed {
			return true, nil
		}
	}
	return false, nil
}
