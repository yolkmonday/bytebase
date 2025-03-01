package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerIssueRoutes(g *echo.Group) {
	g.POST("/issue", func(c echo.Context) error {
		ctx := context.Background()
		issueCreate := &api.IssueCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create issue request").SetInternal(err)
		}

		issue, err := s.createIssue(ctx, issueCreate, c.Get(getPrincipalIDContextKey()).(int))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create issue").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create issue response").SetInternal(err)
		}
		return nil
	})

	g.GET("/issue", func(c echo.Context) error {
		ctx := context.Background()
		issueFind := &api.IssueFind{}
		projectIDStr := c.QueryParams().Get("project")
		if projectIDStr != "" {
			projectID, err := strconv.Atoi(projectIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("project query parameter is not a number: %s", projectIDStr)).SetInternal(err)
			}
			issueFind.ProjectID = &projectID
		}
		if issueStatusListStr := c.QueryParam("status"); issueStatusListStr != "" {
			statusList := []api.IssueStatus{}
			for _, status := range strings.Split(issueStatusListStr, ",") {
				statusList = append(statusList, api.IssueStatus(status))
			}
			issueFind.StatusList = &statusList
		}
		if limitStr := c.QueryParam("limit"); limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("limit query parameter is not a number: %s", limitStr)).SetInternal(err)
			}
			issueFind.Limit = &limit
		}
		userIDStr := c.QueryParams().Get("user")
		if userIDStr != "" {
			userID, err := strconv.Atoi(userIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("user query parameter is not a number: %s", userIDStr)).SetInternal(err)
			}
			issueFind.PrincipalID = &userID
		}
		list, err := s.IssueService.FindIssueList(ctx, issueFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch issue list").SetInternal(err)
		}

		for _, issue := range list {
			if err := s.composeIssueRelationship(ctx, issue); err != nil {
				return err
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal issue list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/issue/:issueID", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}

		issue, err := s.composeIssueByID(ctx, id)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal issue ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/issue/:issueID", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}

		issuePatch := &api.IssuePatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issuePatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted update issue request").SetInternal(err)
		}

		issueFind := &api.IssueFind{
			ID: &id,
		}
		issue, err := s.IssueService.FindIssue(ctx, issueFind)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Unable to find issue ID to update: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID when updating issue: %v", id)).SetInternal(err)
		}

		updatedIssue, err := s.IssueService.PatchIssue(ctx, issuePatch)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update issue ID: %v", id)).SetInternal(err)
		}

		payloadList := [][]byte{}
		if issuePatch.Name != nil && *issuePatch.Name != issue.Name {
			payload, err := json.Marshal(api.ActivityIssueFieldUpdatePayload{
				FieldID:   api.IssueFieldName,
				OldValue:  issue.Name,
				NewValue:  *issuePatch.Name,
				IssueName: issue.Name,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing issue name: %v", updatedIssue.Name)).SetInternal(err)
			}
			payloadList = append(payloadList, payload)
		}
		if issuePatch.Description != nil && *issuePatch.Description != issue.Description {
			payload, err := json.Marshal(api.ActivityIssueFieldUpdatePayload{
				FieldID:   api.IssueFieldDescription,
				OldValue:  issue.Description,
				NewValue:  *issuePatch.Description,
				IssueName: issue.Name,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing issue description: %v", updatedIssue.Name)).SetInternal(err)
			}
			payloadList = append(payloadList, payload)
		}
		if issuePatch.AssigneeID != nil && *issuePatch.AssigneeID != issue.AssigneeID {
			payload, err := json.Marshal(api.ActivityIssueFieldUpdatePayload{
				FieldID:   api.IssueFieldAssignee,
				OldValue:  strconv.Itoa(issue.AssigneeID),
				NewValue:  strconv.Itoa(*issuePatch.AssigneeID),
				IssueName: issue.Name,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing issue assignee: %v", updatedIssue.Name)).SetInternal(err)
			}
			payloadList = append(payloadList, payload)
		}

		for _, payload := range payloadList {
			activityCreate := &api.ActivityCreate{
				CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
				ContainerID: issue.ID,
				Type:        api.ActivityIssueFieldUpdate,
				Level:       api.ActivityInfo,
				Payload:     string(payload),
			}
			_, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{
				issue: issue,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating issue: %v", updatedIssue.Name)).SetInternal(err)
			}
		}

		if err := s.composeIssueRelationship(ctx, updatedIssue); err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedIssue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update issue response: %v", updatedIssue.Name)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/issue/:issueID/status", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}

		issueStatusPatch := &api.IssueStatusPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueStatusPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted update issue status request").SetInternal(err)
		}

		issue, err := s.composeIssueByID(ctx, id)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID: %v", id)).SetInternal(err)
		}

		updatedIssue, err := s.changeIssueStatus(ctx, issue, issueStatusPatch.Status, issueStatusPatch.UpdaterID, issueStatusPatch.Comment)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound).SetInternal(err)
			} else if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(err)
		}

		if err := s.composeIssueRelationship(ctx, updatedIssue); err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedIssue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal issue ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) composeIssueByID(ctx context.Context, id int) (*api.Issue, error) {
	issueFind := &api.IssueFind{
		ID: &id,
	}
	issue, err := s.IssueService.FindIssue(ctx, issueFind)
	if err != nil {
		return nil, err
	}

	if err := s.composeIssueRelationship(ctx, issue); err != nil {
		return nil, err
	}

	return issue, nil
}

