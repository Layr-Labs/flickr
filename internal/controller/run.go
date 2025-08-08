package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/yourorg/flickr/internal/docker"
	"github.com/yourorg/flickr/internal/eth"
	"github.com/yourorg/flickr/internal/ref"
)

type Controller struct {
	RM     eth.ReleaseManagerClient
	Docker docker.Docker
}

type RunConfig struct {
	AVS            common.Address
	OperatorSetID  uint32
	ReleaseID      *uint64
	ReleaseManager common.Address
	RPCURL         string
	Name           string
	Detached       bool
	Env            map[string]string
	Cmd            []string
}

func New(rm eth.ReleaseManagerClient, dockerRunner docker.Docker) *Controller {
	return &Controller{
		RM:     rm,
		Docker: dockerRunner,
	}
}

func (c *Controller) Execute(ctx context.Context, cfg RunConfig) error {
	// 1) Fetch release
	var (
		rel   eth.Release
		relID uint64
		err   error
	)
	
	if cfg.ReleaseID == nil {
		rel, relID, err = c.RM.GetLatestRelease(ctx, cfg.AVS, cfg.OperatorSetID)
		if err != nil {
			// Provide better error message for common issues
			errMsg := err.Error()
			if strings.Contains(errMsg, "arithmetic underflow") || strings.Contains(errMsg, "NoReleases") {
				return fmt.Errorf(`no releases available for this operator set

To push a release, run:
  flickr push --image <your-image>

Current configuration:
  AVS: %s
  Operator Set: %d`, cfg.AVS.Hex(), cfg.OperatorSetID)
			}
			if strings.Contains(errMsg, "array out-of-bounds") {
				return fmt.Errorf(`no releases found (the operator set may not have any releases yet)

To push a release, run:
  flickr push --image <your-image>`)
			}
			return fmt.Errorf("failed to get latest release: %w", err)
		}
	} else {
		rel, err = c.RM.GetRelease(ctx, cfg.AVS, cfg.OperatorSetID, *cfg.ReleaseID)
		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "array out-of-bounds") {
				return fmt.Errorf("release ID %d does not exist", *cfg.ReleaseID)
			}
			return fmt.Errorf("failed to get release %d: %w", *cfg.ReleaseID, err)
		}
		relID = *cfg.ReleaseID
	}
	
	// Validate release has artifacts
	if len(rel.Artifacts) == 0 {
		return fmt.Errorf("no artifacts in release")
	}
	
	// Take first artifact only (MVP)
	art := rel.Artifacts[0]
	
	// Convert digest to string format
	digest := ref.Digest32ToSha256String(art.Digest32)
	
	// Build pullable reference
	reference, err := ref.BuildReference(art.Registry, digest)
	if err != nil {
		return fmt.Errorf("failed to build reference: %w", err)
	}
	
	// 2) Docker pull
	if err := c.Docker.Pull(ctx, reference); err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	
	// 3) Docker run with AVS context
	env := map[string]string{
		"AVS_ADDRESS":     cfg.AVS.Hex(),
		"OPERATOR_SET_ID": fmt.Sprintf("%d", cfg.OperatorSetID),
		"RELEASE_ID":      fmt.Sprintf("%d", relID),
		"UPGRADE_BY_TIME": fmt.Sprintf("%d", rel.UpgradeByTime),
	}
	
	// Merge user-provided env vars
	for k, v := range cfg.Env {
		env[k] = v
	}
	
	runOpts := docker.RunOptions{
		Name:     cfg.Name,
		Detached: cfg.Detached,
		Env:      env,
		Cmd:      cfg.Cmd,
	}
	
	if err := c.Docker.Run(ctx, reference, runOpts); err != nil {
		return fmt.Errorf("failed to run container: %w", err)
	}
	
	return nil
}