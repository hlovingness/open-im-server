// Copyright © 2023 OpenIM. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import (
	"context"
	"time"

	"github.com/OpenIMSDK/tools/log"

	"github.com/dtm-labs/rockscache"
	"github.com/redis/go-redis/v9"

	"github.com/OpenIMSDK/tools/utils"

	relationtb "github.com/openimsdk/open-im-server/v3/pkg/common/db/table/relation"
)

const (
	groupExpireTime     = time.Second * 60 * 60 * 12
	groupInfoKey        = "GROUP_INFO:"
	groupMemberIDsKey   = "GROUP_MEMBER_IDS:"
	groupMembersHashKey = "GROUP_MEMBERS_HASH2:"
	groupMemberInfoKey  = "GROUP_MEMBER_INFO:"
	joinedGroupsKey     = "JOIN_GROUPS_KEY:"
	groupMemberNumKey   = "GROUP_MEMBER_NUM_CACHE:"
)

type GroupCache interface {
	metaCache
	NewCache() GroupCache
	GetGroupsInfo(ctx context.Context, groupIDs []string) (groups []*relationtb.GroupModel, err error)
	GetGroupInfo(ctx context.Context, groupID string) (group *relationtb.GroupModel, err error)
	DelGroupsInfo(groupIDs ...string) GroupCache

	GetGroupMembersHash(ctx context.Context, groupID string) (hashCode uint64, err error)
	GetGroupMemberHashMap(ctx context.Context, groupIDs []string) (map[string]*relationtb.GroupSimpleUserID, error)
	DelGroupMembersHash(groupID string) GroupCache

	GetGroupMemberIDs(ctx context.Context, groupID string) (groupMemberIDs []string, err error)
	GetGroupsMemberIDs(ctx context.Context, groupIDs []string) (groupMemberIDs map[string][]string, err error)

	DelGroupMemberIDs(groupID string) GroupCache

	GetJoinedGroupIDs(ctx context.Context, userID string) (joinedGroupIDs []string, err error)
	DelJoinedGroupID(userID ...string) GroupCache

	GetGroupMemberInfo(ctx context.Context, groupID, userID string) (groupMember *relationtb.GroupMemberModel, err error)
	GetGroupMembersInfo(ctx context.Context, groupID string, userID []string) (groupMembers []*relationtb.GroupMemberModel, err error)
	GetAllGroupMembersInfo(ctx context.Context, groupID string) (groupMembers []*relationtb.GroupMemberModel, err error)
	GetGroupMembersPage(ctx context.Context, groupID string, userID []string, showNumber, pageNumber int32) (total uint32, groupMembers []*relationtb.GroupMemberModel, err error)

	DelGroupMembersInfo(groupID string, userID ...string) GroupCache

	GetGroupMemberNum(ctx context.Context, groupID string) (memberNum int64, err error)
	DelGroupsMemberNum(groupID ...string) GroupCache
}

type GroupCacheRedis struct {
	metaCache
	groupDB        relationtb.GroupModelInterface
	groupMemberDB  relationtb.GroupMemberModelInterface
	groupRequestDB relationtb.GroupRequestModelInterface
	expireTime     time.Duration
	rcClient       *rockscache.Client
	hashCode       func(ctx context.Context, groupID string) (uint64, error)
}

func NewGroupCacheRedis(
	rdb redis.UniversalClient,
	groupDB relationtb.GroupModelInterface,
	groupMemberDB relationtb.GroupMemberModelInterface,
	groupRequestDB relationtb.GroupRequestModelInterface,
	hashCode func(ctx context.Context, groupID string) (uint64, error),
	opts rockscache.Options,
) GroupCache {
	rcClient := rockscache.NewClient(rdb, opts)

	return &GroupCacheRedis{
		rcClient: rcClient, expireTime: groupExpireTime,
		groupDB: groupDB, groupMemberDB: groupMemberDB, groupRequestDB: groupRequestDB,
		hashCode:  hashCode,
		metaCache: NewMetaCacheRedis(rcClient),
	}
}

func (g *GroupCacheRedis) NewCache() GroupCache {
	return &GroupCacheRedis{
		rcClient:       g.rcClient,
		expireTime:     g.expireTime,
		groupDB:        g.groupDB,
		groupMemberDB:  g.groupMemberDB,
		groupRequestDB: g.groupRequestDB,
		metaCache:      NewMetaCacheRedis(g.rcClient, g.metaCache.GetPreDelKeys()...),
	}
}

func (g *GroupCacheRedis) getGroupInfoKey(groupID string) string {
	return groupInfoKey + groupID
}

