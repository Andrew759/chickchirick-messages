package message

import (
	"chickchirick-messages/internal/controller/c_controller"
	"chickchirick-messages/internal/middleware"
	model "chickchirick-messages/internal/model/message"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type ReactionController struct {
	Controller c_controller.Controller
}

func (rc *ReactionController) RegisterRoutes() {
	e := rc.Controller.E
	auth := middleware.Auth(rc.Controller.DI.AuthClient)

	e.GET("/messages/reactions", auth, rc.ListReactionsBatch)
	e.GET("/messages/:id/reactions", auth, rc.ListReactions)
	e.PUT("/messages/:id/reactions", auth, rc.ToggleReaction)
}

func (rc *ReactionController) redisClient() *redis.Client {
	if rdb := rc.Controller.DI.RedisDecorator; rdb != nil {
		return rdb.Client
	}
	return nil
}

func (rc *ReactionController) currentUser(c *gin.Context) (model.UserRelation, bool) {
	raw, ok := c.Get("user_uuid")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return model.UserRelation{}, false
	}
	uuidStr, _ := raw.(string)
	if uuidStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return model.UserRelation{}, false
	}
	ur, err := model.GetUserRelationByUuid(c.Request.Context(), rc.Controller.DI.DBDecorator.GDB(), uuidStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user relation not found"})
		return model.UserRelation{}, false
	}
	return ur, true
}

type reactionAgg struct {
	Emoji   string `json:"emoji"`
	Count   int    `json:"count"`
	Me      bool   `json:"me"`
	UserIds []int  `json:"userIds,omitempty"`
}

func aggregateReactions(rows []model.Reaction, myUserID int) []reactionAgg {
	type bucket struct {
		count   int
		me      bool
		userIds []int
	}
	order := make([]string, 0)
	m := map[string]*bucket{}
	for _, r := range rows {
		b, ok := m[r.Emoji]
		if !ok {
			b = &bucket{}
			m[r.Emoji] = b
			order = append(order, r.Emoji)
		}
		b.count++
		b.userIds = append(b.userIds, r.UserId)
		if r.UserId == myUserID {
			b.me = true
		}
	}
	out := make([]reactionAgg, 0, len(order))
	for _, emoji := range order {
		b := m[emoji]
		out = append(out, reactionAgg{
			Emoji:   emoji,
			Count:   b.count,
			Me:      b.me,
			UserIds: b.userIds,
		})
	}
	return out
}

func (rc *ReactionController) ListReactions(c *gin.Context) {
	ur, ok := rc.currentUser(c)
	if !ok {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}
	rows, err := model.GetReactionsByMessageId(c.Request.Context(), rc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"messageId": id,
		"reactions": aggregateReactions(rows, ur.UserId),
	})
}

func (rc *ReactionController) ListReactionsBatch(c *gin.Context) {
	ur, ok := rc.currentUser(c)
	if !ok {
		return
	}
	raw := c.Query("ids")
	if raw == "" {
		c.JSON(http.StatusOK, gin.H{"items": []any{}})
		return
	}
	parts := strings.Split(raw, ",")
	ids := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.Atoi(p)
		if err != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) > 500 {
		ids = ids[:500]
	}
	rows, err := model.GetReactionsByMessageIds(c.Request.Context(), rc.Controller.DI.DBDecorator.GDB(), ids)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	byMsg := map[int][]model.Reaction{}
	for _, r := range rows {
		byMsg[r.MessageId] = append(byMsg[r.MessageId], r)
	}
	items := make([]gin.H, 0, len(byMsg))
	for mid, list := range byMsg {
		items = append(items, gin.H{
			"messageId": mid,
			"reactions": aggregateReactions(list, ur.UserId),
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

type toggleReactionRequest struct {
	Emoji string `json:"emoji"`
}

func (rc *ReactionController) ToggleReaction(c *gin.Context) {
	ur, ok := rc.currentUser(c)
	if !ok {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}
	var req toggleReactionRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Emoji) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "emoji is required"})
		return
	}
	emoji := strings.TrimSpace(req.Emoji)
	ctx := c.Request.Context()
	db := rc.Controller.DI.DBDecorator.GDB()

	if _, err := model.GetMessageById(ctx, db, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		return
	}

	action := "add"
	_, err = model.FindReaction(ctx, db, id, ur.UserId, emoji)
	if err == nil {
		if err := model.DeleteReaction(ctx, db, id, ur.UserId, emoji); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		action = "remove"
	} else if errors.Is(err, model.ReactionNotFoundErr) {
		r := model.Reaction{
			MessageId: id,
			UserId:    ur.UserId,
			Emoji:     emoji,
		}
		if err := model.CreateReaction(ctx, db, &r); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows, _ := model.GetReactionsByMessageId(ctx, db, id)
	agg := aggregateReactions(rows, ur.UserId)

	// fan-out
	_ = rc.publishReaction(c, id, ur.UserId, emoji, action, agg)

	c.JSON(http.StatusOK, gin.H{
		"messageId": id,
		"action":    action,
		"emoji":     emoji,
		"reactions": agg,
	})
}

func (rc *ReactionController) publishReaction(c *gin.Context, messageID, actorID int, emoji, action string, agg []reactionAgg) error {
	rdb := rc.redisClient()
	if rdb == nil {
		return nil
	}
	ctx := c.Request.Context()
	db := rc.Controller.DI.DBDecorator.GDB()

	userIds := map[int]struct{}{}
	userIds[actorID] = struct{}{}

	if p, err := model.GetPersonalByMessageId(ctx, db, messageID); err == nil {
		userIds[p.SenderId] = struct{}{}
		userIds[p.RecipientId] = struct{}{}
	}
	if g, err := model.GetGroupByMessageId(ctx, db, messageID); err == nil {
		if cg, err := model.GetGroupMetaByUuid(ctx, db, g.GroupId); err == nil {
			if ids, err := model.ListGroupMemberUserIds(ctx, db, cg.Id); err == nil {
				for _, id := range ids {
					userIds[id] = struct{}{}
				}
			}
		}
	}
	if cm, err := model.GetChannelMsgByMessageId(ctx, db, messageID); err == nil {
		if ch, err := model.GetChannelByUuid(ctx, db, cm.ChannelId); err == nil {
			if ids, err := model.ListChannelMemberUserIds(ctx, db, ch.Id); err == nil {
				for _, id := range ids {
					userIds[id] = struct{}{}
				}
			}
		}
	}

	payload, err := json.Marshal(map[string]any{
		"message_id": messageID,
		"emoji":      emoji,
		"action":     action,
		"user_id":    actorID,
		"reactions":  agg,
	})
	if err != nil {
		return err
	}
	// Транспорт через new_message + prefix — без изменения protobuf MessageEvent.
	eventData, err := json.Marshal(map[string]any{
		"type": "new_message",
		"message": map[string]any{
			"id":       messageID,
			"senderId": actorID,
			"text":     "__REACTION__" + string(payload),
		},
	})
	if err != nil {
		return err
	}
	for uid := range userIds {
		if uid <= 0 {
			continue
		}
		_ = rdb.Publish(ctx, fmt.Sprintf("user_events_%d", uid), eventData).Err()
	}
	return nil
}
