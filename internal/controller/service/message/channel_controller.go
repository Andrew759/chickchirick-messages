package message

import (
	"chickchirick-messages/internal/controller/c_controller"
	"chickchirick-messages/internal/middleware"
	model "chickchirick-messages/internal/model/message"
	"chickchirick-messages/internal/service"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type ChannelController struct {
	Controller c_controller.Controller
}

func (cc *ChannelController) RegisterRoutes() {
	e := cc.Controller.E
	auth := middleware.Auth(cc.Controller.DI.AuthClient)

	e.GET("/channels", auth, cc.ListMyChannels)
	e.GET("/channels/public", auth, cc.ListPublicChannels)
	e.POST("/channels", auth, cc.CreateChannel)
	e.GET("/channels/:uuid", auth, cc.GetChannel)
	e.PATCH("/channels/:uuid", auth, cc.UpdateChannel)
	e.DELETE("/channels/:uuid", auth, cc.DeleteChannel)

	e.GET("/channels/:uuid/members", auth, cc.ListMembers)
	e.POST("/channels/:uuid/members", auth, cc.AddMembers)
	e.PATCH("/channels/:uuid/members/:userId", auth, cc.UpdateMemberRole)
	e.DELETE("/channels/:uuid/members/:userId", auth, cc.RemoveMember)

	e.POST("/channels/:uuid/subscribe", auth, cc.Subscribe)
	e.DELETE("/channels/:uuid/subscribe", auth, cc.Unsubscribe)

	e.POST("/channels/:uuid/messages", auth, cc.SendChannelMessage)
	e.GET("/channels/:uuid/messages", auth, cc.ListChannelMessages)
	e.POST("/channels/:uuid/typing", auth, cc.ChannelTyping)
}

func (cc *ChannelController) currentUser(c *gin.Context) (model.UserRelation, bool) {
	raw, ok := c.Get("user_uuid")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return model.UserRelation{}, false
	}
	userUUID, _ := raw.(string)
	if userUUID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return model.UserRelation{}, false
	}
	ur, err := model.GetUserRelationByUuid(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), userUUID)
	if err != nil || ur.UserId == 0 {
		parsed, perr := uuid.Parse(userUUID)
		if perr != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return model.UserRelation{}, false
		}
		ur = model.UserRelation{UserUuid: parsed}
		if err := model.CreateUserRelation(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), &ur); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve user"})
			return model.UserRelation{}, false
		}
	}
	return ur, true
}

func (cc *ChannelController) redisClient() *redis.Client {
	if rdb := cc.Controller.DI.RedisDecorator; rdb != nil {
		return rdb.Client
	}
	return nil
}

type createChannelRequest struct {
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	AvatarFileUuid *string `json:"avatar_file_uuid"`
	IsPublic       bool    `json:"is_public"`
	// E2EEEnabled default true; для публичных без шифрования передать false
	E2EEEnabled   *bool `json:"e2ee_enabled"`
	MemberUserIds []int `json:"member_user_ids"`
}

type channelResponse struct {
	Id             int     `json:"id"`
	ChannelUuid    string  `json:"channel_uuid"`
	Title          string  `json:"title"`
	Description    string  `json:"description,omitempty"`
	AvatarFileUuid *string `json:"avatar_file_uuid,omitempty"`
	IsPublic       bool    `json:"is_public"`
	E2EEEnabled    bool    `json:"e2ee_enabled"`
	IsChannel      bool    `json:"is_channel"` // для фронта (ChatList)
	CreatedBy      int     `json:"created_by"`
	MemberCount    int     `json:"member_count,omitempty"`
	MyRole         string  `json:"my_role,omitempty"`
}

func toChannelResponse(ch model.Channel, memberCount int, myRole string) channelResponse {
	resp := channelResponse{
		Id:          ch.Id,
		ChannelUuid: ch.ChannelUuid.String(),
		Title:       ch.Title,
		Description: ch.Description,
		IsPublic:    ch.IsPublic,
		E2EEEnabled: ch.E2EEEnabled,
		IsChannel:   true,
		CreatedBy:   ch.CreatedBy,
		MemberCount: memberCount,
		MyRole:      myRole,
	}
	if ch.AvatarFileUuid != nil {
		s := ch.AvatarFileUuid.String()
		resp.AvatarFileUuid = &s
	}
	return resp
}

