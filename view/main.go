package main

import (
	"advisorProject/config"
	"advisorProject/controller"
	"advisorProject/middleware"
	"advisorProject/model"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/robfig/cron/v3"
	"log"
)

// this way to create *model.User and Adv will get a nil pointer, very dangerous
// var user *model.User
// var Adv *model.Advisor

var user = &model.User{}
var advisor = &model.Advisor{}
var route = controller.Server

func main() {
	defer config.DB.Close()
	//user route
	usr := route.Group("/user")
	{
		usr.POST("/register", user.Register)
		usr.POST("/login", user.Login)

	}
	//use a middleware to check whether the token right or not
	v := route.Group("user/auth")
	//use a middleware to check token validity if user want to update personal information
	v.Use(middleware.JwtUserAuth())
	{
		v.POST("/update", user.Update)
		v.POST("/view", user.ViewAdvisorList)
		v.POST("/visit/:id", user.Visit)
		v.POST("/create/:type", middleware.SaveAdvisor(model.Ad), user.Create)
		//v.POST("/orderlist", middleware.CheckExpiration(), middleware.ReturnExpediteCoins(), middleware.ReturnCoins(), user.ViewOrderList)
		//v.POST("/expedite", middleware.CheckExpiration(), middleware.ReturnExpediteCoins(), middleware.ReturnCoins(), user.Expedite)
		//v.POST("/expedite/:id", middleware.CheckExpiration(), middleware.ReturnExpediteCoins(), middleware.ReturnCoins(), user.ExpediteID)
		//v.POST("/orderinfo", middleware.CheckExpiration(), middleware.ReturnExpediteCoins(), middleware.ReturnCoins(), user.OrderInfo)
		//v.POST("/orderinfo/:id", middleware.CheckExpiration(), middleware.ReturnExpediteCoins(), middleware.ReturnCoins(), user.CheckOrderInfo)
		v.POST("/orderlist", user.ViewOrderList)
		v.POST("/expedite", user.Expedite)
		v.POST("/expedite/:id", user.ExpediteID)
		v.POST("/orderinfo", user.OrderInfo)
		v.POST("/orderinfo/:id", user.CheckOrderInfo)
		v.POST("/rate/:id", user.RateReviewAndTip)
		v.POST("/collect/:id", user.CollectAd)
		v.POST("/collectlist", user.CollectList)
		v.POST("/transactions", user.TransactionsDetails)
		v.POST("cancel/:id", user.Cancel)
	}
	//advisor route
	adv := route.Group("/advisor")
	{
		adv.POST("/register", advisor.Register)
		adv.POST("/login", advisor.Login)

	}
	v1 := route.Group("advisor/auth")
	v1.Use(middleware.JwtAdvisorAuth())
	{
		v1.POST("/update", advisor.Update)
		v1.POST("/setprice", advisor.DisplayService)
		v1.POST("/setprice/:id", advisor.SetPrice)
		//v1.POST("/vieworders", middleware.CheckExpiration(), middleware.ReturnExpediteCoins(), middleware.ReturnCoins(), advisor.ViewOrders)
		//v1.POST("/vieworders/:id", middleware.CheckExpiration(), middleware.ReturnExpediteCoins(), middleware.ReturnCoins(), advisor.SpecifiedInfo)
		//v1.POST("/reply/:id", middleware.CheckExpiration(), middleware.ReturnExpediteCoins(), middleware.ReturnCoins(), advisor.ReplyOrder)
		v1.POST("/vieworders", advisor.ViewOrders)
		v1.POST("/vieworders/:id", advisor.SpecifiedInfo)
		v1.POST("/reply/:id", advisor.ReplyOrder)
		v1.POST("/transactions", advisor.TransactionsDetails)
	}
	model.Rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis 地址
		Password: "",               // 密码 (默认为空)
		DB:       0,                // 使用默认 DB
	})
	defer model.Rdb.Close()
	// 测试连接
	pong, err := model.Rdb.Ping(model.Ctx).Result()
	if err != nil {
		panic("redis connect failed")
	}
	fmt.Println(pong, ",redis server connection established")
	c := cron.New(cron.WithSeconds())
	_, err = c.AddFunc("* * * * * *", func() {
		model.CronJob()
	})
	if err != nil {
		log.Println(err.Error())
	}
	//_, err = c.AddFunc("* * * * *", func() {
	//	model.Cache(model.Rdb)
	//})
	//if err != nil {
	//	log.Println(err.Error())
	//}
	c.Start()

	//controller.UserInfoGet()
	controller.Server.Run(":8080")

}
