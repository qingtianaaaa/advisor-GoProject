package model

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/didi/gendry/builder"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"math"
	"net/http"
	"strconv"
	"time"
)

type InService struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Bio  string `json:"bio"`
}
type OrderDisplay struct {
	Id            string
	Type          string
	BasicInfo     string
	GeneralSitu   string
	SpecifiedQues string
	Cost          string
	IsExpired     string
	Expedite      string
}
type OrderOverview struct {
	OrderId       string
	Type          string
	IsCompleted   string
	CheckInfo     string
	RateAndReview string
}
type OrderConcreteInfo struct {
	OrderId       string
	Type          string
	IsCompleted   string
	OrderTime     string
	DeliveryTime  string
	Name          string
	DateOfBirth   string
	Gender        string
	GeneralSitu   string
	SpecifiedQues string
	Reply         string
}

func (u *User) ViewAdvisorList(c *gin.Context) {
	advisors := &[]InService{} // creat a slice to store advisors who are in service
	err := selectAdvisor(advisors)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(*advisors) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sorry!There are no advisors in service"})
		return
	}
	//not displaying id
	filter := []gin.H{}
	for _, adv := range *advisors {
		filterAdv := gin.H{
			"Name":    adv.Name,
			"Tag":     adv.Bio,
			"Consult": "http://localhost:8080/user/auth/visit/" + strconv.Itoa(adv.ID),
		}
		filter = append(filter, filterAdv)
	}
	c.JSON(http.StatusOK, filter)

}

func selectAdvisor(ads *[]InService) error {
	where := map[string]interface{}{
		"status": "in-service",
	}
	var inservice InService
	selectField := []string{"name", "bio", "id"}
	cond, val, err := builder.BuildSelect("advisor", where, selectField)
	if err != nil {
		return err
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return err
	}
	for rows.Next() {
		err = rows.Scan(&inservice.Name, &inservice.Bio, &inservice.ID)
		if err != nil {
			return err
		}
		*ads = append(*ads, inservice)
	}
	return nil
}

func (u *User) Expedite(c *gin.Context) {
	orderIdSlice := []string{}
	orderId := ""
	var orders []OrderDisplay
	order := OrderDisplay{}
	status := ""

	cond, val, err := builder.BuildSelect("orders", map[string]interface{}{"user": u.Phone}, []string{"id"})
	if err != nil {

		c.JSON(401, gin.H{"error": "build select failed"})
		return
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		c.JSON(401, gin.H{"error": "database query failed"})
		return
	}
	for rows.Next() {
		err = rows.Scan(&orderId)
		if err != nil {
			c.JSON(401, gin.H{"error": "rows scan failed"})
			return
		}
		orderIdSlice = append(orderIdSlice, orderId)
	}
	if len(orderIdSlice) == 0 {
		c.JSON(401, gin.H{"error": "there are no orders currently"})
		return
	}
	where := map[string]interface{}{
		"id in": orderIdSlice,
	}
	selectField := []string{"id", "type", "basic_info", "general_situ", "specified_ques", "cost"}
	cond, val, err = builder.BuildSelect("orders_info", where, selectField)
	if err != nil {
		c.JSON(401, gin.H{"error": "build select failed"})
		return
	}
	rows, err = db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		fmt.Println(err.Error())
		c.JSON(401, gin.H{"error": "database query failed--1--expedite"})
		return
	}
	for rows.Next() {
		err = rows.Scan(&order.Id, &order.Type, &order.BasicInfo, &order.GeneralSitu, &order.SpecifiedQues, &order.Cost)
		if err != nil {
			c.JSON(401, gin.H{"error": "rows scan failed"})
			return
		}
		cond, val, err = builder.BuildSelect("orders", map[string]interface{}{"id": order.Id}, []string{"is_completed"})
		if err != nil {
			c.JSON(401, gin.H{"error": "build select failed"})
			return
		}
		rows1, err := db.Query(cond, val...)
		if err != nil {
			c.JSON(401, gin.H{"error": "database query failed--2--expedite"})
			return
		}
		if rows1.Next() {
			err = rows1.Scan(&status)
			if err != nil {
				c.JSON(401, gin.H{"error": "rows scan failed"})
				return
			}
			if status == "Completed" || status == "completed" {
				order.IsExpired = "Completed"
				order.Expedite = "this order is Completed"
			}
			if status == "expired" || status == "Expired" || status == "expired (refunded)" {
				order.IsExpired = "expired"
				order.Expedite = "this order is expired"
			}
			if status == "pending" {
				order.IsExpired = "pending"
				order.Expedite = "http://localhost:8080/user/auth/expedite/" + order.Id
			}
		}
		orders = append(orders, order)
	}
	if len(orders) == 0 {
		c.JSON(401, gin.H{"error": "you have no orders"})
		return
	}
	c.JSON(200, gin.H{"orders": orders})
}