func (cc *ChannelController) CreateChannel(c *gin.Context) {
	ur, ok := cc.currentUser(c)
	if !ok {
		return
	}

	var req createChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}
	if req.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}

	e2ee := true
	if req.E2EEEnabled != nil {
		e2ee = *req.E2EEEnabled
	}

	var avatar *uuid.UUID
	if req.AvatarFileUuid != nil && *req.AvatarFileUuid != "" {
		parsed, err := uuid.Parse(*req.AvatarFileUuid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid avatar_file_uuid"})
			return
		}
		avatar = &parsed
	}

	ch, members, err := service.CreateChannel(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ur, service.CreateChannelInput{
		Title:          req.Title,
		Description:    req.Description,
		AvatarFileUuid: avatar,
		IsPublic:       req.IsPublic,
		E2EEEnabled:    e2ee,
		MemberUserIds:  req.MemberUserIds,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create channel: " + err.Error()})
		return
	}

	rdbClient := cc.redisClient()
	notifyIDs := make([]int, 0, len(members))
	for _, m := range members {
		notifyIDs = append(notifyIDs, m.UserId)
	}
	_ = service.PublishChannelAdded(
		c.Request.Context(),
		rdbClient,
		notifyIDs,
		service.ChannelToPayload(ch, len(members)),
	)

	// Системное сообщение (plaintext system; клиент может не шифровать system)
	sysText := "Канал «" + ch.Title + "» создан"
	_, _, _ = service.CreateChannelMessage(
		c.Request.Context(),
		cc.Controller.DI.DBDecorator.GDB(),
		rdbClient,
		ur,
		ch.ChannelUuid,
		sysText,
		nil,
	)

	c.JSON(http.StatusCreated, toChannelResponse(ch, len(members), model.ChannelRoleOwner))
}

func (cc *ChannelController) ListMyChannels(c *gin.Context) {
	ur, ok := cc.currentUser(c)
	if !ok {
		return
	}

	channels, err := model.ListChannelsForUser(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ur.UserId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]channelResponse, 0, len(channels))
	for _, ch := range channels {
		ids, _ := model.ListChannelMemberUserIds(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id)
		role := ""
		if m, err := model.GetChannelMember(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id, ur.UserId); err == nil {
			role = m.Role
		}
		out = append(out, toChannelResponse(ch, len(ids), role))
	}
	c.JSON(http.StatusOK, out)
}

func (cc *ChannelController) ListPublicChannels(c *gin.Context) {
	_, ok := cc.currentUser(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	q := c.Query("q")

	channels, err := model.ListPublicChannels(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), limit, offset, q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]channelResponse, 0, len(channels))
	for _, ch := range channels {
		n, _ := model.CountChannelMembers(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id)
		out = append(out, toChannelResponse(ch, int(n), ""))
	}
	c.JSON(http.StatusOK, out)
}

func (cc *ChannelController) GetChannel(c *gin.Context) {
	ur, ok := cc.currentUser(c)
	if !ok {
		return
	}
	channelUuid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel uuid"})
		return
	}

	ch, err := model.GetChannelByUuid(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), channelUuid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	okMember, _ := model.IsChannelMember(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id, ur.UserId)
	if !okMember && !ch.IsPublic {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member"})
		return
	}

	ids, _ := model.ListChannelMemberUserIds(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id)
	role := ""
	if m, err := model.GetChannelMember(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id, ur.UserId); err == nil {
		role = m.Role
	}
	c.JSON(http.StatusOK, toChannelResponse(ch, len(ids), role))
}

type updateChannelRequest struct {
	Title          *string `json:"title"`
	Description    *string `json:"description"`
	AvatarFileUuid *string `json:"avatar_file_uuid"`
	IsPublic       *bool   `json:"is_public"`
}

func (cc *ChannelController) UpdateChannel(c *gin.Context) {
	ur, ok := cc.currentUser(c)
	if !ok {
		return
	}
	channelUuid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel uuid"})
		return
	}

	ch, err := model.GetChannelByUuid(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), channelUuid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}
	member, err := model.GetChannelMember(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id, ur.UserId)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member"})
		return
	}
	if member.Role != model.ChannelRoleOwner && member.Role != model.ChannelRoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "only owner/admin can update channel"})
		return
	}

	var req updateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	if req.Title != nil && *req.Title != "" {
		ch.Title = *req.Title
	}
	if req.Description != nil {
		ch.Description = *req.Description
	}
	if req.IsPublic != nil {
		ch.IsPublic = *req.IsPublic
	}
	if req.AvatarFileUuid != nil {
		if *req.AvatarFileUuid == "" {
			ch.AvatarFileUuid = nil
		} else {
			parsed, err := uuid.Parse(*req.AvatarFileUuid)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid avatar_file_uuid"})
				return
			}
			ch.AvatarFileUuid = &parsed
		}
	}
	if err := model.UpdateChannel(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), &ch); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ids, _ := model.ListChannelMemberUserIds(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id)
	c.JSON(http.StatusOK, toChannelResponse(ch, len(ids), member.Role))
}

