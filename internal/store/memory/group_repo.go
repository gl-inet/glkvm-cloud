package memory

import (
	"context"
	"sync"

	"rttys/internal/domain/devicegroup"
	"rttys/internal/domain/group"
)

type GroupRepo struct {
	mu sync.RWMutex

	// user_groups
	userGroups map[int64]group.UserGroup

	// device_groups
	deviceGroups map[int64]devicegroup.DeviceGroup

	// user_group_members: user_id -> []user_group_id
	userToUserGroups map[int64][]int64

	// user_group_device_group_links: user_group_id -> []device_group_id
	userGroupToDeviceGroups map[int64][]int64
}

func NewGroupRepo() *GroupRepo {
	return &GroupRepo{
		userGroups:             map[int64]group.UserGroup{},
		deviceGroups:           map[int64]devicegroup.DeviceGroup{},
		userToUserGroups:       map[int64][]int64{},
		userGroupToDeviceGroups: map[int64][]int64{},
	}
}

func (r *GroupRepo) SeedUserGroups(items []group.UserGroup) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, it := range items {
		r.userGroups[it.ID] = it
	}
}

func (r *GroupRepo) SeedDeviceGroups(items []devicegroup.DeviceGroup) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, it := range items {
		r.deviceGroups[it.ID] = it
	}
}

func (r *GroupRepo) SeedUserMembership(userID int64, userGroupIDs []int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.userToUserGroups[userID] = append([]int64(nil), userGroupIDs...)
}

func (r *GroupRepo) SeedUserGroupDeviceGroupLinks(userGroupID int64, deviceGroupIDs []int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.userGroupToDeviceGroups[userGroupID] = append([]int64(nil), deviceGroupIDs...)
}

// ListUserGroupsVisibleToUser:
// - admin: all user groups
// - user : only groups the user belongs to
func (r *GroupRepo) ListUserGroupsVisibleToUser(ctx context.Context, userID int64, isAdmin bool) ([]group.UserGroup, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if isAdmin {
		out := make([]group.UserGroup, 0, len(r.userGroups))
		for _, g := range r.userGroups {
			out = append(out, g)
		}
		return out, nil
	}

	ugIDs := r.userToUserGroups[userID]
	out := make([]group.UserGroup, 0, len(ugIDs))
	for _, id := range ugIDs {
		if g, ok := r.userGroups[id]; ok {
			out = append(out, g)
		}
	}
	return out, nil
}

// ListDeviceGroupIDsByUser returns device_group IDs the user can see via the scope join.
func (r *GroupRepo) ListDeviceGroupIDsByUser(ctx context.Context, userID int64) ([]int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ugIDs := r.userToUserGroups[userID]
	seen := map[int64]struct{}{}
	var out []int64
	for _, ug := range ugIDs {
		for _, dg := range r.userGroupToDeviceGroups[ug] {
			if _, ok := seen[dg]; ok {
				continue
			}
			seen[dg] = struct{}{}
			out = append(out, dg)
		}
	}
	return out, nil
}

// ListDeviceGroupsVisibleToUser:
// - admin: all device groups
// - user : distinct device groups linked from user's user groups
func (r *GroupRepo) ListDeviceGroupsVisibleToUser(ctx context.Context, userID int64, isAdmin bool) ([]devicegroup.DeviceGroup, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if isAdmin {
		out := make([]devicegroup.DeviceGroup, 0, len(r.deviceGroups))
		for _, dg := range r.deviceGroups {
			out = append(out, dg)
		}
		return out, nil
	}

	ugIDs := r.userToUserGroups[userID]
	seen := map[int64]struct{}{}
	var out []devicegroup.DeviceGroup
	for _, ug := range ugIDs {
		for _, dgID := range r.userGroupToDeviceGroups[ug] {
			if _, ok := seen[dgID]; ok {
				continue
			}
			seen[dgID] = struct{}{}
			if dg, ok := r.deviceGroups[dgID]; ok {
				out = append(out, dg)
			}
		}
	}
	return out, nil
}
