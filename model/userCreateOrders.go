package model

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/didi/gendry/builder"
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
)

type CreateOrderRequest struct {
	Type           string `json:"type"`
	Basic_Info     string `json:"basic_info"`
	General_Situ   string `json:"general_situ"`
	Specified_Ques string `json:"specified_ques"`
	Cost           string `json:"cost"`
}
type OrderList struct {
	UserName      string
	SpecifiedQues string
	ServiceType   string
	IsCompleted   string
	CreateTime    string
}

func (u *User) Create(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(401, gin.H{"error": "create order failed"})
		return
	}
	advisorData, have := c.Get("advisor")
	if !have {
		c.JSON(401, gin.H{"error": "advisor doesn't exist"})
		return
	}
	//extract advisor and user
	serviceType := c.Param("type")

	advisor := advisorData.(*Visit)
	user := u
	var remain float64
	cost := "0"
	err := isAccept(advisor.Phone, serviceType)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	err = isBalanceSufficient(advisor.Phone, user.Phone, serviceType, &remain, &cost)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	err = createOrder(advisor.Phone, user.Phone, serviceType, req, cost, remain)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	//err = deductCoins(user.Phone, remain)
	//if err != nil {
	//	c.JSON(401, gin.H{"error": err.Error()})
	//	return
	//}
	c.JSON(200, gin.H{"message": "order create successfully!", "cost": cost, "remain": fmt.Sprintf("%.1f", remain)})
}

func transactions(phone, serviceType, cost string) error {

	//data := map[string]interface{}{
	//	"phone":   phone,
	//	"type":    serviceType,
	//	"credits": "-" + cost,
	//}
	//
	//cond, val, err := builder.BuildInsert("transactions")
	return nil
}

func isAccept(phone, numType string) error {
	where := map[string]interface{}{
		"phone":    phone,
		"num_type": numType,
	}
	var valid string
	cond, val, err := builder.BuildSelect("service", where, []string{"acceptance"})
	if err != nil {
		return errors.New("build select failed")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return errors.New("database query failed")
	}
	if rows.Next() {
		err = rows.Scan(&valid)
		if err != nil {
			return errors.New("rows scan failed")
		}
	}
	if valid == "OFF" {
		return errors.New("this service is closed")
	}
	return nil
}

func isBalanceSufficient(a_ph, u_ph, num_type string, remain *float64, cost *string) error {
	var userBalance string

	where := map[string]interface{}{"phone": a_ph, "num_type": num_type}
	cond, val, err := builder.BuildSelect("service", where, []string{"price"})
	if err != nil {
		return errors.New("build select failed--1")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return errors.New("database query failed--1")
	}
	if rows.Next() {
		err = rows.Scan(cost)
		if err != nil {
			return errors.New("price doesn't exist")
		}
	}
	where = map[string]interface{}{"phone": u_ph}
	cond, val, err = builder.BuildSelect("user", where, []string{"coin"})
	if err != nil {
		return errors.New("build select failed--2")
	}
	rows, err = db.Query(cond, val...)
	if err != nil {
		return errors.New("database query failed--2")
	}
	if rows.Next() {
		err = rows.Scan(&userBalance)
		if err != nil {
			return errors.New("user coin doesn't exist")
		}
	}
	charge, err := strconv.ParseFloat(*cost, 64)
	if err != nil {
		return errors.New("parse float failed--1")
	}
	balance, err := strconv.ParseFloat(userBalance, 64)
	if err != nil {
		return errors.New("parse float failed--2")
	}
	if balance < charge {
		return errors.New("insufficient balance,please recharge first")
	}
	*remain = balance - charge
	return nil
}

func createOrder(a_ph, u_ph, serviceType string, req CreateOrderRequest, cost string, remain float64) error {
	tx, err := db.Begin()
	if err != nil {
		return errors.New("init tx failed")
	}
	serviceName := ""
	where := map[string]interface{}{
		"phone":    a_ph,
		"num_type": serviceType,
	}
	cond, val, err := builder.BuildSelect("service", where, []string{"service_type"})
	if err != nil {
		return errors.New("build select failed")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return errors.New("database query failed")
	}
	if rows.Next() {
		err = rows.Scan(&serviceName)
		if err != nil {
			return errors.New("rows scan failed")
		}
	}

	if req.General_Situ == "" {
		return errors.New("general situation cannot be empty,please describe your situation")
	}
	req.Cost = cost
	req.Type = serviceName
	data := struct2Map(req)
	cond, val, err = builder.BuildInsert("orders_info", []map[string]interface{}{data})
	if err != nil {
		return errors.New("build insert failed")
	}
	res, err := tx.Exec(cond, val...)
	if err != nil {
		err1 := tx.Rollback()
		if err1 != nil {
			return err1
		}
		return errors.New("database execute failed--1")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return errors.New("get id failed")
	}

	current := time.Now()
	layout := "2006-01-02 15:04:05"
	currentTime := current.Format(layout)
	data = map[string]interface{}{
		"id":           id,
		"user":         u_ph,
		"advisor":      a_ph,
		"type":         req.Type,
		"is_completed": "pending",
		"create_time":  currentTime,
	}
	cond, val, err = builder.BuildInsert("orders", []map[string]interface{}{data})
	if err != nil {
		return errors.New("build insert failed")
	}
	_, err = tx.Exec(cond, val...)
	if err != nil {
		err1 := tx.Rollback()
		if err1 != nil {
			return err1
		}
		return errors.New("database execute failed--2")
	}

	data = map[string]interface{}{
		"phone":   u_ph,
		"type":    req.Type,
		"credits": "-" + cost,
		"time":    currentTime,
	}

	cond, val, err = builder.BuildInsert("transactions", []map[string]interface{}{data})
	if err != nil {
		return errors.New("build select failed")
	}
	_, err = tx.Exec(cond, val...)
	if err != nil {
		err1 := tx.Rollback()
		if err1 != nil {
			return err1
		}
		return errors.New("database query failed")
	}

	trct := fmt.Sprintf("%.1f", remain)
	cond, val, err = builder.BuildUpdate("user", map[string]interface{}{"phone": u_ph}, map[string]interface{}{"coin": trct})
	if err != nil {
		return errors.New("build update failed")
	}
	_, err = tx.Exec(cond, val...)
	if err != nil {
		err1 := tx.Rollback()
		if err1 != nil {
			return err1
		}
		return errors.New("database execute failed")
	}

	err = increaseOrders(a_ph, "total_order", tx)
	if err != nil {
		return errors.New("increase total orders failed--createOrder")
	}
	err = tx.Commit()
	if err != nil {
		return errors.New("commit failed")
	}
	return nil
}