func (cc *ChannelController) DeleteChannel(c *gin.Context) {
	ur, ok := cc.currentUser(c)
	if !ok {
		return
	}
	channelUuid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel uuid"})
		return
	}
	ch, err := model.GetChannelByUuid(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), channelUuid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}
	member, err := model.GetChannelMember(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id, ur.UserId)
	if err != nil || member.Role != model.ChannelRoleOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "only owner can delete channel"})
		return
	}
	if err := model.DeleteChannel(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (cc *ChannelController) ListMembers(c *gin.Context) {
	ur, ok := cc.currentUser(c)
	if !ok {
		return
	}
	channelUuid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel uuid"})
		return
	}
	ch, err := model.GetChannelByUuid(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), channelUuid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}
	okMember, _ := model.IsChannelMember(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id, ur.UserId)
	if !okMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member"})
		return
	}
	members, err := model.ListChannelMembers(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, members)
}

type addChannelMembersRequest struct {
	MemberUserIds []int  `json:"member_user_ids"`
	Role          string `json:"role"` // optional: admin | subscriber
}

func (cc *ChannelController) AddMembers(c *gin.Context) {
	ur, ok := cc.currentUser(c)
	if !ok {
		return
	}
	channelUuid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel uuid"})
		return
	}
	ch, err := model.GetChannelByUuid(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), channelUuid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}
	member, err := model.GetChannelMember(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id, ur.UserId)
	if err != nil || (member.Role != model.ChannelRoleOwner && member.Role != model.ChannelRoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only owner/admin can add members"})
		return
	}
	var req addChannelMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	role := model.ChannelRoleSubscriber
	if req.Role == model.ChannelRoleAdmin {
		role = model.ChannelRoleAdmin
	}

	added := 0
	addedIDs := make([]int, 0)
	for _, uid := range req.MemberUserIds {
		if uid <= 0 {
			continue
		}
		exists, _ := model.IsChannelMember(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id, uid)
		if exists {
			continue
		}
		m := model.ChannelMember{ChannelId: ch.Id, UserId: uid, Role: role}
		if err := model.CreateChannelMember(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), &m); err == nil {
			added++
			addedIDs = append(addedIDs, uid)
		}
	}

	if len(addedIDs) > 0 {
		ids, _ := model.ListChannelMemberUserIds(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id)
		rdbClient := cc.redisClient()
		_ = service.PublishChannelAdded(
			c.Request.Context(),
			rdbClient,
			addedIDs,
			service.ChannelToPayload(ch, len(ids)),
		)
		if ch.E2EEEnabled {
			_ = service.PublishChannelMembersChanged(
				c.Request.Context(),
				rdbClient,
				ids,
				service.ChannelMembersChangedPayload{
					ChannelUuid:  ch.ChannelUuid.String(),
					Title:        ch.Title,
					AddedUserIds: addedIDs,
					MemberCount:  len(ids),
					ActorUserId:  ur.UserId,
					E2EEEnabled:  ch.E2EEEnabled,
				},
			)
		}
	}

	c.JSON(http.StatusOK, gin.H{"added": added})
}

type updateMemberRoleRequest struct {
	Role string `json:"role"`
}

func (cc *ChannelController) UpdateMemberRole(c *gin.Context) {
	ur, ok := cc.currentUser(c)
	if !ok {
		return
	}
	channelUuid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel uuid"})
		return
	}
	targetId, err := strconv.Atoi(c.Param("userId"))
	if err != nil || targetId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid userId"})
		return
	}
	ch, err := model.GetChannelByUuid(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), channelUuid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}
	actor, err := model.GetChannelMember(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id, ur.UserId)
	if err != nil || actor.Role != model.ChannelRoleOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "only owner can change roles"})
		return
	}
	var req updateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	if req.Role != model.ChannelRoleAdmin && req.Role != model.ChannelRoleSubscriber {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be admin or subscriber"})
		return
	}
	if targetId == ur.UserId {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot change own role this way"})
		return
	}
	if err := model.UpdateChannelMemberRole(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id, targetId, req.Role); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user_id": targetId, "role": req.Role})
}

func (cc *ChannelController) RemoveMember(c *gin.Context) {
	ur, ok := cc.currentUser(c)
	if !ok {
		return
	}
	channelUuid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel uuid"})
		return
	}
	targetId, err := strconv.Atoi(c.Param("userId"))
	if err != nil || targetId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid userId"})
		return
	}
	ch, err := model.GetChannelByUuid(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), channelUuid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}
	member, err := model.GetChannelMember(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id, ur.UserId)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member"})
		return
	}
	// self-leave или owner/admin kick
	if ur.UserId != targetId && member.Role != model.ChannelRoleOwner && member.Role != model.ChannelRoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	target, err := model.GetChannelMember(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id, targetId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
		return
	}
	if target.Role == model.ChannelRoleOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot remove owner"})
		return
	}
	if err := model.DeleteChannelMember(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id, targetId); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	ids, _ := model.ListChannelMemberUserIds(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id)
	rdbClient := cc.redisClient()
	if ch.E2EEEnabled {
		_ = service.PublishChannelMembersChanged(
			c.Request.Context(),
			rdbClient,
			ids,
			service.ChannelMembersChangedPayload{
				ChannelUuid:    ch.ChannelUuid.String(),
				Title:          ch.Title,
				RemovedUserIds: []int{targetId},
				MemberCount:    len(ids),
				ActorUserId:    ur.UserId,
				E2EEEnabled:    ch.E2EEEnabled,
			},
		)
	}
	c.Status(http.StatusNoContent)
}

