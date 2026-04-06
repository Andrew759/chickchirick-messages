package message

import (
	"chickchirick-messages/internal/controller/c_controller"
	personal "chickchirick-messages/internal/model/message"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PersonalController struct {
	Controller c_controller.Controller
}

func (pc *PersonalController) RegisterRoutes() {
	e := pc.Controller.E

	e.GET("/personals", pc.GetPersonals)
	e.POST("/personal", pc.CreatePersonal)
	e.GET("/personal/:id", pc.GetPersonal)
	e.PUT("/personal/:id", pc.UpdatePersonal)
	e.DELETE("/personal/:id", pc.DeletePersonal)
}

func (pc *PersonalController) GetPersonals(c *gin.Context) {
	personals, err := personal.GetPersonal(pc.Controller.DI.DBDecorator.GDB())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, personals)
}

func (pc *PersonalController) GetPersonal(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p, err := personal.GetPersonalById(pc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "personal not found: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, p)
}

func (pc *PersonalController) CreatePersonal(c *gin.Context) {
	var p personal.Personal
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}

	if err := personal.CreatePersonal(pc.Controller.DI.DBDecorator.GDB(), &p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create personal: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, p)
}

func (pc *PersonalController) UpdatePersonal(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var p personal.Personal
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}

	err = personal.UpdatePersonalById(pc.Controller.DI.DBDecorator.GDB(), &p, id)
	if err != nil {
		if errors.Is(err, personal.PersonalNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update personal: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, p)
}

func (pc *PersonalController) DeletePersonal(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = personal.DeletePersonalById(pc.Controller.DI.DBDecorator.GDB(), id)
	if err != nil {
		if errors.Is(err, personal.PersonalNotFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete personal: " + err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}