func (s *Server) composeIssueRelationship(ctx context.Context, issue *api.Issue) error {
	var err error

	issue.Creator, err = s.composePrincipalByID(ctx, issue.CreatorID)
	if err != nil {
		return err
	}

	issue.Updater, err = s.composePrincipalByID(ctx, issue.UpdaterID)
	if err != nil {
		return err
	}

	issue.Assignee, err = s.composePrincipalByID(ctx, issue.AssigneeID)
	if err != nil {
		return err
	}

	issueSubscriberFind := &api.IssueSubscriberFind{
		IssueID: &issue.ID,
	}
	list, err := s.IssueSubscriberService.FindIssueSubscriberList(ctx, issueSubscriberFind)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch subscriber list for issue %d", issue.ID)).SetInternal(err)
	}

	issue.SubscriberIDList = []int{}
	for _, subscriber := range list {
		issue.SubscriberIDList = append(issue.SubscriberIDList, subscriber.SubscriberID)
	}

	issue.Project, err = s.composeProjectByID(ctx, issue.ProjectID)
	if err != nil {
		return err
	}

	issue.Pipeline, err = s.composePipelineByID(ctx, issue.PipelineID)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) createIssue(ctx context.Context, issueCreate *api.IssueCreate, creatorID int) (*api.Issue, error) {
	if issueCreate.ValidateOnly {
		return s.createIssueValidateOnly(ctx, issueCreate, creatorID)
	}
	// Run pre-condition check first to make sure all tasks are valid, otherwise we will create partial pipelines
	// since we are not creating pipeline/stage list/task list in a single transaction.
	// We may still run into this issue when we actually create those pipeline/stage list/task list, however, that's
	// quite unlikely so we will live with it for now.
	if issueCreate.AssigneeID == api.UnknownID {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, assignee missing")
	}

	// If frontend does not pass the stageList, we will generate it from backend.
	if len(issueCreate.Pipeline.StageList) == 0 {
		pc, err := s.getPipelineFromIssue(ctx, issueCreate, creatorID)
		if err != nil {
			return nil, err
		}
		issueCreate.Pipeline = *pc
	}
	issueCreate.Pipeline.CreatorID = creatorID

	createdPipeline, err := s.PipelineService.CreatePipeline(ctx, &issueCreate.Pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline for issue. Error %w", err)
	}

	for _, stageCreate := range issueCreate.Pipeline.StageList {
		stageCreate.CreatorID = creatorID
		stageCreate.PipelineID = createdPipeline.ID
		createdStage, err := s.StageService.CreateStage(ctx, &stageCreate)
		if err != nil {
			return nil, fmt.Errorf("failed to create stage for issue. Error %w", err)
		}

		for _, taskCreate := range stageCreate.TaskList {
			taskCreate.CreatorID = creatorID
			taskCreate.PipelineID = createdPipeline.ID
			taskCreate.StageID = createdStage.ID
			instanceFind := &api.InstanceFind{
				ID: &taskCreate.InstanceID,
			}
			instance, err := s.InstanceService.FindInstance(ctx, instanceFind)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch instance in issue creation: %v", err)
			}
			if taskCreate.Type == api.TaskDatabaseSchemaUpdate {
				payload := api.TaskDatabaseSchemaUpdatePayload{}
				payload.MigrationType = taskCreate.MigrationType
				payload.Statement = taskCreate.Statement
				if taskCreate.RollbackStatement != "" {
					payload.RollbackStatement = taskCreate.RollbackStatement
				}
				if taskCreate.VCSPushEvent != nil {
					payload.VCSPushEvent = taskCreate.VCSPushEvent
				}
				bytes, err := json.Marshal(payload)
				if err != nil {
					return nil, fmt.Errorf("failed to create schema update task, unable to marshal payload %w", err)
				}
				taskCreate.Payload = string(bytes)
			} else if taskCreate.Type == api.TaskDatabaseRestore {
				// Snowflake needs to use upper case of DatabaseName.
				if instance.Engine == db.Snowflake {
					taskCreate.DatabaseName = strings.ToUpper(taskCreate.DatabaseName)
				}
				payload := api.TaskDatabaseRestorePayload{}
				payload.DatabaseName = taskCreate.DatabaseName
				payload.BackupID = *taskCreate.BackupID
				bytes, err := json.Marshal(payload)
				if err != nil {
					return nil, fmt.Errorf("failed to create restore database task, unable to marshal payload %w", err)
				}
				taskCreate.Payload = string(bytes)
			}
			if _, err = s.TaskService.CreateTask(ctx, &taskCreate); err != nil {
				return nil, fmt.Errorf("failed to create task for issue. Error %w", err)
			}
		}
	}

	issueCreate.CreatorID = creatorID
	issueCreate.PipelineID = createdPipeline.ID
	issue, err := s.IssueService.CreateIssue(ctx, issueCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue. Error %w", err)
	}

	createActivityPayload := api.ActivityIssueCreatePayload{
		IssueName: issue.Name,
	}
	if issueCreate.RollbackIssueID != nil {
		createActivityPayload.RollbackIssueID = *issueCreate.RollbackIssueID
	}

	bytes, err := json.Marshal(createActivityPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to create activity after creating the issue: %v. Error %w", issue.Name, err)
	}
	activityCreate := &api.ActivityCreate{
		CreatorID:   creatorID,
		ContainerID: issue.ID,
		Type:        api.ActivityIssueCreate,
		Level:       api.ActivityInfo,
		Payload:     string(bytes),
	}
	_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{
		issue: issue,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create activity after creating the issue: %v. Error %w", issue.Name, err)
	}

	// If we are creating a rollback issue, then we will also post a comment on the original issue
	if issueCreate.RollbackIssueID != nil {
		issueFind := &api.IssueFind{
			ID: issueCreate.RollbackIssueID,
		}
		rollbackIssue, err := s.IssueService.FindIssue(ctx, issueFind)
		if err != nil {
			return nil, fmt.Errorf("failed to create activity after creating the rollback issue: %v. Error %w", issue.Name, err)
		}
		bytes, err := json.Marshal(api.ActivityIssueCommentCreatePayload{
			IssueName: rollbackIssue.Name,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create activity after creating the rollback issue: %v. Error %w", issue.Name, err)
		}
		activityCreate := &api.ActivityCreate{
			CreatorID:   creatorID,
			ContainerID: *issueCreate.RollbackIssueID,
			Type:        api.ActivityIssueCommentCreate,
			Level:       api.ActivityInfo,
			Comment:     fmt.Sprintf("Created rollback issue %q", issue.Name),
			Payload:     string(bytes),
		}
		_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{
			issue: rollbackIssue,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create activity after creating the rollback issue: %v. Error %w", issue.Name, err)
		}
	}

	if err := s.composeIssueRelationship(ctx, issue); err != nil {
		return nil, err
	}

	if _, err := s.ScheduleNextTaskIfNeeded(ctx, issue.Pipeline); err != nil {
		return nil, fmt.Errorf("failed to schedule task after creating the issue: %v. Error %w", issue.Name, err)
	}

	for _, subscriberID := range issueCreate.SubscriberIDList {
		subscriberCreate := &api.IssueSubscriberCreate{
			IssueID:      issue.ID,
			SubscriberID: subscriberID,
		}
		_, err := s.IssueSubscriberService.CreateIssueSubscriber(ctx, subscriberCreate)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to add subscriber %d after creating issue %d", subscriberID, issue.ID)).SetInternal(err)
		}
	}

	return issue, nil
}