func (cc *ChannelController) Subscribe(c *gin.Context) {
	ur, ok := cc.currentUser(c)
	if !ok {
		return
	}
	channelUuid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel uuid"})
		return
	}
	ch, err := service.SubscribeToChannel(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), cc.redisClient(), ur, channelUuid)
	if err != nil {
		if errors.Is(err, model.ChannelForbiddenErr) {
			c.JSON(http.StatusForbidden, gin.H{"error": "private channel: ask admin to invite"})
			return
		}
		if errors.Is(err, model.ChannelNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	n, _ := model.CountChannelMembers(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id)
	c.JSON(http.StatusOK, toChannelResponse(ch, int(n), model.ChannelRoleSubscriber))
}

func (cc *ChannelController) Unsubscribe(c *gin.Context) {
	ur, ok := cc.currentUser(c)
	if !ok {
		return
	}
	channelUuid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel uuid"})
		return
	}
	if err := service.UnsubscribeFromChannel(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), cc.redisClient(), ur, channelUuid); err != nil {
		if errors.Is(err, model.ChannelNotFoundErr) || errors.Is(err, model.ChannelMemberNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

type sendChannelMessageRequest struct {
	Text             string `json:"text"`
	RespondMessageId *int64 `json:"respond_message_id"`
}

func (cc *ChannelController) SendChannelMessage(c *gin.Context) {
	ur, ok := cc.currentUser(c)
	if !ok {
		return
	}
	channelUuid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel uuid"})
		return
	}
	var req sendChannelMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	if req.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text is required"})
		return
	}

	rdb := cc.redisClient()
	msg, ch, err := service.CreateChannelMessage(
		c.Request.Context(),
		cc.Controller.DI.DBDecorator.GDB(),
		rdb,
		ur,
		channelUuid,
		req.Text,
		req.RespondMessageId,
	)
	if err != nil {
		if errors.Is(err, model.ChannelPostForbiddenErr) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, model.ChannelNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         msg.Id,
		"senderId":   ur.UserId,
		"channelId":  ch.ChannelUuid.String(),
		"text":       msg.Text,
		"createdAt":  msg.CreatedAt,
		"is_channel": true,
	})
}

func (cc *ChannelController) ListChannelMessages(c *gin.Context) {
	ur, ok := cc.currentUser(c)
	if !ok {
		return
	}
	channelUuid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel uuid"})
		return
	}
	ch, err := model.GetChannelByUuid(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), channelUuid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}
	okMember, _ := model.IsChannelMember(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), ch.Id, ur.UserId)
	if !okMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	links, err := model.GetChannelMsgsByChannelId(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), channelUuid, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type msgOut struct {
		Id        int    `json:"id"`
		SenderId  int    `json:"senderId"`
		ChannelId string `json:"channelId"`
		Text      string `json:"text"`
		CreatedAt string `json:"createdAt"`
	}
	out := make([]msgOut, 0, len(links))
	for _, link := range links {
		m, err := model.GetMessageById(c.Request.Context(), cc.Controller.DI.DBDecorator.GDB(), link.MessageId)
		if err != nil {
			continue
		}
		out = append(out, msgOut{
			Id:        m.Id,
			SenderId:  link.SenderId,
			ChannelId: channelUuid.String(),
			Text:      m.Text,
			CreatedAt: m.CreatedAt.UTC().Format("2006-01-02T15:04:05.000000Z07:00"),
		})
	}
	c.JSON(http.StatusOK, out)
}

type channelTypingRequest struct {
	Typing bool `json:"typing"`
}

func (cc *ChannelController) ChannelTyping(c *gin.Context) {
	ur, ok := cc.currentUser(c)
	if !ok {
		return
	}
	channelUuid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel uuid"})
		return
	}
	var req channelTypingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	if err := service.PublishChannelTyping(
		c.Request.Context(),
		cc.Controller.DI.DBDecorator.GDB(),
		cc.redisClient(),
		channelUuid,
		ur.UserId,
		req.Typing,
	); err != nil {
		if errors.Is(err, model.ChannelPostForbiddenErr) || errors.Is(err, model.ChannelForbiddenErr) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, model.ChannelNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
