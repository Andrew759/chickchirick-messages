package message

import (
	"chickchirick-messages/internal/controller/c_controller"
	group "chickchirick-messages/internal/model/message"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type GroupController struct {
	Controller c_controller.Controller
}

func (gc *GroupController) RegisterRoutes() {
	e := gc.Controller.E

	e.GET("/groups", gc.GetGroups)
	e.POST("/group", gc.CreateGroup)
	e.GET("/group/:id", gc.GetGroup)
	e.PUT("/group/:id", gc.UpdateGroup)
	e.DELETE("/group/:id", gc.DeleteGroup)
}

func (gc *GroupController) GetGroups(c *gin.Context) {
	ctx := c.Request.Context()
	groups, err := group.GetGroups(ctx, gc.Controller.DI.DBDecorator.GDB())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, groups)
}

func (gc *GroupController) GetGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	g, err := group.GetGroupById(ctx, gc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, g)
}

func (gc *GroupController) CreateGroup(c *gin.Context) {
	var g group.Group
	if err := c.ShouldBindJSON(&g); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := group.CreateGroup(ctx, gc.Controller.DI.DBDecorator.GDB(), &g); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create group: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, g)
}

func (gc *GroupController) UpdateGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var g group.Group
	if err := c.ShouldBindJSON(&g); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	err = group.UpdateGroupById(ctx, gc.Controller.DI.DBDecorator.GDB(), &g, id)
	if err != nil {
		if errors.Is(err, group.GroupNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update group: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, g)
}

func (gc *GroupController) DeleteGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	err = group.DeleteGroupById(ctx, gc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		if errors.Is(err, group.GroupNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete group: " + err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}
