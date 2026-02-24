package app

import (
	"context"
	"fmt"
	"path"
)

type objectStorePlanMode string

const (
	objectStorePlanModeRoot    objectStorePlanMode = "root"
	objectStorePlanModeObjects objectStorePlanMode = "objects"
)

type objectStorePlannerConfig struct {
	mode         objectStorePlanMode
	rootKind     EntryKind
	scopeLabel   string
	scope        string
	prefix       string
	delimiter    string
	selected     []Entry
	destination  Location
	listByPrefix func(ctx context.Context, scope string, prefix string) ([]string, error)
	buildSource  func(scope string, objectPath string) (TransferObjectRef, error)
}

func buildObjectStoreCopyPlan(ctx context.Context, cfg objectStorePlannerConfig) ([]TransferPlanItem, error) {
	plan := make([]TransferPlanItem, 0, len(cfg.selected))
	for _, entry := range cfg.selected {
		switch cfg.mode {
		case objectStorePlanModeRoot:
			if entry.Kind != cfg.rootKind || entry.Name == "" {
				continue
			}
			keys, err := cfg.listByPrefix(ctx, entry.Name, "")
			if err != nil {
				return nil, err
			}
			for _, key := range keys {
				srcRef, err := cfg.buildSource(entry.Name, key)
				if err != nil {
					return nil, err
				}
				dstRef, err := resolveDestinationRef(cfg.destination, path.Join(entry.Name, key))
				if err != nil {
					return nil, err
				}
				plan = append(plan, TransferPlanItem{Source: srcRef, Destination: dstRef})
			}
		case objectStorePlanModeObjects:
			if cfg.scope == "" {
				return nil, fmt.Errorf("%s not selected", cfg.scopeLabel)
			}
			switch entry.Kind {
			case KindObject:
				objectPath := entry.FullPath
				if objectPath == "" {
					objectPath = joinObjectPath(cfg.prefix, entry.Name)
				}
				srcRef, err := cfg.buildSource(cfg.scope, objectPath)
				if err != nil {
					return nil, err
				}
				dstRef, err := resolveDestinationRef(cfg.destination, entry.Name)
				if err != nil {
					return nil, err
				}
				plan = append(plan, TransferPlanItem{Source: srcRef, Destination: dstRef})
			case KindDirectory:
				dirPath := entry.FullPath
				if dirPath == "" {
					dirPath = joinObjectPath(cfg.prefix, entry.Name)
				}
				queryPrefix := enterPrefix(dirPath, cfg.delimiter)
				keys, err := cfg.listByPrefix(ctx, cfg.scope, queryPrefix)
				if err != nil {
					return nil, err
				}
				for _, key := range keys {
					rel := path.Join(entry.Name, trimPrefix(key, queryPrefix))
					srcRef, err := cfg.buildSource(cfg.scope, key)
					if err != nil {
						return nil, err
					}
					dstRef, err := resolveDestinationRef(cfg.destination, rel)
					if err != nil {
						return nil, err
					}
					plan = append(plan, TransferPlanItem{Source: srcRef, Destination: dstRef})
				}
			}
		default:
			return nil, fmt.Errorf("unknown object store plan mode: %s", cfg.mode)
		}
	}

	return plan, nil
}