func (u *User) ExpediteID(c *gin.Context) {
	tx, err := db.Begin()
	if err != nil {
		c.JSON(401, gin.H{"error": "init tx failed"})
		return
	}
	cost := 0.0
	id := c.Param("id")
	err = isCompleted(id, u.Phone)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	err = hasBeenExpedited(id)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	remains, err := isSufficient(id, u.Phone, &cost)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}

	currentTime := time.Now()
	layout := "2006-01-02 15:04:05"
	exactTime := currentTime.Format(layout)
	where := map[string]interface{}{"id": id}
	cond, val, err := builder.BuildUpdate("orders", where, map[string]interface{}{"expedite": exactTime})
	if err != nil {
		c.JSON(401, gin.H{"error": "build update failed"})
		return
	}
	_, err = tx.Exec(cond, val...)
	if err != nil {
		err1 := tx.Rollback()
		if err1 != nil {
			c.JSON(401, gin.H{"error": err1.Error()})
			return
		}
		c.JSON(401, gin.H{"error": "database execute failed"})
		return
	}

	data := map[string]interface{}{
		"phone":   u.Phone,
		"type":    "Speed Up Order",
		"credits": "-" + fmt.Sprintf("%.1f", cost/2),
		"time":    exactTime,
	}
	cond, val, err = builder.BuildInsert("transactions", []map[string]interface{}{data})
	if err != nil {
		c.JSON(401, gin.H{"error": "build insert failed"})
		return
	}
	_, err = tx.Exec(cond, val...)
	if err != nil {
		err1 := tx.Rollback()
		if err1 != nil {
			c.JSON(401, gin.H{"error": err1.Error()})
			return
		}
		c.JSON(401, gin.H{"error": "database execute failed"})
		return
	}
	trct := fmt.Sprintf("%.1f", remains)
	cond, val, err = builder.BuildUpdate("user", map[string]interface{}{"phone": u.Phone}, map[string]interface{}{"coin": trct})
	if err != nil {
		c.JSON(401, gin.H{"error": "build update failed"})
		return
	}
	_, err = tx.Exec(cond, val...)
	if err != nil {
		err1 := tx.Rollback()
		if err1 != nil {
			c.JSON(401, gin.H{"error": err1.Error()})
			return
		}
		c.JSON(401, gin.H{"error": "database execute failed"})
		return
	}
	//err = deductCoins(u.Phone, remains)
	//if err != nil {
	//	c.JSON(401, gin.H{"error": err.Error()})
	//	return
	//}
	c.JSON(200, gin.H{"message": "expedite successfully", "remain": remains})
}

func hasBeenExpedited(id string) error {
	var expedite sql.NullString
	cond, val, err := builder.BuildSelect("orders", map[string]interface{}{"id": id}, []string{"expedite"})
	if err != nil {
		return errors.New("build select failed--hasbeenexpedite")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return errors.New("database query failed--hasbeenexpedite")
	}
	if rows.Next() {
		err = rows.Scan(&expedite)
		if err != nil {
			return errors.New("rows scan failed--hasbeenexpedite")
		}
		if expedite.Valid {
			//err = hasBeenExpired(expedite.String, id)
			//if err != nil {
			//	return err
			//}
			return errors.New("this order has been expedited")
		}
	} else {
		return errors.New("order id doesn't exist")
	}
	return nil
}

//
//func hasBeenExpired(expediteTime, id string) error {
//	currentTime := time.Now()
//	currentFormat := currentTime.Format(time.RFC3339)
//	current, err := time.Parse(time.RFC3339, currentFormat)
//	if err != nil {
//		return errors.New("time parse failed")
//	}
//	expedite, err := time.Parse(time.RFC3339, expediteTime)
//	if err != nil {
//		return errors.New("time parse failed")
//	}
//	duration := current.Sub(expedite)
//	if duration >= 3*time.Minute {
//		err = hasBeenReplied(id)
//		if err != nil {
//			return err
//		}
//		return nil
//	}
//	return errors.New("this order has been expedited")
//}

