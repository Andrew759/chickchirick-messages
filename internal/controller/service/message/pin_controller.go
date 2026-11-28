package message

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"chickchirick-messages/internal/controller/c_controller"
	"chickchirick-messages/internal/middleware"
	model "chickchirick-messages/internal/model/message"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type PinController struct {
	Controller c_controller.Controller
}

func (pc *PinController) RegisterRoutes() {
	e := pc.Controller.E
	auth := middleware.Auth(pc.Controller.DI.AuthClient)

	e.GET("/messages/pins", auth, pc.ListPins)
	e.PUT("/messages/:id/pin", auth, pc.PinMessage)
	e.DELETE("/messages/:id/pin", auth, pc.UnpinMessage)
}

func (pc *PinController) redisClient() *redis.Client {
	if rdb := pc.Controller.DI.RedisDecorator; rdb != nil {
		return rdb.Client
	}
	return nil
}

func (pc *PinController) db() *gorm.DB {
	return pc.Controller.DI.DBDecorator.GDB()
}

func (pc *PinController) currentUser(c *gin.Context) (model.UserRelation, bool) {
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
	ur, err := model.GetUserRelationByUuid(c.Request.Context(), pc.db(), uuidStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user relation not found"})
		return model.UserRelation{}, false
	}
	return ur, true
}

type pinScope struct {
	peerLow     *int
	peerHigh    *int
	groupUuid   *uuid.UUID
	channelUuid *uuid.UUID
	userIds     []int
}

func peerPair(a, b int) (int, int) {
	if a < b {
		return a, b
	}
	return b, a
}

func (pc *PinController) resolveScopeFromMessage(ctx context.Context, messageID int) (pinScope, error) {
	db := pc.db()
	var s pinScope

	if p, err := model.GetPersonalByMessageId(ctx, db, messageID); err == nil {
		low, high := peerPair(p.SenderId, p.RecipientId)
		s.peerLow, s.peerHigh = &low, &high
		s.userIds = []int{p.SenderId, p.RecipientId}
		return s, nil
	}
	if g, err := model.GetGroupByMessageId(ctx, db, messageID); err == nil {
		gid := g.GroupId
		s.groupUuid = &gid
		if cg, err := model.GetGroupMetaByUuid(ctx, db, gid); err == nil {
			if ids, err := model.ListGroupMemberUserIds(ctx, db, cg.Id); err == nil {
				s.userIds = ids
			}
		}
		return s, nil
	}
	if cm, err := model.GetChannelMsgByMessageId(ctx, db, messageID); err == nil {
		cid := cm.ChannelId
		s.channelUuid = &cid
		if ch, err := model.GetChannelByUuid(ctx, db, cid); err == nil {
			if ids, err := model.ListChannelMemberUserIds(ctx, db, ch.Id); err == nil {
				s.userIds = ids
			}
		}
		return s, nil
	}
	return s, model.PinScopeErr
}

type pinScopeQuery struct {
	PeerId      int    `form:"peer_id"`
	GroupUuid   string `form:"group_uuid"`
	ChannelUuid string `form:"channel_uuid"`
}

func (pc *PinController) scopeFromQuery(c *gin.Context, me model.UserRelation) (pinScope, error) {
	var q pinScopeQuery
	_ = c.ShouldBindQuery(&q)
	var s pinScope
	if q.GroupUuid != "" {
		gid, err := uuid.Parse(q.GroupUuid)
		if err != nil {
			return s, err
		}
		s.groupUuid = &gid
		return s, nil
	}
	if q.ChannelUuid != "" {
		cid, err := uuid.Parse(q.ChannelUuid)
		if err != nil {
			return s, err
		}
		s.channelUuid = &cid
		return s, nil
	}
	if q.PeerId > 0 {
		low, high := peerPair(me.UserId, q.PeerId)
		s.peerLow, s.peerHigh = &low, &high
		return s, nil
	}
	return s, model.PinScopeErr
}

type pinDTO struct {
	MessageId int    `json:"messageId"`
	Text      string `json:"text,omitempty"`
	PinnedBy  int    `json:"pinnedBy"`
}

func previewText(raw string) string {
	text := raw
	if strings.HasPrefix(text, "__GID__:") || strings.HasPrefix(text, "__CID__:") {
		if i := strings.Index(text, "\n"); i >= 0 {
			text = text[i+1:]
		}
	}
	text = strings.TrimSpace(text)
	if len(text) > 120 {
		return text[:117] + "…"
	}
	return text
}