func (s *Server) createIssueValidateOnly(ctx context.Context, issueCreate *api.IssueCreate, creatorID int) (*api.Issue, error) {
	if issueCreate.AssigneeID == api.UnknownID {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, assignee missing")
	}
	pc, err := s.getPipelineFromIssue(ctx, issueCreate, creatorID)
	if err != nil {
		return nil, err
	}
	pipeline, err := s.createPipelineValidateOnly(ctx, pc)
	if err != nil {
		return nil, err
	}

	issue := &api.Issue{
		CreatorID:        creatorID,
		CreatedTs:        time.Now().Unix(),
		UpdaterID:        creatorID,
		UpdatedTs:        time.Now().Unix(),
		ProjectID:        issueCreate.ProjectID,
		Name:             issueCreate.Name,
		Status:           api.IssueOpen,
		Type:             issueCreate.Type,
		Description:      issueCreate.Description,
		AssigneeID:       issueCreate.AssigneeID,
		SubscriberIDList: issueCreate.SubscriberIDList,
		PipelineID:       pipeline.ID,
		Pipeline:         pipeline,
	}

	issueCreate.Pipeline = *pc
	issueCreate.Pipeline.CreatorID = creatorID

	return issue, nil
}

func (s *Server) createPipelineValidateOnly(ctx context.Context, pc *api.PipelineCreate) (*api.Pipeline, error) {
	pipeline := &api.Pipeline{
		Name:      pc.Name,
		Status:    api.PipelineOpen,
		CreatorID: pc.CreatorID,
		CreatedTs: time.Now().Unix(),
		UpdaterID: pc.CreatorID,
		UpdatedTs: time.Now().Unix(),
	}
	for _, sc := range pc.StageList {
		stage := &api.Stage{
			Name:          sc.Name,
			CreatorID:     sc.CreatorID,
			CreatedTs:     time.Now().Unix(),
			UpdaterID:     sc.CreatorID,
			UpdatedTs:     time.Now().Unix(),
			PipelineID:    sc.PipelineID,
			EnvironmentID: sc.EnvironmentID,
			// 	TaskList      []*Task
		}
		for _, tc := range sc.TaskList {
			task := &api.Task{
				Name:              tc.Name,
				Status:            tc.Status,
				CreatorID:         tc.CreatorID,
				CreatedTs:         time.Now().Unix(),
				UpdaterID:         tc.CreatorID,
				UpdatedTs:         time.Now().Unix(),
				Type:              tc.Type,
				Payload:           tc.Payload,
				EarliestAllowedTs: tc.EarliestAllowedTs,
				PipelineID:        pipeline.ID,
				StageID:           stage.ID,
				InstanceID:        tc.InstanceID,
				DatabaseID:        tc.DatabaseID,
			}
			stage.TaskList = append(stage.TaskList, task)
		}
		pipeline.StageList = append(pipeline.StageList, stage)
	}
	return pipeline, nil
}