//func hasBeenReplied(id string) error {
//	isFinished := ""
//	var user, cost string
//	where := map[string]interface{}{"id": id}
//	selectField := []string{"is_completed"}
//	cond, val, err := builder.BuildSelect("orders", where, selectField)
//	if err != nil {
//		return errors.New("build select failed--hasbeenreplied")
//	}
//	rows, err := db.Query(cond, val...)
//	defer rows.Close()
//	if err != nil {
//		return errors.New("database query failed--hasbeenreplied")
//	}
//	if rows.Next() {
//		err = rows.Scan(&isFinished)
//		if err != nil {
//			return errors.New("rows scan failed")
//		}
//		if isFinished == "completed" || isFinished == "Completed" {
//			return errors.New("this order has been completed")
//		}
//		if isFinished == "expired" || isFinished == "Expired" {
//			return errors.New("this order has been expired")
//		}
//		if isFinished == "pending" {
//			cond, val, err = builder.BuildSelect("orders_info", map[string]interface{}{"id": id}, []string{"user", "cost"})
//			if err != nil {
//				return errors.New("build select failed--hasbeenreplied")
//			}
//			rows1, err := db.Query(cond, val...)
//			if err != nil {
//				return errors.New("database query failed--hasbeenreplied")
//			}
//			if rows1.Next() {
//				err = rows1.Scan(&user, &cost)
//				if err != nil {
//					return errors.New("rows1 scan failed--hasbeenreplied")
//				}
//				err = returnCoinsToUser(user, cost)
//				if err != nil {
//					return err
//				}
//			}
//		}
//	}
//	return nil
//}

func returnCoinsToUser(user string, cost string) error {
	coin := ""
	cond, val, err := builder.BuildSelect("user", map[string]interface{}{"phone": user}, []string{"coin"})
	if err != nil {
		return errors.New("build select failed--returncoinstouser")
	}
	rows, err := db.Query(cond, val...)
	if err != nil {
		return errors.New("database query failed--returncoinstouser")
	}
	if rows.Next() {
		err = rows.Scan(&coin)
		if err != nil {
			return errors.New("rows scan failed--returncointouser")
		}
	}
	userBalance, err := strconv.ParseFloat(coin, 64)
	if err != nil {
		return errors.New("coin parse failed--returncointouser")
	}
	orderCost, err := strconv.ParseFloat(cost, 64)
	if err != nil {
		return errors.New("cost parse failed--returncointouser")
	}
	expediteCost := math.Round(orderCost/2*10) / 10
	returnCost := userBalance + expediteCost
	cond, val, err = builder.BuildUpdate("user", map[string]interface{}{"phone": user}, map[string]interface{}{"coin": returnCost})
	if err != nil {
		return errors.New("build update failed--returncoinstouser")
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return errors.New("database execute failed--returncoinstouser")
	}
	return nil
}

func isCompleted(id, phone string) error {
	var user, isFinished string
	where := map[string]interface{}{"id": id}
	selectField := []string{"user", "is_completed"}
	cond, val, err := builder.BuildSelect("orders", where, selectField)
	if err != nil {
		return errors.New("build select failed")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return errors.New("database query failed--iscompleted")
	}
	if rows.Next() {
		err = rows.Scan(&user, &isFinished)
		if err != nil {
			return errors.New("rows scan failed")
		}
		if user != phone {
			return errors.New("this is not your order")
		}
		if isFinished == "completed" || isFinished == "Completed" {
			return errors.New("this order is finished")
		}
		if isFinished == "expired" || isFinished == "expired (refunded)" {
			return errors.New("this order is expired")
		}
	} else {
		return errors.New("order id doesn't exist")
	}
	return nil
}

