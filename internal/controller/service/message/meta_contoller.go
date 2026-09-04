package message

import (
	"chickchirick-messages/internal/controller/c_controller"
	meta "chickchirick-messages/internal/model/message"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MetaController struct {
	Controller c_controller.Controller
}

func (mc *MetaController) RegisterRoutes() {
	e := mc.Controller.E

	e.GET("/metas", mc.GetMetas)
	e.POST("/meta", mc.CreateMeta)
	e.GET("/meta/:id", mc.GetMeta)
	e.PUT("/meta/:id", mc.UpdateMeta)
	e.DELETE("/meta/:id", mc.DeleteMeta)
}

func (mc *MetaController) GetMetas(c *gin.Context) {
	ctx := c.Request.Context()
	metas, err := meta.GetMetas(ctx, mc.Controller.DI.DBDecorator.GDB())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metas)
}

func (mc *MetaController) GetMeta(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	m, err := meta.GetMetaById(ctx, mc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "meta not found: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, m)
}

func (mc *MetaController) CreateMeta(c *gin.Context) {
	var m meta.Meta
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := meta.CreateMeta(ctx, mc.Controller.DI.DBDecorator.GDB(), &m); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create meta: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, m)
}

func (mc *MetaController) UpdateMeta(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var m meta.Meta
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	err = meta.UpdateMetaById(ctx, mc.Controller.DI.DBDecorator.GDB(), &m, id)
	if err != nil {
		if errors.Is(err, meta.MetaNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update meta: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, m)
}

func (mc *MetaController) DeleteMeta(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	err = meta.DeleteMetaById(ctx, mc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		if errors.Is(err, meta.MetaNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete meta: " + err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}
