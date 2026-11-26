package service

import "context"

// Delete removes the product logically if the version still matches.
func (s *Products) Delete(ctx context.Context, id string, version int32) error {
	return s.repo.Delete(ctx, id, version)
}