func isSufficient(id, phone string, expediteCost *float64) (float64, error) {
	user := ""
	userBalance := 0.0
	cond, val, err := builder.BuildSelect("orders", map[string]interface{}{"id": id}, []string{"user"})
	if err != nil {
		return 0, errors.New("build select failed")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return 0, errors.New("database query failed")
	}
	if rows.Next() {
		err = rows.Scan(&user)
		if err != nil {
			fmt.Println(err.Error())
			return 0, errors.New("rows scan failed")
		}
	}
	where := map[string]interface{}{"id": id}
	selectField := []string{"cost"}
	cond, val, err = builder.BuildSelect("orders_info", where, selectField)
	if err != nil {
		return 0, errors.New("build select failed")
	}
	rows, err = db.Query(cond, val...)
	if err != nil {
		return 0, errors.New("database query failed--issufficient")
	}

	if rows.Next() {
		err = rows.Scan(expediteCost)
		if err != nil {
			return 0, errors.New("rows scan failed")
		}
	} else {
		return 0, errors.New("order id doesn't exist")
	}
	if phone != user {
		return 0, errors.New("this is not your order")
	}
	cond, val, err = builder.BuildSelect("user", map[string]interface{}{"phone": phone}, []string{"coin"})
	if err != nil {
		return 0, errors.New("build select failed")
	}
	rows, err = db.Query(cond, val...)
	if err != nil {
		return 0, errors.New("database query coin failed")
	}
	if rows.Next() {
		err = rows.Scan(&userBalance)
		if err != nil {
			return 0, errors.New("rows scan failed")
		}
	} else {
		return 0, errors.New("user doesn't exist")
	}

	x := math.Round(*expediteCost/2*10) / 10
	y := math.Round(userBalance*10) / 10
	if y >= x {
		return y - x, nil
	}
	return 0, errors.New("expedite failed.balance is not enough")
}

func (u *User) OrderInfo(c *gin.Context) {
	orders := []OrderOverview{}
	err := displayOrder(u.Phone, &orders)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	if len(orders) == 0 {
		c.JSON(401, gin.H{"error": "there are no orders completed or expired"})
		return
	}
	c.JSON(200, gin.H{"orders": orders})
}
func displayOrder(phone string, orders *[]OrderOverview) error {

	//order := &OrderOverview{} //zc
	order := OrderOverview{} //zc
	//var order OrderOverview //zc

	cond, val, err := builder.BuildSelect("orders", map[string]interface{}{"user": phone}, []string{"id", "type", "is_completed"})
	if err != nil {
		return errors.New("build select failed")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return errors.New("database query failed--displayorder")
	}
	for rows.Next() {
		//var order OrderOverview

		err = rows.Scan(&order.OrderId, &order.Type, &order.IsCompleted)
		if err != nil {
			return errors.New("rows scan failed")
		}
		if order.IsCompleted == "pending" {
			continue
		}
		if order.IsCompleted == "completed" {
			cond, val, err = builder.BuildSelect("answer", map[string]interface{}{"order_id": order.OrderId}, []string{"*"})
			if err != nil {
				return errors.New("build select failed")
			}
			rows1, err := db.Query(cond, val...)
			if err != nil {
				fmt.Println(err.Error())
				return errors.New("database query failed")
			}
			if rows1.Next() {
				order.RateAndReview = "you have reviewed this order"
			} else {
				order.RateAndReview = "http://localhost:8080/user/auth/rate/" + order.OrderId
			}

		}
		if order.IsCompleted == "expired" || order.IsCompleted == "expired (refunded)" {
			order.RateAndReview = "expired order,can't rate and review"
		}
		order.CheckInfo = "http://localhost:8080/user/auth/orderinfo/" + order.OrderId
		*orders = append(*orders, order)
	}
	return nil
}

func (u *User) CheckOrderInfo(c *gin.Context) {
	id := c.Param("id")
	err := isBelongToUser(id, u.Phone)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	orderInfoJson, err := Rdb.Get(Ctx, fmt.Sprintf("orderInfo:%s", id)).Result()
	if errors.Is(err, redis.Nil) {
		fmt.Println("=======")
		fmt.Println("orderInfo在redis中不存在，序列化后存入redis")
		fmt.Println("=======")
		//redis中没有缓存 查询数据库
		order := &OrderConcreteInfo{}
		err = checkInfo(order, id)
		if err != nil {
			c.JSON(401, gin.H{"error": err.Error()})
			return
		}

		//序列化
		orderInfoData, err := json.Marshal(order)
		if err != nil {
			c.JSON(401, gin.H{"error": err.Error()})
			return
		}
		err = Rdb.Set(Ctx, fmt.Sprintf("orderInfo:%s", id), orderInfoData, 3*24*time.Hour).Err()
		if err != nil {
			c.JSON(401, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"order": order})
		return
	} else if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	} else {
		//redis中存在订单数据缓存 进行反序列化
		fmt.Println("=======")
		fmt.Println("redis中存在 直接从缓存取出")
		fmt.Println("=======")

		order := &OrderConcreteInfo{}
		err = json.Unmarshal([]byte(orderInfoJson), order)
		if err != nil {
			c.JSON(401, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"order": order})
		return
	}

}

