package message

import (
	"chickchirick-messages/internal/controller/c_controller"
	file "chickchirick-messages/internal/model/message"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type FileController struct {
	Controller c_controller.Controller
}

func (fc *FileController) RegisterRoutes() {
	e := fc.Controller.E

	e.GET("/files", fc.GetFiles)
	e.POST("/file", fc.CreateFile)
	e.GET("/file/:id", fc.GetFile)
	e.PUT("/file/:id", fc.UpdateFile)
	e.DELETE("/file/:id", fc.DeleteFile)
}

func (fc *FileController) GetFiles(c *gin.Context) {
	ctx := c.Request.Context()
	files, err := file.GetFiles(ctx, fc.Controller.DI.DBDecorator.GDB())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, files)
}

func (fc *FileController) GetFile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	f, err := file.GetFileById(ctx, fc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, f)
}

func (fc *FileController) CreateFile(c *gin.Context) {
	var f file.File
	if err := c.ShouldBindJSON(&f); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := file.CreateFile(ctx, fc.Controller.DI.DBDecorator.GDB(), &f); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create file: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, f)
}

func (fc *FileController) UpdateFile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var f file.File
	if err := c.ShouldBindJSON(&f); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	err = file.UpdateFileById(ctx, fc.Controller.DI.DBDecorator.GDB(), &f, id)
	if err != nil {
		if errors.Is(err, file.FileNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update file: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, f)
}

func (fc *FileController) DeleteFile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	err = file.DeleteFileById(ctx, fc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		if errors.Is(err, file.FileNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete file: " + err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}