func (s *Server) getPipelineFromIssue(ctx context.Context, issueCreate *api.IssueCreate, creatorID int) (*api.PipelineCreate, error) {
	switch issueCreate.Type {
	case api.IssueDatabaseCreate:
		m := api.CreateDatabaseContext{}
		if err := json.Unmarshal([]byte(issueCreate.CreateContext), &m); err != nil {
			return nil, err
		}
		if m.DatabaseName == "" {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, database name missing")
		}

		// Find instance.
		instance, err := s.composeInstanceByID(ctx, m.InstanceID)
		if err != nil {
			return nil, err
		}

		switch instance.Engine {
		case db.ClickHouse:
			// ClickHouse does not support character set and collation at the database level.
			if m.CharacterSet != "" {
				return nil, echo.NewHTTPError(
					http.StatusBadRequest,
					fmt.Sprintf("Failed to create issue, ClickHouse does not support character set, got %s\n", m.CharacterSet),
				)
			}
			if m.Collation != "" {
				return nil, echo.NewHTTPError(
					http.StatusBadRequest,
					fmt.Sprintf("Failed to create issue, ClickHouse does not support collation, got %s\n", m.Collation),
				)
			}
		case db.Snowflake:
			if m.CharacterSet != "" {
				return nil, echo.NewHTTPError(
					http.StatusBadRequest,
					fmt.Sprintf("Failed to create issue, Snowflake does not support character set, got %s\n", m.CharacterSet),
				)
			}
			if m.Collation != "" {
				return nil, echo.NewHTTPError(
					http.StatusBadRequest,
					fmt.Sprintf("Failed to create issue, Snowflake does not support collation, got %s\n", m.Collation),
				)
			}

			// Snowflake needs to use upper case of DatabaseName.
			m.DatabaseName = strings.ToUpper(m.DatabaseName)
		default:
			if m.CharacterSet == "" {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, character set missing")
			}
			// For postgres, we don't explicitly specify a default since the default might be UNSET (denoted by "C").
			// If that's the case, setting an explicit default such as "en_US.UTF-8" might fail if the instance doesn't
			// install it.
			if instance.Engine != db.Postgres && m.Collation == "" {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, collation missing")
			}
		}

		payload := api.TaskDatabaseCreatePayload{}
		payload.ProjectID = issueCreate.ProjectID
		payload.CharacterSet = m.CharacterSet
		payload.Collation = m.Collation
		payload.DatabaseName, payload.Statement = getDatabaseNameAndStatement(instance.Engine, m.DatabaseName, m.CharacterSet, m.Collation)
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to create database creation task, unable to marshal payload %w", err)
		}

		if m.BackupID != 0 || m.BackupName != "" {
			restorePayload := api.TaskDatabaseRestorePayload{}
			restorePayload.DatabaseName = m.DatabaseName
			restorePayload.BackupID = m.BackupID
			restoreBytes, err := json.Marshal(restorePayload)
			if err != nil {
				return nil, fmt.Errorf("failed to create restore database task, unable to marshal payload %w", err)
			}

			return &api.PipelineCreate{
				Name:      fmt.Sprintf("Pipeline - Create database %v from backup %v", payload.DatabaseName, m.BackupName),
				CreatorID: creatorID,
				StageList: []api.StageCreate{
					{
						Name:          "Create database",
						EnvironmentID: instance.EnvironmentID,
						TaskList: []api.TaskCreate{
							{
								InstanceID:   m.InstanceID,
								Name:         fmt.Sprintf("Create database %v", payload.DatabaseName),
								Status:       api.TaskPendingApproval,
								Type:         api.TaskDatabaseCreate,
								DatabaseName: payload.DatabaseName,
								Payload:      string(bytes),
							},
						},
					},
					{
						Name:          "Restore backup",
						EnvironmentID: instance.EnvironmentID,
						TaskList: []api.TaskCreate{
							{
								InstanceID:   m.InstanceID,
								Name:         fmt.Sprintf("Restore backup %v", m.BackupName),
								Status:       api.TaskPending,
								Type:         api.TaskDatabaseRestore,
								DatabaseName: payload.DatabaseName,
								BackupID:     &m.BackupID,
								Payload:      string(restoreBytes),
							},
						},
					},
				},
			}, nil
		}
		return &api.PipelineCreate{
			Name:      fmt.Sprintf("Pipeline - Create database %v", payload.DatabaseName),
			CreatorID: creatorID,
			StageList: []api.StageCreate{
				{
					Name:          "Create database",
					EnvironmentID: instance.EnvironmentID,
					TaskList: []api.TaskCreate{
						{
							InstanceID:   m.InstanceID,
							Name:         fmt.Sprintf("Create database %v", payload.DatabaseName),
							Status:       api.TaskPendingApproval,
							Type:         api.TaskDatabaseCreate,
							DatabaseName: payload.DatabaseName,
							Payload:      string(bytes),
						},
					},
				},
			},
		}, nil
	case api.IssueDatabaseSchemaUpdate:
		m := api.UpdateSchemaContext{}
		if err := json.Unmarshal([]byte(issueCreate.CreateContext), &m); err != nil {
			return nil, err
		}
		pc := &api.PipelineCreate{
			CreatorID: creatorID,
		}
		switch m.MigrationType {
		case db.Baseline:
			pc.Name = "Establish database baseline pipeline"
		case db.Migrate:
			pc.Name = "Update database schema pipeline"
		default:
			return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid migration type %q", m.MigrationType))
		}

		for _, d := range m.UpdateSchemaDetailList {
			if m.MigrationType == db.Migrate && d.Statement == "" {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, sql statement missing")
			}
			databaseFind := &api.DatabaseFind{
				ID: &d.DatabaseID,
			}
			database, err := s.composeDatabaseByFind(ctx, databaseFind)
			if err != nil {
				if common.ErrorCode(err) == common.NotFound {
					return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", d.DatabaseID))
				}
				return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", d.DatabaseID)).SetInternal(err)
			}

			taskName := fmt.Sprintf("Establish %q baseline", database.Name)
			if m.MigrationType == db.Migrate {
				taskName = fmt.Sprintf("Update %q schema", database.Name)
			}
			payload := api.TaskDatabaseSchemaUpdatePayload{}
			payload.MigrationType = m.MigrationType
			payload.Statement = d.Statement
			if d.RollbackStatement != "" {
				payload.RollbackStatement = d.RollbackStatement
			}
			if m.VCSPushEvent != nil {
				payload.VCSPushEvent = m.VCSPushEvent
			}
			bytes, err := json.Marshal(payload)
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal database schema update payload: %v", err))
			}

			pc.StageList = append(pc.StageList, api.StageCreate{
				Name:          fmt.Sprintf("%s %s", database.Instance.Environment.Name, database.Name),
				EnvironmentID: database.Instance.Environment.ID,
				TaskList: []api.TaskCreate{
					{
						Name:              taskName,
						InstanceID:        database.Instance.ID,
						DatabaseID:        &database.ID,
						Status:            api.TaskPendingApproval,
						Type:              api.TaskDatabaseSchemaUpdate,
						Statement:         d.Statement,
						RollbackStatement: d.RollbackStatement,
						MigrationType:     m.MigrationType,
						Payload:           string(bytes),
					},
				},
			})
		}
		return pc, nil
	}
	return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid issue type %q", issueCreate.Type))
}

