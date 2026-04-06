package message

import (
	"chickchirick-messages/internal/controller/c_controller"
	userRelation "chickchirick-messages/internal/model/message"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserRelationController struct {
	Controller c_controller.Controller
}

func (urc *UserRelationController) RegisterRoutes() {
	e := urc.Controller.E

	e.GET("/user-relations", urc.GetUserRelations)
	e.POST("/user-relation", urc.CreateUserRelation)
	e.GET("/user-relation/:id", urc.GetUserRelation)
	e.PUT("/user-relation/:id", urc.UpdateUserRelation)
	e.DELETE("/user-relation/:id", urc.DeleteUserRelation)
}

func (urc *UserRelationController) GetUserRelations(c *gin.Context) {
	userRelations, err := userRelation.GetUserRelation(urc.Controller.DI.DBDecorator.GDB())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, userRelations)
}

func (urc *UserRelationController) GetUserRelation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ur, err := userRelation.GetUserRelationById(urc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user relation not found: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, ur)
}

func (urc *UserRelationController) CreateUserRelation(c *gin.Context) {
	var ur userRelation.UserRelation
	if err := c.ShouldBindJSON(&ur); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}

	if err := userRelation.CreateUserRelation(urc.Controller.DI.DBDecorator.GDB(), &ur); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user relation: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ur)
}

func (urc *UserRelationController) UpdateUserRelation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var ur userRelation.UserRelation
	if err := c.ShouldBindJSON(&ur); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	err = userRelation.UpdateUserRelationById(urc.Controller.DI.DBDecorator.GDB(), &ur, id)
	if err != nil {
		if errors.Is(err, userRelation.UserRelationNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user relation: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, ur)
}

func (urc *UserRelationController) DeleteUserRelation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = userRelation.DeleteUserRelationById(urc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		if errors.Is(err, userRelation.UserRelationNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user relation: " + err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}
