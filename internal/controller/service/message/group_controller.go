package message

import (
	"chickchirick-messages/internal/controller/c_controller"
	"chickchirick-messages/internal/middleware"
	model "chickchirick-messages/internal/model/message"
	"chickchirick-messages/internal/service"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GroupController struct {
	Controller c_controller.Controller
}

func (gc *GroupController) RegisterRoutes() {
	e := gc.Controller.E
	auth := middleware.Auth(gc.Controller.DI.AuthClient)

	// legacy CRUD по строкам groups (message→group link) оставляем
	e.GET("/groups", gc.GetGroups)
	e.POST("/group", gc.CreateGroup)
	e.GET("/group/:id", gc.GetGroup)
	e.PUT("/group/:id", gc.UpdateGroup)
	e.DELETE("/group/:id", gc.DeleteGroup)

	// полноценные групповые чаты
	e.GET("/group-chats", auth, gc.ListMyGroupChats)
	e.POST("/group-chats", auth, gc.CreateGroupChat)
	e.GET("/group-chats/:uuid", auth, gc.GetGroupChat)
	e.PATCH("/group-chats/:uuid", auth, gc.UpdateGroupChat)
	e.GET("/group-chats/:uuid/members", auth, gc.ListMembers)
	e.POST("/group-chats/:uuid/members", auth, gc.AddMembers)
	e.DELETE("/group-chats/:uuid/members/:userId", auth, gc.RemoveMember)
	e.POST("/group-chats/:uuid/messages", auth, gc.SendGroupMessage)
}

func (gc *GroupController) currentUser(c *gin.Context) (model.UserRelation, bool) {
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
	ur, err := model.GetUserRelationByUuid(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), userUUID)
	if err != nil || ur.UserId == 0 {
		// авто-создание relation при первом обращении
		parsed, perr := uuid.Parse(userUUID)
		if perr != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return model.UserRelation{}, false
		}
		ur = model.UserRelation{UserUuid: parsed}
		if err := model.CreateUserRelation(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), &ur); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve user"})
			return model.UserRelation{}, false
		}
	}
	return ur, true
}

type createGroupChatRequest struct {
	Title          string  `json:"title"`
	AvatarFileUuid *string `json:"avatar_file_uuid"`
	MemberUserIds  []int   `json:"member_user_ids"`
}

type groupChatResponse struct {
	Id             int     `json:"id"`
	GroupUuid      string  `json:"group_uuid"`
	Title          string  `json:"title"`
	AvatarFileUuid *string `json:"avatar_file_uuid,omitempty"`
	CreatedBy      int     `json:"created_by"`
	MemberCount    int     `json:"member_count,omitempty"`
}

func toGroupChatResponse(g model.ChatGroup, memberCount int) groupChatResponse {
	resp := groupChatResponse{
		Id:          g.Id,
		GroupUuid:   g.GroupUuid.String(),
		Title:       g.Title,
		CreatedBy:   g.CreatedBy,
		MemberCount: memberCount,
	}
	if g.AvatarFileUuid != nil {
		s := g.AvatarFileUuid.String()
		resp.AvatarFileUuid = &s
	}
	return resp
}

func (gc *GroupController) CreateGroupChat(c *gin.Context) {
	ur, ok := gc.currentUser(c)
	if !ok {
		return
	}

	var req createGroupChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}
	if req.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
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

	group, members, err := service.CreateGroupChat(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), ur, service.CreateGroupInput{
		Title:          req.Title,
		AvatarFileUuid: avatar,
		MemberUserIds:  req.MemberUserIds,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create group: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, toGroupChatResponse(group, len(members)))
}

func (gc *GroupController) ListMyGroupChats(c *gin.Context) {
	ur, ok := gc.currentUser(c)
	if !ok {
		return
	}

	groups, err := model.ListChatGroupsForUser(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), ur.UserId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]groupChatResponse, 0, len(groups))
	for _, g := range groups {
		ids, _ := model.ListGroupMemberUserIds(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), g.Id)
		out = append(out, toGroupChatResponse(g, len(ids)))
	}
	c.JSON(http.StatusOK, out)
}

func (gc *GroupController) GetGroupChat(c *gin.Context) {
	ur, ok := gc.currentUser(c)
	if !ok {
		return
	}
	groupUuid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group uuid"})
		return
	}

	g, err := model.GetChatGroupByUuid(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), groupUuid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	okMember, err := model.IsGroupMember(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), g.Id, ur.UserId)
	if err != nil || !okMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member"})
		return
	}
	ids, _ := model.ListGroupMemberUserIds(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), g.Id)
	c.JSON(http.StatusOK, toGroupChatResponse(g, len(ids)))
}

type updateGroupChatRequest struct {
	Title          *string `json:"title"`
	AvatarFileUuid *string `json:"avatar_file_uuid"`
}