func (g *GroupCacheRedis) getJoinedGroupsKey(userID string) string {
	return joinedGroupsKey + userID
}

func (g *GroupCacheRedis) getGroupMembersHashKey(groupID string) string {
	return groupMembersHashKey + groupID
}

func (g *GroupCacheRedis) getGroupMemberIDsKey(groupID string) string {
	return groupMemberIDsKey + groupID
}

func (g *GroupCacheRedis) getGroupMemberInfoKey(groupID, userID string) string {
	return groupMemberInfoKey + groupID + "-" + userID
}

func (g *GroupCacheRedis) getGroupMemberNumKey(groupID string) string {
	return groupMemberNumKey + groupID
}

func (g *GroupCacheRedis) GetGroupIndex(group *relationtb.GroupModel, keys []string) (int, error) {
	key := g.getGroupInfoKey(group.GroupID)
	for i, _key := range keys {
		if _key == key {
			return i, nil
		}
	}

	return 0, errIndex
}

func (g *GroupCacheRedis) GetGroupMemberIndex(groupMember *relationtb.GroupMemberModel, keys []string) (int, error) {
	key := g.getGroupMemberInfoKey(groupMember.GroupID, groupMember.UserID)
	for i, _key := range keys {
		if _key == key {
			return i, nil
		}
	}

	return 0, errIndex
}

// / groupInfo.
func (g *GroupCacheRedis) GetGroupsInfo(ctx context.Context, groupIDs []string) (groups []*relationtb.GroupModel, err error) {
	return batchGetCache2(ctx, g.rcClient, g.expireTime, groupIDs, func(groupID string) string {
		return g.getGroupInfoKey(groupID)
	}, func(ctx context.Context, groupID string) (*relationtb.GroupModel, error) {
		return g.groupDB.Take(ctx, groupID)
	})
}

func (g *GroupCacheRedis) GetGroupInfo(ctx context.Context, groupID string) (group *relationtb.GroupModel, err error) {
	return getCache(ctx, g.rcClient, g.getGroupInfoKey(groupID), g.expireTime, func(ctx context.Context) (*relationtb.GroupModel, error) {
		return g.groupDB.Take(ctx, groupID)
	})
}

func (g *GroupCacheRedis) DelGroupsInfo(groupIDs ...string) GroupCache {
	newGroupCache := g.NewCache()
	keys := make([]string, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		keys = append(keys, g.getGroupInfoKey(groupID))
	}
	newGroupCache.AddKeys(keys...)

	return newGroupCache
}

// groupMembersHash.
func (g *GroupCacheRedis) GetGroupMembersHash(ctx context.Context, groupID string) (hashCode uint64, err error) {
	return getCache(ctx, g.rcClient, g.getGroupMembersHashKey(groupID), g.expireTime, func(ctx context.Context) (uint64, error) {
		return g.hashCode(ctx, groupID)
	})
}

func (g *GroupCacheRedis) GetGroupMemberHashMap(ctx context.Context, groupIDs []string) (map[string]*relationtb.GroupSimpleUserID, error) {
	res := make(map[string]*relationtb.GroupSimpleUserID)
	for _, groupID := range groupIDs {
		hash, err := g.GetGroupMembersHash(ctx, groupID)
		if err != nil {
			return nil, err
		}
		log.ZInfo(ctx, "GetGroupMemberHashMap", "groupID", groupID, "hash", hash)
		num, err := g.GetGroupMemberNum(ctx, groupID)
		if err != nil {
			return nil, err
		}
		res[groupID] = &relationtb.GroupSimpleUserID{Hash: hash, MemberNum: uint32(num)}
	}

	return res, nil
}

func (g *GroupCacheRedis) DelGroupMembersHash(groupID string) GroupCache {
	cache := g.NewCache()
	cache.AddKeys(g.getGroupMembersHashKey(groupID))

	return cache
}

// groupMemberIDs.
func (g *GroupCacheRedis) GetGroupMemberIDs(ctx context.Context, groupID string) (groupMemberIDs []string, err error) {
	return getCache(ctx, g.rcClient, g.getGroupMemberIDsKey(groupID), g.expireTime, func(ctx context.Context) ([]string, error) {
		return g.groupMemberDB.FindMemberUserID(ctx, groupID)
	})
}

func (g *GroupCacheRedis) GetGroupsMemberIDs(ctx context.Context, groupIDs []string) (map[string][]string, error) {
	m := make(map[string][]string)
	for _, groupID := range groupIDs {
		userIDs, err := g.GetGroupMemberIDs(ctx, groupID)
		if err != nil {
			return nil, err
		}
		m[groupID] = userIDs
	}

	return m, nil
}

