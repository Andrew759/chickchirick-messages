package message

import (
	"chickchirick-messages/internal/controller/c_controller"
	deleted "chickchirick-messages/internal/model/message"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type DeletedController struct {
	Controller c_controller.Controller
}

func (dc *DeletedController) RegisterRoutes() {
	e := dc.Controller.E

	e.GET("/deleted-list", dc.GetDeletedList)
	e.POST("/deleted", dc.CreateDeleted)
	e.GET("/deleted/:id", dc.GetDeleted)
	e.PUT("/deleted/:id", dc.UpdateDeleted)
	e.DELETE("/deleted/:id", dc.DeleteDeleted)
}

func (dc *DeletedController) GetDeletedList(c *gin.Context) {
	deletedList, err := deleted.GetDeleted(dc.Controller.DI.DBDecorator.GDB())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, deletedList)
}

func (dc *DeletedController) GetDeleted(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	d, err := deleted.GetDeletedById(dc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "deleted not found: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, d)
}

func (dc *DeletedController) CreateDeleted(c *gin.Context) {
	var d deleted.Deleted
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}

	if err := deleted.CreateDeleted(dc.Controller.DI.DBDecorator.GDB(), &d); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create deleted: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, d)
}

func (dc *DeletedController) UpdateDeleted(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var d deleted.Deleted
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}

	err = deleted.UpdateDeletedById(dc.Controller.DI.DBDecorator.GDB(), &d, id)
	if err != nil {
		if errors.Is(err, deleted.DeletedNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update deleted: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, d)
}

func (dc *DeletedController) DeleteDeleted(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = deleted.DeleteDeletedById(dc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		if errors.Is(err, deleted.DeletedNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete deleted: " + err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}