func getDatabaseNameAndStatement(dbType db.Type, databaseName, characterSet, collation string) (string, string) {
	// Snowflake needs to use upper case of DatabaseName.
	if dbType == db.Snowflake {
		databaseName = strings.ToUpper(databaseName)
	}

	var stmt string
	switch dbType {
	case db.MySQL, db.TiDB:
		stmt = fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET %s COLLATE %s", databaseName, characterSet, collation)
	case db.Postgres:
		if collation == "" {
			stmt = fmt.Sprintf("CREATE DATABASE \"%s\" ENCODING %q", databaseName, characterSet)
		} else {
			stmt = fmt.Sprintf("CREATE DATABASE \"%s\" ENCODING %q LC_COLLATE %q", databaseName, characterSet, collation)
		}
	case db.ClickHouse:
		stmt = fmt.Sprintf("CREATE DATABASE `%s`", databaseName)
	case db.Snowflake:
		databaseName = strings.ToUpper(databaseName)
		stmt = fmt.Sprintf("CREATE DATABASE %s", databaseName)
	}

	return databaseName, stmt
}

func (s *Server) changeIssueStatus(ctx context.Context, issue *api.Issue, newStatus api.IssueStatus, updaterID int, comment string) (*api.Issue, error) {
	var pipelineStatus api.PipelineStatus
	switch newStatus {
	case api.IssueOpen:
		pipelineStatus = api.PipelineOpen
	case api.IssueDone:
		// Returns error if any of the tasks is not DONE.
		for _, stage := range issue.Pipeline.StageList {
			for _, task := range stage.TaskList {
				if task.Status != api.TaskDone {
					return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("failed to resolve issue: %v, task %v has not finished", issue.Name, task.Name)}
				}
			}
		}
		pipelineStatus = api.PipelineDone
	case api.IssueCanceled:
		// If we want to cancel the issue, we find the current running tasks, mark each of them CANCELED.
		// We keep PENDING and FAILED tasks as is since the issue maybe reopened later, and it's better to
		// keep those tasks in the same state before the issue was canceled.
		for _, stage := range issue.Pipeline.StageList {
			for _, task := range stage.TaskList {
				if task.Status == api.TaskRunning {
					if _, err := s.changeTaskStatus(ctx, task, api.TaskCanceled, updaterID); err != nil {
						return nil, fmt.Errorf("failed to cancel issue: %v, failed to cancel task: %v, error: %w", issue.Name, task.Name, err)
					}
				}
			}
		}
		pipelineStatus = api.PipelineCanceled
	}

	pipelinePatch := &api.PipelinePatch{
		ID:        issue.PipelineID,
		UpdaterID: updaterID,
		Status:    &pipelineStatus,
	}
	if _, err := s.PipelineService.PatchPipeline(ctx, pipelinePatch); err != nil {
		return nil, fmt.Errorf("failed to update issue status: %v, failed to update pipeline status: %w", issue.Name, err)
	}

	issuePatch := &api.IssuePatch{
		ID:        issue.ID,
		UpdaterID: updaterID,
		Status:    &newStatus,
	}
	updatedIssue, err := s.IssueService.PatchIssue(ctx, issuePatch)
	if err != nil {
		if common.ErrorCode(err) == common.NotFound {
			return nil, fmt.Errorf("failed to update issue status: %v, error: %w", issue.Name, err)
		}
		return nil, fmt.Errorf("failed update issue status: %v, error: %w", issue.Name, err)
	}

	payload, err := json.Marshal(api.ActivityIssueStatusUpdatePayload{
		OldStatus: issue.Status,
		NewStatus: newStatus,
		IssueName: updatedIssue.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal activity after changing the issue status: %v, error: %w", issue.Name, err)
	}

	activityCreate := &api.ActivityCreate{
		CreatorID:   updaterID,
		ContainerID: issue.ID,
		Type:        api.ActivityIssueStatusUpdate,
		Level:       api.ActivityInfo,
		Comment:     comment,
		Payload:     string(payload),
	}

	_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{
		issue: updatedIssue,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create activity after changing the issue status: %v, error: %w", issue.Name, err)
	}

	return updatedIssue, nil
}

