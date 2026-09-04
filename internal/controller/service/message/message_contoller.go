package message

import (
	"chickchirick-messages/internal/controller/c_controller"
	"chickchirick-messages/internal/middleware"
	"chickchirick-messages/internal/model/message"
	"chickchirick-messages/internal/service/history"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MessagesController struct {
	Controller c_controller.Controller
}

func (mc *MessagesController) RegisterRoutes() {
	e := mc.Controller.E

	e.GET("/messages", mc.GetMessages)
	e.GET("/messages/history", middleware.Auth(mc.Controller.DI.AuthClient), mc.GetHistory)
	e.POST("/message", mc.CreateMessage)
	e.GET("/message/:id", mc.GetMessage)
	e.PUT("/message/:id", mc.UpdateMessage)
	e.DELETE("/message/:id", mc.DeleteMessage)
}

func (mc *MessagesController) GetHistory(c *gin.Context) {
	userUUIDValue, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user is not authenticated"})
		return
	}

	userUUID, ok := userUUIDValue.(string)
	if !ok || userUUID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authenticated user"})
		return
	}

	result, err := history.GetHistory(c.Request.Context(), mc.Controller.DI.DBDecorator.GDB(), userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (mc *MessagesController) GetMessages(c *gin.Context) {
	ctx := c.Request.Context()
	messages, err := message.GetMessages(ctx, mc.Controller.DI.DBDecorator.GDB())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

func (mc *MessagesController) GetMessage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	m, err := message.GetMessageById(ctx, mc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "message not found: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, m)
}

func (mc *MessagesController) CreateMessage(c *gin.Context) {
	var m message.Message

	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := message.CreateMessage(ctx, mc.Controller.DI.DBDecorator.GDB(), &m); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create message: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, m)
}

func (mc *MessagesController) UpdateMessage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var m message.Message
	if err = c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	err = message.UpdateMessageById(ctx, mc.Controller.DI.DBDecorator.GDB(), &m, id)
	if err != nil {
		if errors.Is(err, message.MessageNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update message: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, m)
}

func (mc *MessagesController) DeleteMessage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	err = message.DeleteMessageById(ctx, mc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		if errors.Is(err, message.MessageNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete message: " + err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}
