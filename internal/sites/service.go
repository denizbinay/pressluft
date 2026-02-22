package sites

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

const defaultJobType = "site_create"

var ErrMutationConflict = errors.New("site mutation conflict")
var ErrNoTargetNode = errors.New("no target node available")
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
	List(ctx context.Context) ([]nodes.Node, error)
}

type ReadinessChecker interface {
	Evaluate(ctx context.Context, node nodes.Node) (nodes.ReadinessReport, error)
}

type Service struct {
	store     store.SiteStore
	nodeStore NodeStore
	queue     JobQueue
	readiness ReadinessChecker
	now       func() time.Time
	jobType   string
}

func NewService(siteStore store.SiteStore, nodeStore NodeStore, queue JobQueue, readiness ReadinessChecker) *Service {
	return &Service{
		store:     siteStore,
		nodeStore: nodeStore,
		queue:     queue,
		readiness: readiness,
		now:       func() time.Time { return time.Now().UTC() },
		jobType:   defaultJobType,
	}
}

func (s *Service) List(ctx context.Context) ([]store.Site, error) {
	items, err := s.store.ListSites(ctx)
	if err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}
	return items, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (store.Site, error) {
	site, err := s.store.GetSiteByID(ctx, id)
	if err != nil {
		return store.Site{}, fmt.Errorf("get site by id: %w", err)
	}
	return site, nil
}

func (s *Service) Create(ctx context.Context, name string, slug string) (string, error) {
	now := s.now()

	nodesList, err := s.nodeStore.List(ctx)
	if err != nil {
		return "", fmt.Errorf("list nodes: %w", err)
	}

	providerNodes := make([]nodes.Node, 0, len(nodesList))
	for _, node := range nodesList {
		if node.IsLocal {
			continue
		}
		providerNodes = append(providerNodes, node)
	}
	if len(providerNodes) == 0 {
		return "", &NodeNotReadyError{Message: "target node is not ready: node_unreachable. Create a provider-backed node from /nodes first.", ReasonCodes: []string{nodes.ReasonNodeUnreachable}}
	}

	targetNode := providerNodes[0]
	if s.readiness != nil {
		var firstNotReady *NodeNotReadyError
		for _, candidate := range providerNodes {
			report, err := s.readiness.Evaluate(ctx, candidate)
			if err != nil {
				return "", fmt.Errorf("evaluate node readiness: %w", err)
			}
			if report.IsReady {
				targetNode = candidate
				firstNotReady = nil
				break
			}
			if firstNotReady == nil {
				message := "target node is not ready: " + strings.Join(report.ReasonCodes, ", ")
				if len(report.Guidance) > 0 {
					message += ". " + strings.Join(report.Guidance, " ")
				}
				firstNotReady = &NodeNotReadyError{Message: message, ReasonCodes: report.ReasonCodes}
			}
		}
		if firstNotReady != nil {
			return "", firstNotReady
		}
	}

	site, _, err := s.store.CreateSiteWithProductionEnvironment(ctx, store.CreateSiteInput{
		Name:       name,
		Slug:       slug,
		NodeID:     targetNode.ID,
		NodePublic: targetNode.PublicIP,
		Now:        now,
	})
	if err != nil {
		return "", fmt.Errorf("create site and environment: %w", err)
	}

	nodeID := targetNode.ID
	job, err := s.queue.Enqueue(ctx, jobs.EnqueueInput{
		JobType:       s.jobType,
		SiteID:        &site.ID,
		EnvironmentID: site.PrimaryEnvironmentID,
		NodeID:        &nodeID,
		MaxAttempts:   3,
		CreatedAt:     now,
	})
	if err != nil {
		if errors.Is(err, jobs.ErrConflict) {
			return "", ErrMutationConflict
		}
		return "", fmt.Errorf("enqueue site create job: %w", err)
	}

	return job.ID, nil
}
