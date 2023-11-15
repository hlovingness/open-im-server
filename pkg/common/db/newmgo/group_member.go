package newmgo

import (
	"context"
	"github.com/OpenIMSDK/protocol/constant"
	"github.com/openimsdk/open-im-server/v3/pkg/common/db/newmgo/mgotool"
	"github.com/openimsdk/open-im-server/v3/pkg/common/db/table/relation"
	"github.com/openimsdk/open-im-server/v3/pkg/common/pagination"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewGroupMember(db *mongo.Database) (relation.GroupMemberModelInterface, error) {
	return &GroupMemberMgo{coll: db.Collection("group_member")}, nil
}

type GroupMemberMgo struct {
	coll *mongo.Collection
}

func (g *GroupMemberMgo) Create(ctx context.Context, groupMembers []*relation.GroupMemberModel) (err error) {
	return mgotool.InsertMany(ctx, g.coll, groupMembers)
}

func (g *GroupMemberMgo) Delete(ctx context.Context, groupID string, userIDs []string) (err error) {
	return mgotool.DeleteMany(ctx, g.coll, bson.M{"group_id": groupID, "user_id": bson.M{"$in": userIDs}})
}

func (g *GroupMemberMgo) Update(ctx context.Context, groupID string, userID string, data map[string]any) (err error) {
	return mgotool.UpdateOne(ctx, g.coll, bson.M{"group_id": groupID, "user_id": userID}, bson.M{"$set": data}, true)
}

func (g *GroupMemberMgo) Find(ctx context.Context, groupIDs []string, userIDs []string, roleLevels []int32) (groupMembers []*relation.GroupMemberModel, err error) {
	//TODO implement me
	panic("implement me")
}

func (g *GroupMemberMgo) FindMemberUserID(ctx context.Context, groupID string) (userIDs []string, err error) {
	return mgotool.Find[string](ctx, g.coll, bson.M{"group_id": groupID}, options.Find().SetProjection(bson.M{"user_id": 1}))
}

func (g *GroupMemberMgo) Take(ctx context.Context, groupID string, userID string) (groupMember *relation.GroupMemberModel, err error) {
	return mgotool.FindOne[*relation.GroupMemberModel](ctx, g.coll, bson.M{"group_id": groupID, "user_id": userID})
}

func (g *GroupMemberMgo) TakeOwner(ctx context.Context, groupID string) (groupMember *relation.GroupMemberModel, err error) {
	return mgotool.FindOne[*relation.GroupMemberModel](ctx, g.coll, bson.M{"group_id": groupID, "role_level": constant.GroupOwner})
}

func (g *GroupMemberMgo) SearchMember(ctx context.Context, keyword string, groupIDs []string, userIDs []string, roleLevels []int32, pagination pagination.Pagination) (total int64, groupList []*relation.GroupMemberModel, err error) {
	//TODO implement me
	panic("implement me")
}

func (g *GroupMemberMgo) FindUserJoinedGroupID(ctx context.Context, userID string) (groupIDs []string, err error) {
	return mgotool.Find[string](ctx, g.coll, bson.M{"user_id": userID}, options.Find().SetProjection(bson.M{"group_id": 1}))
}

func (g *GroupMemberMgo) TakeGroupMemberNum(ctx context.Context, groupID string) (count int64, err error) {
	return mgotool.Count(ctx, g.coll, bson.M{"group_id": groupID})
}

func (g *GroupMemberMgo) FindUserManagedGroupID(ctx context.Context, userID string) (groupIDs []string, err error) {
	filter := bson.M{
		"user_id": userID,
		"role_level": bson.M{
			"$in": []int{constant.GroupOwner, constant.GroupAdmin},
		},
	}
	return mgotool.Find[string](ctx, g.coll, filter, options.Find().SetProjection(bson.M{"group_id": 1}))
}