func (g *GroupCacheRedis) DelGroupMemberIDs(groupID string) GroupCache {
	cache := g.NewCache()
	cache.AddKeys(g.getGroupMemberIDsKey(groupID))

	return cache
}

func (g *GroupCacheRedis) GetJoinedGroupIDs(ctx context.Context, userID string) (joinedGroupIDs []string, err error) {
	return getCache(ctx, g.rcClient, g.getJoinedGroupsKey(userID), g.expireTime, func(ctx context.Context) ([]string, error) {
		return g.groupMemberDB.FindUserJoinedGroupID(ctx, userID)
	})
}

func (g *GroupCacheRedis) DelJoinedGroupID(userIDs ...string) GroupCache {
	keys := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		keys = append(keys, g.getJoinedGroupsKey(userID))
	}
	cache := g.NewCache()
	cache.AddKeys(keys...)

	return cache
}

func (g *GroupCacheRedis) GetGroupMemberInfo(ctx context.Context, groupID, userID string) (groupMember *relationtb.GroupMemberModel, err error) {
	return getCache(ctx, g.rcClient, g.getGroupMemberInfoKey(groupID, userID), g.expireTime, func(ctx context.Context) (*relationtb.GroupMemberModel, error) {
		return g.groupMemberDB.Take(ctx, groupID, userID)
	})
}

func (g *GroupCacheRedis) GetGroupMembersInfo(ctx context.Context, groupID string, userIDs []string) ([]*relationtb.GroupMemberModel, error) {
	return batchGetCache2(ctx, g.rcClient, g.expireTime, userIDs, func(userID string) string {
		return g.getGroupMemberInfoKey(groupID, userID)
	}, func(ctx context.Context, userID string) (*relationtb.GroupMemberModel, error) {
		return g.groupMemberDB.Take(ctx, groupID, userID)
	})
}

func (g *GroupCacheRedis) GetGroupMembersPage(
	ctx context.Context,
	groupID string,
	userIDs []string,
	showNumber, pageNumber int32,
) (total uint32, groupMembers []*relationtb.GroupMemberModel, err error) {
	groupMemberIDs, err := g.GetGroupMemberIDs(ctx, groupID)
	if err != nil {
		return 0, nil, err
	}
	if userIDs != nil {
		userIDs = utils.BothExist(userIDs, groupMemberIDs)
	} else {
		userIDs = groupMemberIDs
	}
	groupMembers, err = g.GetGroupMembersInfo(ctx, groupID, utils.Paginate(userIDs, int(showNumber), int(showNumber)))

	return uint32(len(userIDs)), groupMembers, err
}

func (g *GroupCacheRedis) GetAllGroupMembersInfo(ctx context.Context, groupID string) (groupMembers []*relationtb.GroupMemberModel, err error) {
	groupMemberIDs, err := g.GetGroupMemberIDs(ctx, groupID)
	if err != nil {
		return nil, err
	}

	return g.GetGroupMembersInfo(ctx, groupID, groupMemberIDs)
}

func (g *GroupCacheRedis) GetAllGroupMemberInfo(ctx context.Context, groupID string) ([]*relationtb.GroupMemberModel, error) {
	groupMemberIDs, err := g.GetGroupMemberIDs(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return g.GetGroupMembersInfo(ctx, groupID, groupMemberIDs)
}

func (g *GroupCacheRedis) DelGroupMembersInfo(groupID string, userIDs ...string) GroupCache {
	keys := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		keys = append(keys, g.getGroupMemberInfoKey(groupID, userID))
	}
	cache := g.NewCache()
	cache.AddKeys(keys...)

	return cache
}

func (g *GroupCacheRedis) GetGroupMemberNum(ctx context.Context, groupID string) (memberNum int64, err error) {
	return getCache(ctx, g.rcClient, g.getGroupMemberNumKey(groupID), g.expireTime, func(ctx context.Context) (int64, error) {
		return g.groupMemberDB.TakeGroupMemberNum(ctx, groupID)
	})
}

func (g *GroupCacheRedis) DelGroupsMemberNum(groupID ...string) GroupCache {
	keys := make([]string, 0, len(groupID))
	for _, groupID := range groupID {
		keys = append(keys, g.getGroupMemberNumKey(groupID))
	}
	cache := g.NewCache()
	cache.AddKeys(keys...)

	return cache
}
