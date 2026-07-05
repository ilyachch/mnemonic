package kb

// KnowledgeBase describes one selected knowledge base at runtime.
type KnowledgeBase struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Slug               string `json:"slug"`
	Kind               string `json:"kind"`
	Description        string `json:"description,omitempty"`
	CustomInstructions string `json:"custom_instructions,omitempty"`
	LinksStyle         string `json:"links_style,omitempty"`
	RootDir            string `json:"root_dir"`
	RepoRootDir        string `json:"repo_root_dir,omitempty"`
	ManifestPath       string `json:"manifest_path,omitempty"`
	StateDir           string `json:"state_dir"`
	IndexPath          string `json:"index_path"`
}
