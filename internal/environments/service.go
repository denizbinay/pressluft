package environments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/jobs"
	"pressluft/internal/nodes"
	"pressluft/internal/store"
)

const defaultJobType = "env_create"

var ErrMutationConflict = errors.New("environment mutation conflict")
var ErrNodeNotReady = errors.New("target node not ready")

type NodeNotReadyError struct {
	Message     string
	ReasonCodes []string
}

func (e *NodeNotReadyError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return "target node not ready"
}

func (e *NodeNotReadyError) Is(target error) bool {
	return target == ErrNodeNotReady
}

type JobQueue interface {
	Enqueue(ctx context.Context, input jobs.EnqueueInput) (jobs.Job, error)
}

type NodeStore interface {
	GetByID(ctx context.Context, id string) (nodes.Node, error)
}

type ReadinessChecker interface {
	Evaluate(ctx context.Context, node nodes.Node) (nodes.ReadinessReport, error)
}

type Service struct {
	store     store.SiteStore
	queue     JobQueue
	nodeStore NodeStore
	readiness ReadinessChecker
	now       func() time.Time
	jobType   string
}

type CreateInput struct {
	SiteID              string
	Name                string
	Slug                string
	EnvironmentType     string
	SourceEnvironmentID *string
	PromotionPreset     string
}

func NewService(siteStore store.SiteStore, queue JobQueue, nodeStore NodeStore, readiness ReadinessChecker) *Service {
	return &Service{
		store:     siteStore,
		queue:     queue,
		nodeStore: nodeStore,
		readiness: readiness,
		now:       func() time.Time { return time.Now().UTC() },
		jobType:   defaultJobType,
	}
}

func (s *Service) ListBySiteID(ctx context.Context, siteID string) ([]store.Environment, error) {
	items, err := s.store.ListEnvironmentsBySiteID(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("list site environments: %w", err)
	}
	return items, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (store.Environment, error) {
	item, err := s.store.GetEnvironmentByID(ctx, id)
	if err != nil {
		return store.Environment{}, fmt.Errorf("get environment by id: %w", err)
	}
	return item, nil
}

func (s *Service) Create(ctx context.Context, input CreateInput) (string, error) {
	now := s.now()
	if s.readiness != nil && s.nodeStore != nil && input.SourceEnvironmentID != nil && strings.TrimSpace(*input.SourceEnvironmentID) != "" {
		sourceEnvironmentID := strings.TrimSpace(*input.SourceEnvironmentID)
		sourceEnvironment, err := s.store.GetEnvironmentByID(ctx, sourceEnvironmentID)
		if err != nil {
			if errors.Is(err, store.ErrEnvironmentNotFound) {
				return "", fmt.Errorf("create environment: %w", store.ErrInvalidEnvironmentCreate)
			}
			return "", fmt.Errorf("load source environment for readiness: %w", err)
		}
		if sourceEnvironment.SiteID != input.SiteID {
			return "", fmt.Errorf("create environment: %w", store.ErrInvalidEnvironmentCreate)
		}
		node, err := s.nodeStore.GetByID(ctx, sourceEnvironment.NodeID)
		if err != nil {
			if errors.Is(err, nodes.ErrNotFound) {
				return "", &NodeNotReadyError{Message: "target node is not ready: node_unreachable", ReasonCodes: []string{nodes.ReasonNodeUnreachable}}
			}
			return "", fmt.Errorf("load node for readiness: %w", err)
		}
		report, err := s.readiness.Evaluate(ctx, node)
		if err != nil {
			return "", fmt.Errorf("evaluate node readiness: %w", err)
		}
		if !report.IsReady {
			message := "target node is not ready: " + strings.Join(report.ReasonCodes, ", ")
			if len(report.Guidance) > 0 {
				message += ". " + strings.Join(report.Guidance, " ")
			}
			return "", &NodeNotReadyError{Message: message, ReasonCodes: report.ReasonCodes}
		}
	}

	environment, err := s.store.CreateEnvironment(ctx, store.CreateEnvironmentInput{
		SiteID:              input.SiteID,
		Name:                input.Name,
		Slug:                input.Slug,
		EnvironmentType:     input.EnvironmentType,
		SourceEnvironmentID: input.SourceEnvironmentID,
		PromotionPreset:     input.PromotionPreset,
		Now:                 now,
	})
	if err != nil {
		return "", fmt.Errorf("create environment: %w", err)
	}

	job, err := s.queue.Enqueue(ctx, jobs.EnqueueInput{
		JobType:       s.jobType,
		SiteID:        &environment.SiteID,
		EnvironmentID: &environment.ID,
		NodeID:        &environment.NodeID,
		MaxAttempts:   3,
		CreatedAt:     now,
	})
	if err != nil {
		if errors.Is(err, jobs.ErrConflict) {
			return "", ErrMutationConflict
		}
		return "", fmt.Errorf("enqueue environment create job: %w", err)
	}

	return job.ID, nil
}
