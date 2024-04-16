package controller

//
import (
	"advisorProject/model"
	"github.com/gin-gonic/gin"
)

// define a gin engine
var Server = gin.Default()
var user *model.User

// post request
func UserRegisterPost() {
	Server.POST("/register", user.Register)
}
func UserUpdatePost() {
	Server.POST("/update", user.Update)
}
func UserLoginPost() {
	Server.POST("/login", user.Login)
}

//func UserInfoGet() {
//	Server.GET("/user/protected", middleware.ProtectedRoute)
//}
