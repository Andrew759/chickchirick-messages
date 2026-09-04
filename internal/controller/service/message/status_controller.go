package message

import (
	"chickchirick-messages/internal/controller/c_controller"
	status "chickchirick-messages/internal/model/message"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type StatusController struct {
	Controller c_controller.Controller
}

func (sc *StatusController) RegisterRoutes() {
	e := sc.Controller.E

	e.GET("/statuses", sc.GetStatuses)
	e.POST("/status", sc.CreateStatus)
	e.GET("/status/:id", sc.GetStatus)
	e.PUT("/status/:id", sc.UpdateStatus)
	e.DELETE("/status/:id", sc.DeleteStatus)
}

func (sc *StatusController) GetStatuses(c *gin.Context) {
	ctx := c.Request.Context()
	statuses, err := status.GetStatus(ctx, sc.Controller.DI.DBDecorator.GDB())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, statuses)
}

func (sc *StatusController) GetStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	s, err := status.GetStatusById(ctx, sc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "status not found: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, s)
}

func (sc *StatusController) CreateStatus(c *gin.Context) {
	var s status.Status
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := status.CreateStatus(ctx, sc.Controller.DI.DBDecorator.GDB(), &s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create status: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, s)
}

func (sc *StatusController) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var s status.Status
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	err = status.UpdateStatusById(ctx, sc.Controller.DI.DBDecorator.GDB(), &s, id)
	if err != nil {
		if errors.Is(err, status.StatusNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, s)
}

func (sc *StatusController) DeleteStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	err = status.DeleteStatusById(ctx, sc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		if errors.Is(err, status.StatusNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete status: " + err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}