func increaseOrders(phone string, selectField string, tx *sql.Tx) error {
	sumOfOrders := 0
	cond, val, err := builder.BuildSelect("advisor", map[string]interface{}{"phone": phone}, []string{selectField})
	if err != nil {
		return errors.New("build select failed--increaseOrders")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return errors.New("database query failed--increaseOrders")
	}
	if rows.Next() {
		err = rows.Scan(&sumOfOrders)
		if err != nil {
			return errors.New("rows scan failed--increaseOrders")
		}
	}
	sumOfOrders += 1

	cond, val, err = builder.BuildUpdate("advisor", map[string]interface{}{"phone": phone}, map[string]interface{}{selectField: sumOfOrders})
	if err != nil {
		return errors.New("build update failed--increaseOrders")
	}
	_, err = tx.Exec(cond, val...)
	if err != nil {
		err1 := tx.Rollback()
		if err1 != nil {
			return errors.New("tx rollback failed")
		}
		return errors.New("database execute failed--increaseOrders")
	}
	return nil
}

//func deductCoins(userPhone string, remains float64) error {
//	truncate := fmt.Sprintf("%.1f", remains)
//	cond, val, err := builder.BuildUpdate("user", map[string]interface{}{"phone": userPhone}, map[string]interface{}{"coin": truncate})
//	if err != nil {
//		return errors.New("build update failed")
//	}
//	_, err = db.Exec(cond, val...)
//	if err != nil {
//		return errors.New("database execute failed")
//	}
//	return nil
//}

func (u *User) ViewOrderList(c *gin.Context) {
	var orders []OrderList
	order := OrderList{}
	var id int
	var phone, createTime string
	selectField := []string{
		"id",
		"user",
		"type",
		"is_completed",
		"create_time",
	}
	cond, val, err := builder.BuildSelect("orders", map[string]interface{}{}, selectField)
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
		err = rows.Scan(&id, &phone, &order.ServiceType, &order.IsCompleted, &createTime)
		layout := "2006-01-02 15:04:05"
		create, err := time.Parse(layout, createTime)
		if err != nil {
			fmt.Println(err.Error())
			c.JSON(401, gin.H{"error": "time parse failed"})
			return
		}
		formatTime := create.Format("Jan 2,2006")
		order.CreateTime = formatTime
		if err != nil {

			c.JSON(401, gin.H{"error": "rows scan failed"})
			return
		}
		order.UserName, err = obtainName(phone)
		if err != nil {
			c.JSON(401, gin.H{"error": err.Error()})
			return
		}
		order.SpecifiedQues, err = obtainQues(id)
		orders = append(orders, order)
	}
	if len(orders) == 0 {
		c.JSON(401, gin.H{"error": "there are no orders currently"})
		return
	}
	c.JSON(200, gin.H{"order list": orders})
}

func obtainName(phone string) (string, error) {
	where := map[string]interface{}{
		"phone": phone,
	}
	var name string
	cond, val, err := builder.BuildSelect("user", where, []string{"name"})
	if err != nil {
		return "", errors.New("build select failed")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return "", errors.New("database query failed")
	}

	if rows.Next() {
		err = rows.Scan(&name)
		if err != nil {
			return "", errors.New("rows scan failed")
		}
		return name, nil
	}
	return "", errors.New("phone doesn't exist")
}

func obtainQues(id int) (string, error) {
	var ques string
	where := map[string]interface{}{
		"id": id,
	}
	cond, val, err := builder.BuildSelect("orders_info", where, []string{"specified_ques"})
	if err != nil {
		return "", errors.New("build select failed")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return "", errors.New("database query failed")
	}
	if rows.Next() {
		err = rows.Scan(&ques)
		if err != nil {
			return "", errors.New("rows scan failed")
		}
		return ques, nil
	}
	return "", errors.New("id doesn't exist")
}
