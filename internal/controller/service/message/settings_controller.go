package message

import (
	"chickchirick-messages/internal/controller/c_controller"
	settings "chickchirick-messages/internal/model/message"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SettingsController struct {
	Controller c_controller.Controller
}

func (sc *SettingsController) RegisterRoutes() {
	e := sc.Controller.E

	e.GET("/settings", sc.GetSettings)
	e.POST("/setting", sc.CreateSetting)
	e.GET("/setting/:id", sc.GetSetting)
	e.PUT("/setting/:id", sc.UpdateSetting)
	e.DELETE("/setting/:id", sc.DeleteSetting)
}

func (sc *SettingsController) GetSettings(c *gin.Context) {
	ctx := c.Request.Context()
	settingList, err := settings.GetSettings(ctx, sc.Controller.DI.DBDecorator.GDB())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, settingList)
}

func (sc *SettingsController) GetSetting(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	s, err := settings.GetSettingsById(ctx, sc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Setting not found: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, s)
}

func (sc *SettingsController) CreateSetting(c *gin.Context) {
	var s settings.Settings
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := settings.CreateSettings(ctx, sc.Controller.DI.DBDecorator.GDB(), &s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create setting: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, s)
}

func (sc *SettingsController) UpdateSetting(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var s settings.Settings
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	err = settings.UpdateSettingsById(ctx, sc.Controller.DI.DBDecorator.GDB(), &s, id)
	if err != nil {
		if errors.Is(err, settings.StatusNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update setting: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, s)
}

func (sc *SettingsController) DeleteSetting(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	err = settings.DeleteSettingsById(ctx, sc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete setting: " + err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
