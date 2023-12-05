package database

import (
	"articulate/internal/types"
	"sync"
)

type MemoryDB struct {
	store *promptStore
}

type promptStore struct {
	mu      sync.RWMutex
	Prompts *types.PromptIntents `mapstructure:"prompts"`
}

func newPromptStore() *promptStore {
	return &promptStore{
		mu:      sync.RWMutex{},
		Prompts: &types.PromptIntents{},
	}
}

func NewPromptStore() *MemoryDB {
	return &MemoryDB{
		store: newPromptStore(),
	}
}

func (db *MemoryDB) GetPromptIntent(promptId string) (types.PromptIntent, bool) {
	db.store.mu.RLock()
	defer db.store.mu.RUnlock()

	prompts := db.store.Prompts

	for _, prompt := range *prompts {
		if *prompt.Id == promptId {
			return *prompt.Copy(), true
		}
	}

	return types.PromptIntent{}, false
}

func (db *MemoryDB) GetPromptIntents() types.PromptIntents {
	db.store.mu.RLock()
	defer db.store.mu.RUnlock()

    prompts := db.store.Prompts
    if prompts == nil {
        return types.PromptIntents{}
    }

    return *prompts.Copy()
}

func (db *MemoryDB) SetPromptIntent(prompt types.PromptIntent) error {
	db.store.mu.Lock()
	defer db.store.mu.Unlock()

	prompts := db.store.Prompts

	*prompts = append(*prompts, &prompt)
	return nil
}