func checkInfo(order *OrderConcreteInfo, id string) error {
	order.OrderId = id
	exactTime := ""
	replyTime := ""
	birth := ""
	where := map[string]interface{}{"id": id}
	phone := ""
	selectField := []string{"user", "type", "is_completed", "create_time"}
	cond, val, err := builder.BuildSelect("orders", where, selectField)
	if err != nil {
		return errors.New("build select failed")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		fmt.Println(err.Error())
		return errors.New("database query failed--checkinfo")
	}
	if rows.Next() {
		err = rows.Scan(&phone, &order.Type, &order.IsCompleted, &exactTime)
		if err != nil {
			return err
		}
		layout := "2006-01-02 15:04:05"
		t, err := time.Parse(layout, exactTime)
		if err != nil {
			return errors.New("time parse failed")
		}
		creatTime := t.Format("Jan 2,2006 15:04:05")
		order.OrderTime = creatTime
	}

	cond, val, err = builder.BuildSelect("reply", map[string]interface{}{"id": id}, []string{"message", "reply_time"})
	if err != nil {
		return errors.New("build reply select failed--1")
	}
	rows, err = db.Query(cond, val...)
	if err != nil {
		return errors.New("database query reply failed--1")
	}
	if rows.Next() {
		err = rows.Scan(&order.Reply, &replyTime)
		layout := "2006-01-02 15:04:05"
		t, err := time.Parse(layout, replyTime)
		if err != nil {
			return errors.New("time parse failed")
		}
		order.DeliveryTime = t.Format("Jan 02,2006 15:04:05")
		if err != nil {
			return errors.New("rows scan failed--1")
		}
	} else {
		order.Reply = "no reply"
		if order.IsCompleted == "expired" || order.IsCompleted == "Expired" || order.IsCompleted == "expired (refunded)" {
			order.DeliveryTime = "Expired"
		}
	}

	cond, val, err = builder.BuildSelect("user", map[string]interface{}{"phone": phone}, []string{"name", "birth", "gender"})
	if err != nil {
		return errors.New("build select failed--2")
	}
	rows, err = db.Query(cond, val...)
	if err != nil {
		return errors.New("database query failed--2")
	}
	if rows.Next() {
		err = rows.Scan(&order.Name, &birth, &order.Gender)
		if err != nil {
			return errors.New("rows scan failed--2")
		}
		t, err := time.Parse("2006-01-02", birth)
		if err != nil {
			return errors.New("time parse failed")
		}
		order.DateOfBirth = t.Format("Jan 02,2006")
	}

	cond, val, err = builder.BuildSelect("orders_info", map[string]interface{}{"id": id}, []string{"general_situ", "specified_ques"})
	if err != nil {
		return errors.New("build select failed--3")
	}
	rows, err = db.Query(cond, val...)
	if err != nil {
		return errors.New("database query failed--3")
	}
	if rows.Next() {
		err = rows.Scan(&order.GeneralSitu, &order.SpecifiedQues)
		if err != nil {
			return errors.New("rows scan failed--3")
		}
	}

	return nil
}

func isBelongToUser(id, phone string) error {
	var orderId string
	var isFinished string
	cond, val, err := builder.BuildSelect("orders", map[string]interface{}{"user": phone}, []string{"id", "is_completed"})
	if err != nil {
		return errors.New("build select failed")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return errors.New("database query failed--isbelongtouser")
	}
	for rows.Next() {
		err = rows.Scan(&orderId, &isFinished)
		if err != nil {
			return errors.New("rows scan failed")
		}
		if orderId == id {
			if isFinished == "pending" || isFinished == "Pending" {
				return errors.New("this order is not completed yet")
			}
			return nil
		}
	}
	return errors.New("order id doesn't exist")
}
