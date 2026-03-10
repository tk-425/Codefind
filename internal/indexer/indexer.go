package indexer

import (
	"fmt"
	"slices"

	"github.com/tk-425/Codefind/internal/lsp"
)

type Indexer struct {
	repoPath string
	manifest *Manifest
	lspState *lsp.WarmState
}

var warmLanguages = lsp.WarmLanguages

func New(repoPath string, manifest *Manifest) (*Indexer, error) {
	if err := validateRepoPath(repoPath); err != nil {
		return nil, fmt.Errorf("invalid repo path: %w", err)
	}
	if manifest == nil {
		manifest = &Manifest{
			SchemaVersion: ManifestSchemaVersion,
			Files:         make(map[string]ManifestFile),
		}
	}
	if manifest.SchemaVersion == 0 {
		manifest.SchemaVersion = ManifestSchemaVersion
	}
	if manifest.Files == nil {
		manifest.Files = make(map[string]ManifestFile)
	}
	return &Indexer{
		repoPath: repoPath,
		manifest: manifest,
	}, nil
}

func (i *Indexer) Discover() (*DiscoveryResult, error) {
	return DiscoverFiles(i.repoPath)
}

func (i *Indexer) DetectChanges() (*ChangeDetectionResult, error) {
	return DetectChanges(i.repoPath, i.manifest)
}

func (i *Indexer) Manifest() *Manifest {
	return i.manifest
}

func (i *Indexer) WarmLSPs() (*lsp.WarmState, error) {
	discovery, err := i.Discover()
	if err != nil {
		return nil, err
	}

	languages := make([]string, 0, len(discovery.Files))
	for _, file := range discovery.Files {
		if file.Language == "" || slices.Contains(languages, file.Language) {
			continue
		}
		languages = append(languages, file.Language)
	}

	state, err := warmLanguages(i.repoPath, languages)
	if err != nil {
		return nil, err
	}
	i.lspState = state
	return state, nil
}

func (i *Indexer) LSPState() *lsp.WarmState {
	return i.lspState
}
