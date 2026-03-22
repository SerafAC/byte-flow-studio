// Package processing provides processing block implementations and the shared block registry.
package processing

import "byteflow-studio/internal/pipeline"

// BlockFactory creates a new Block instance with a given ID.
type BlockFactory func(id string) pipeline.Block

// Registry holds all registered block factories.
// Registration happens in init() functions in each block's package.
var Registry = map[string]BlockFactory{}

// Register adds a factory to the registry. Called from package init().
func Register(blockType string, factory BlockFactory) {
	Registry[blockType] = factory
}