func (gc *GroupController) UpdateGroupChat(c *gin.Context) {
	ur, ok := gc.currentUser(c)
	if !ok {
		return
	}
	groupUuid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group uuid"})
		return
	}

	g, err := model.GetChatGroupByUuid(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), groupUuid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	member, err := model.GetGroupMember(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), g.Id, ur.UserId)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member"})
		return
	}
	if member.Role != model.GroupRoleOwner && member.Role != model.GroupRoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "only owner/admin can update group"})
		return
	}

	var req updateGroupChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	if req.Title != nil && *req.Title != "" {
		g.Title = *req.Title
	}
	if req.AvatarFileUuid != nil {
		if *req.AvatarFileUuid == "" {
			g.AvatarFileUuid = nil
		} else {
			parsed, err := uuid.Parse(*req.AvatarFileUuid)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid avatar_file_uuid"})
				return
			}
			g.AvatarFileUuid = &parsed
		}
	}
	if err := model.UpdateChatGroup(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), &g); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ids, _ := model.ListGroupMemberUserIds(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), g.Id)
	c.JSON(http.StatusOK, toGroupChatResponse(g, len(ids)))
}

func (gc *GroupController) ListMembers(c *gin.Context) {
	ur, ok := gc.currentUser(c)
	if !ok {
		return
	}
	groupUuid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group uuid"})
		return
	}
	g, err := model.GetChatGroupByUuid(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), groupUuid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	okMember, _ := model.IsGroupMember(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), g.Id, ur.UserId)
	if !okMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member"})
		return
	}
	members, err := model.ListGroupMembers(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), g.Id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, members)
}

type addMembersRequest struct {
	MemberUserIds []int `json:"member_user_ids"`
}

func (gc *GroupController) AddMembers(c *gin.Context) {
	ur, ok := gc.currentUser(c)
	if !ok {
		return
	}
	groupUuid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group uuid"})
		return
	}
	g, err := model.GetChatGroupByUuid(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), groupUuid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	member, err := model.GetGroupMember(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), g.Id, ur.UserId)
	if err != nil || (member.Role != model.GroupRoleOwner && member.Role != model.GroupRoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only owner/admin can add members"})
		return
	}
	var req addMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	added := 0
	for _, uid := range req.MemberUserIds {
		if uid <= 0 {
			continue
		}
		exists, _ := model.IsGroupMember(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), g.Id, uid)
		if exists {
			continue
		}
		m := model.ChatGroupMember{GroupId: g.Id, UserId: uid, Role: model.GroupRoleMember}
		if err := model.CreateChatGroupMember(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), &m); err == nil {
			added++
		}
	}
	c.JSON(http.StatusOK, gin.H{"added": added})
}

func (gc *GroupController) RemoveMember(c *gin.Context) {
	ur, ok := gc.currentUser(c)
	if !ok {
		return
	}
	groupUuid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group uuid"})
		return
	}
	targetId := 0
	if _, err := fmtSscanf(c.Param("userId"), &targetId); err != nil || targetId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	g, err := model.GetChatGroupByUuid(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), groupUuid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	member, err := model.GetGroupMember(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), g.Id, ur.UserId)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member"})
		return
	}
	if ur.UserId != targetId && member.Role != model.GroupRoleOwner && member.Role != model.GroupRoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if err := model.DeleteGroupMember(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), g.Id, targetId); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

type sendGroupMessageRequest struct {
	Text             string `json:"text"`
	RespondMessageId *int64 `json:"respond_message_id"`
}

func (gc *GroupController) SendGroupMessage(c *gin.Context) {
	ur, ok := gc.currentUser(c)
	if !ok {
		return
	}
	groupUuid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group uuid"})
		return
	}
	var req sendGroupMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	if req.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text is required"})
		return
	}

	rdb := gc.Controller.DI.RedisDecorator.Client
	msg, group, err := service.CreateGroupMessage(
		c.Request.Context(),
		gc.Controller.DI.DBDecorator.GDB(),
		rdb,
		ur,
		groupUuid,
		req.Text,
		req.RespondMessageId,
	)
	if err != nil {
		if errors.Is(err, model.ChatGroupForbiddenErr) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, model.ChatGroupNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          msg.Id,
		"senderId":    ur.UserId,
		"recipientId": 0,
		"groupId":     group.GroupUuid.String(),
		"text":        msg.Text,
		"createdAt":   msg.CreatedAt,
	})
}

// --- legacy Group (message link) handlers ---

func (gc *GroupController) GetGroups(c *gin.Context) {
	ctx := c.Request.Context()
	groups, err := model.GetGroups(ctx, gc.Controller.DI.DBDecorator.GDB())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, groups)
}

func (gc *GroupController) GetGroup(c *gin.Context) {
	var id int
	if _, err := fmtSscanf(c.Param("id"), &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	g, err := model.GetGroupById(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, g)
}

func (gc *GroupController) CreateGroup(c *gin.Context) {
	var g model.Group
	if err := c.ShouldBindJSON(&g); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}
	if err := model.CreateGroup(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), &g); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create group: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, g)
}

func (gc *GroupController) UpdateGroup(c *gin.Context) {
	var id int
	if _, err := fmtSscanf(c.Param("id"), &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var g model.Group
	if err := c.ShouldBindJSON(&g); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}
	err := model.UpdateGroupById(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), &g, id)
	if err != nil {
		if errors.Is(err, model.GroupNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update group: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, g)
}

func (gc *GroupController) DeleteGroup(c *gin.Context) {
	var id int
	if _, err := fmtSscanf(c.Param("id"), &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := model.DeleteGroupById(c.Request.Context(), gc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		if errors.Is(err, model.GroupNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete group: " + err.Error()})
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func fmtSscanf(s string, id *int) (int, error) {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, errors.New("invalid id")
		}
		n = n*10 + int(ch-'0')
	}
	if s == "" {
		return 0, errors.New("invalid id")
	}
	*id = n
	return 1, nil
}
