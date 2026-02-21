package bootstrap

import (
	"context"
	"database/sql"
	"fmt"

	"pressluft/internal/jobs"
	"pressluft/internal/nodes"
	"pressluft/internal/store"
)

type Result struct {
	NodeID            string
	NodeCreated       bool
	NodeProvisionJob  bool
	NodeCurrentStatus string
}

func Run(ctx context.Context, db *sql.DB, hostname string) (Result, error) {
	nodeRepo := nodes.NewRepository(db)

	node, created, err := nodeRepo.EnsureLocalProvisioningNode(ctx, hostname)
	if err != nil {
		return Result{}, fmt.Errorf("ensure local node: %w", err)
	}

	queued := false
	err = store.WithTx(ctx, db, func(tx *sql.Tx) error {
		jobCreated, err := jobs.EnsureNodeProvisionQueued(ctx, tx, node.ID)
		if err != nil {
			return err
		}
		queued = jobCreated
		return nil
	})
	if err != nil {
		return Result{}, fmt.Errorf("ensure node provision job: %w", err)
	}

	return Result{
		NodeID:            node.ID,
		NodeCreated:       created,
		NodeProvisionJob:  queued,
		NodeCurrentStatus: node.Status,
	}, nil
}