func (pc *PinController) pinsDTO(ctx context.Context, rows []model.Pin) []pinDTO {
	out := make([]pinDTO, 0, len(rows))
	db := pc.db()
	for _, r := range rows {
		text := ""
		if m, err := model.GetMessageById(ctx, db, r.MessageId); err == nil {
			text = previewText(m.Text)
		}
		out = append(out, pinDTO{
			MessageId: r.MessageId,
			Text:      text,
			PinnedBy:  r.PinnedBy,
		})
	}
	return out
}

func (pc *PinController) ListPins(c *gin.Context) {
	ur, ok := pc.currentUser(c)
	if !ok {
		return
	}
	scope, err := pc.scopeFromQuery(c, ur)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "peer_id, group_uuid or channel_uuid required"})
		return
	}
	ctx := c.Request.Context()
	rows, err := model.ListPinsByScope(ctx, pc.db(), scope.peerLow, scope.peerHigh, scope.groupUuid, scope.channelUuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"pins": pc.pinsDTO(ctx, rows)})
}

func (pc *PinController) PinMessage(c *gin.Context) {
	ur, ok := pc.currentUser(c)
	if !ok {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}
	ctx := c.Request.Context()
	db := pc.db()
	if _, err := model.GetMessageById(ctx, db, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		return
	}
	scope, err := pc.resolveScopeFromMessage(ctx, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot resolve chat scope for message"})
		return
	}
	if _, err := model.FindPin(ctx, db, id, scope.peerLow, scope.peerHigh, scope.groupUuid, scope.channelUuid); err == nil {
		rows, _ := model.ListPinsByScope(ctx, db, scope.peerLow, scope.peerHigh, scope.groupUuid, scope.channelUuid)
		c.JSON(http.StatusOK, gin.H{"action": "pinned", "messageId": id, "pins": pc.pinsDTO(ctx, rows)})
		return
	} else if !errors.Is(err, model.PinNotFoundErr) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	p := model.Pin{
		MessageId:    id,
		PeerUserLow:  scope.peerLow,
		PeerUserHigh: scope.peerHigh,
		GroupUuid:    scope.groupUuid,
		ChannelUuid:  scope.channelUuid,
		PinnedBy:     ur.UserId,
	}
	if err := model.CreatePin(ctx, db, &p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	rows, _ := model.ListPinsByScope(ctx, db, scope.peerLow, scope.peerHigh, scope.groupUuid, scope.channelUuid)
	dtos := pc.pinsDTO(ctx, rows)
	_ = pc.publishPin(ctx, id, ur.UserId, "pin", scope, dtos)
	c.JSON(http.StatusOK, gin.H{"action": "pinned", "messageId": id, "pins": dtos})
}

func (pc *PinController) UnpinMessage(c *gin.Context) {
	ur, ok := pc.currentUser(c)
	if !ok {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}
	ctx := c.Request.Context()
	db := pc.db()
	scope, err := pc.resolveScopeFromMessage(ctx, id)
	if err != nil {
		scope, err = pc.scopeFromQuery(c, ur)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot resolve chat scope"})
			return
		}
	}
	if err := model.DeletePinByMessageAndScope(ctx, db, id, scope.peerLow, scope.peerHigh, scope.groupUuid, scope.channelUuid); err != nil && !errors.Is(err, model.PinNotFoundErr) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	rows, _ := model.ListPinsByScope(ctx, db, scope.peerLow, scope.peerHigh, scope.groupUuid, scope.channelUuid)
	dtos := pc.pinsDTO(ctx, rows)
	_ = pc.publishPin(ctx, id, ur.UserId, "unpin", scope, dtos)
	c.JSON(http.StatusOK, gin.H{"action": "unpinned", "messageId": id, "pins": dtos})
}

func (pc *PinController) publishPin(ctx context.Context, messageID, actorID int, action string, scope pinScope, pins []pinDTO) error {
	rdb := pc.redisClient()
	if rdb == nil {
		return nil
	}
	userIds := map[int]struct{}{}
	userIds[actorID] = struct{}{}
	for _, id := range scope.userIds {
		if id > 0 {
			userIds[id] = struct{}{}
		}
	}
	payload, err := json.Marshal(map[string]any{
		"message_id": messageID,
		"action":     action,
		"user_id":    actorID,
		"pins":       pins,
	})
	if err != nil {
		return err
	}
	eventData, err := json.Marshal(map[string]any{
		"type": "new_message",
		"message": map[string]any{
			"id":       messageID,
			"senderId": actorID,
			"text":     "__PIN__" + string(payload),
		},
	})
	if err != nil {
		return err
	}
	for uid := range userIds {
		_ = rdb.Publish(ctx, fmt.Sprintf("user_events_%d", uid), eventData).Err()
	}
	return nil
}
