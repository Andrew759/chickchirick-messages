package factory

import (
	"chickchirick-messages/internal/controller/c_controller"
	internalService "chickchirick-messages/internal/controller/service/message"

	"github.com/gin-gonic/gin"
)

type MessageServer struct{}

func InitMessageServer(e *gin.Engine, mDIC *c_controller.DIContainer) {
	messageServer := &MessageServer{}

	messageServer.initMessageService(e, mDIC)
	messageServer.initDeletedService(e, mDIC)
	messageServer.initFileService(e, mDIC)
	messageServer.initGroupService(e, mDIC)
	messageServer.initMessageMetaService(e, mDIC)
	messageServer.initPersonalService(e, mDIC)
	messageServer.initSettingsService(e, mDIC)
	messageServer.initStatusService(e, mDIC)
	messageServer.initUserRelationService(e, mDIC)
}

func (ms MessageServer) initMessageService(e *gin.Engine, aDIC *c_controller.DIContainer) internalService.MessagesController {
	messageService := internalService.MessagesController{
		Controller: c_controller.Controller{
			E:  e,
			DI: aDIC,
		},
	}
	messageService.RegisterRoutes()

	return messageService
}

func (ms MessageServer) initDeletedService(e *gin.Engine, aDIC *c_controller.DIContainer) internalService.DeletedController {
	deletedService := internalService.DeletedController{
		Controller: c_controller.Controller{
			E:  e,
			DI: aDIC,
		},
	}
	deletedService.RegisterRoutes()

	return deletedService
}

func (ms MessageServer) initFileService(e *gin.Engine, aDIC *c_controller.DIContainer) internalService.FileController {
	fileService := internalService.FileController{
		Controller: c_controller.Controller{
			E:  e,
			DI: aDIC,
		},
	}
	fileService.RegisterRoutes()

	return fileService
}

func (ms MessageServer) initGroupService(e *gin.Engine, aDIC *c_controller.DIContainer) internalService.GroupController {
	groupService := internalService.GroupController{
		Controller: c_controller.Controller{
			E:  e,
			DI: aDIC,
		},
	}
	groupService.RegisterRoutes()

	return groupService
}

func (ms MessageServer) initMessageMetaService(e *gin.Engine, aDIC *c_controller.DIContainer) internalService.MetaController {
	metaService := internalService.MetaController{
		Controller: c_controller.Controller{
			E:  e,
			DI: aDIC,
		},
	}
	metaService.RegisterRoutes()

	return metaService
}

func (ms MessageServer) initPersonalService(e *gin.Engine, aDIC *c_controller.DIContainer) internalService.PersonalController {
	personalService := internalService.PersonalController{
		Controller: c_controller.Controller{
			E:  e,
			DI: aDIC,
		},
	}
	personalService.RegisterRoutes()

	return personalService
}

func (ms MessageServer) initSettingsService(e *gin.Engine, aDIC *c_controller.DIContainer) internalService.SettingsController {
	settingsService := internalService.SettingsController{
		Controller: c_controller.Controller{
			E:  e,
			DI: aDIC,
		},
	}
	settingsService.RegisterRoutes()

	return settingsService
}

func (ms MessageServer) initStatusService(e *gin.Engine, aDIC *c_controller.DIContainer) internalService.StatusController {
	statusService := internalService.StatusController{
		Controller: c_controller.Controller{
			E:  e,
			DI: aDIC,
		},
	}
	statusService.RegisterRoutes()

	return statusService
}

func (ms MessageServer) initUserRelationService(e *gin.Engine, aDIC *c_controller.DIContainer) internalService.UserRelationController {
	userRelationService := internalService.UserRelationController{
		Controller: c_controller.Controller{
			E:  e,
			DI: aDIC,
		},
	}
	userRelationService.RegisterRoutes()

	return userRelationService
}
