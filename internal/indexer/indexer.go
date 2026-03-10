package indexer

import "fmt"

type Indexer struct {
	repoPath string
	manifest *Manifest
}

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
