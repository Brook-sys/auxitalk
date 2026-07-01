package actions

import (
	"errors"
	"sync"

	"github.com/Brook-sys/auxitalk/pkg/types"
)

var ErrActionNotFound = errors.New("action not found")

type Store struct {
	mu      sync.RWMutex
	actions map[string]types.ActionRequest
}

func NewStore() *Store {
	return &Store{actions: map[string]types.ActionRequest{}}
}

func (s *Store) Save(action types.ActionRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.actions[action.ID] = action
}

func (s *Store) Get(id string) (types.ActionRequest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	action, ok := s.actions[id]
	return action, ok
}

func (s *Store) UpdateStatus(id string, status types.ActionStatus) (types.ActionRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	action, ok := s.actions[id]
	if !ok {
		return types.ActionRequest{}, ErrActionNotFound
	}
	action.Status = status
	s.actions[id] = action
	return action, nil
}

func (s *Store) List() []types.ActionRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	actions := make([]types.ActionRequest, 0, len(s.actions))
	for _, action := range s.actions {
		actions = append(actions, action)
	}
	return actions
}

func (s *Store) Pending() []types.ActionRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	actions := make([]types.ActionRequest, 0, len(s.actions))
	for _, action := range s.actions {
		if action.Status == types.ActionStatusRequested || action.Status == types.ActionStatusConfirmed {
			actions = append(actions, action)
		}
	}
	return actions
}