func (s *Server) postInboxIssueActivity(ctx context.Context, issue *api.Issue, activityID int) error {
	if issue.CreatorID != api.SystemBotID {
		inboxCreate := &api.InboxCreate{
			ReceiverID: issue.CreatorID,
			ActivityID: activityID,
		}
		_, err := s.InboxService.CreateInbox(ctx, inboxCreate)
		if err != nil {
			return fmt.Errorf("failed to post activity to creator inbox: %d, error: %w", issue.CreatorID, err)
		}
	}

	if issue.AssigneeID != api.SystemBotID && issue.AssigneeID != issue.CreatorID {
		inboxCreate := &api.InboxCreate{
			ReceiverID: issue.AssigneeID,
			ActivityID: activityID,
		}
		_, err := s.InboxService.CreateInbox(ctx, inboxCreate)
		if err != nil {
			return fmt.Errorf("failed to post activity to assignee inbox: %d, error: %w", issue.AssigneeID, err)
		}
	}

	for _, subscriberID := range issue.SubscriberIDList {
		if subscriberID != api.SystemBotID && subscriberID != issue.CreatorID && subscriberID != issue.AssigneeID {
			inboxCreate := &api.InboxCreate{
				ReceiverID: subscriberID,
				ActivityID: activityID,
			}
			_, err := s.InboxService.CreateInbox(ctx, inboxCreate)
			if err != nil {
				return fmt.Errorf("failed to post activity to subscriber inbox: %d, error: %w", subscriberID, err)
			}
		}
	}

	return nil
}
