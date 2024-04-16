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

type Display struct {
	UserName             string
	SpecifiedQues        string
	ServiceType          string
	IsCompleted          string
	CreateTime           string
	SpecifiedInformation string
}
type OrderInfo struct {
	BasicInfo     string
	GeneralSitu   string
	SpecifiedQues string
	Reply         string
}
type ReplyRequest struct {
	Reply string `json:"reply"`
}

func (a *Advisor) ViewOrders(c *gin.Context) {

	var orders []Display
	order := Display{}
	var id int
	var phone string
	var exactTime string
	where := map[string]interface{}{
		"advisor": a.Phone,
	}
	selectField := []string{"id", "user", "type", "is_completed", "create_time"}
	cond, val, err := builder.BuildSelect("orders", where, selectField)
	if err != nil {
		c.JSON(401, gin.H{"error": "build select failed"})
		return
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		c.JSON(401, gin.H{"error": "database query failed--vieworders"})
		return
	}
	for rows.Next() {
		err = rows.Scan(&id, &phone, &order.ServiceType, &order.IsCompleted, &exactTime)
		if err != nil {
			c.JSON(401, gin.H{"error": "rows scan failed"})
			return
		}
		layout := "2006-01-02 15:04:05"
		format, err := time.Parse(layout, exactTime)
		formatTime := format.Format("Jan 02,2006")
		order.CreateTime = formatTime
		if err != nil {
			c.JSON(401, gin.H{"error": "time parse failed"})
			return
		}

		order.UserName, err = obtainName(phone)
		if err != nil {
			c.JSON(401, gin.H{"error": "name doesn't exist"})
			return
		}
		order.SpecifiedQues, err = obtainQues(id)
		if err != nil {
			c.JSON(401, gin.H{"error": "order id doesn't exist"})
			return
		}
		order.SpecifiedInformation = "http://localhost:8080/advisor/auth/vieworders/" + strconv.Itoa(id)
		orders = append(orders, order)
	}
	if len(orders) == 0 {
		c.JSON(401, gin.H{"error": "there are no orders currently"})
		return
	}
	c.JSON(200, gin.H{"your orders": orders})
}

//func isExpired(create string, isCompleted *string) error {
//	layout := "2006-01-02 15:04:05"
//	t, err := time.Parse(layout, create)
//	if err != nil {
//		return errors.New("create time parse failed")
//	}
//	createTime := t.Format(time.RFC3339)
//	current := time.Now()
//	currentTime := current.Format(time.RFC3339)
//	c, err := time.Parse(time.RFC3339, createTime)
//	if err != nil {
//		return errors.New("createtime parse failed")
//	}
//	now, err := time.Parse(time.RFC3339, currentTime)
//	correct := now.Add(8 * time.Hour)
//	if err != nil {
//		return errors.New("current time parse failed")
//	}
//	duration := correct.Sub(c)
//	if duration > 24*time.Hour {
//		//	*isCompleted = "Expired"
//		//}
//		//if duration > 2*time.Minute {
//		*isCompleted = "Expired"
//	}
//	return nil
//}

func (a *Advisor) SpecifiedInfo(c *gin.Context) {
	id := c.Param("id")
	claims, have := c.Get("advisorClaims")
	if !have {
		c.JSON(401, gin.H{"error": "claims don't exist"})
	}
	advisorData := claims.(*AdvisorClaims)
	advisor := advisorData.Advisor
	order := &OrderInfo{}
	err := searchOrder(id, order, advisor.Phone)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"OrderInfo": *order})
}

func searchOrder(id string, order *OrderInfo, phone string) error {
	var advisorPhone string
	cond, val, err := builder.BuildSelect("orders", map[string]interface{}{"id": id}, []string{"advisor"})
	if err != nil {
		return errors.New("build select failed")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return errors.New("database query failed")
	}
	if rows.Next() {
		err = rows.Scan(&advisorPhone)
		if err != nil {
			return errors.New("rows scan failed")
		}
	}

	where := map[string]interface{}{
		"id": id,
	}
	selectField := []string{"basic_info", "general_situ", "specified_ques"}
	cond, val, err = builder.BuildSelect("orders_info", where, selectField)
	if err != nil {
		return errors.New("build select failed")
	}
	rows, err = db.Query(cond, val...)
	if err != nil {
		return errors.New("database query failed--searchorder")
	}
	if rows.Next() {
		err = rows.Scan(&order.BasicInfo, &order.GeneralSitu, &order.SpecifiedQues)
		if err != nil {
			return errors.New("rows scan failed")
		}
		if advisorPhone != phone {
			return errors.New("this is not your order,you are not authorized to view order information")
		}
		err := whetherExpired(id, &order.Reply)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("order id doesn't exist")
}

func whetherExpired(id string, reply *string) error {
	var isCompleted string
	cond, val, err := builder.BuildSelect("orders", map[string]interface{}{"id": id}, []string{"is_completed"})
	if err != nil {
		return err
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return nil
	}
	if rows.Next() {
		err = rows.Scan(&isCompleted)
		if err != nil {
			return err
		}
		if isCompleted == "expired" || isCompleted == "expired (refunded)" {
			*reply = "can't reply,this order is expired"
		}
		if isCompleted == "completed" || isCompleted == "Completed" {
			*reply = "can't reply,this order is completed"
		}
		if isCompleted == "pending" {
			*reply = "http://localhost:8080/advisor/auth/reply/" + id
		}
	}
	return nil
}

func (a *Advisor) ReplyOrder(c *gin.Context) {
	tx, err := db.Begin()
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
	}
	id := c.Param("id")
	var user, advisor string
	var reply ReplyRequest
	var charge string

	err = isExpired(id)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}

	if err := c.ShouldBindJSON(&reply); err != nil {
		c.JSON(401, gin.H{"error": "reply failed"})
		return
	}
	if len(reply.Reply) < 20 {
		c.JSON(401, gin.H{"error": "please reply more than 20 characters"})
		return
	}
	cond, val, err := builder.BuildSelect("orders", map[string]interface{}{"id": id}, []string{"user", "advisor"})
	if err != nil {
		c.JSON(401, gin.H{"error": "build select failed"})
		return
	}
	rows, err := db.Query(cond, val...)
	if err != nil {
		c.JSON(401, gin.H{"error": "database query failed"})
		return
	}
	if rows.Next() {
		err = rows.Scan(&user, &advisor)
		if err != nil {
			c.JSON(401, gin.H{"error": "rows scan failed"})
			return
		}
	}
	where := map[string]interface{}{
		"id": id,
	}
	selectField := []string{"cost"}
	cond, val, err = builder.BuildSelect("orders_info", where, selectField)
	if err != nil {
		c.JSON(401, gin.H{"error": "build select failed"})
		return
	}
	rows, err = db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		c.JSON(401, gin.H{"error": "database query failed--replyorder"})
		return
	}
	if rows.Next() {
		err = rows.Scan(&charge)
		if err != nil {
			c.JSON(401, gin.H{"error": "rows scan failed"})
			return
		}
	} else {
		c.JSON(401, gin.H{"error": "order id doesn't exist"})
		return
	}
	if a.Phone != advisor {
		c.JSON(401, gin.H{"error": "this is not your order,you are not authorized to reply."})
		return
	}
	now := time.Now()
	layout := "2006-01-02 15:04:05"
	current := now.Format(layout)

	if err != nil {
		c.JSON(401, gin.H{"error": "time parse error"})
		return
	}
	cond, val, err = builder.BuildSelect("reply", map[string]interface{}{"id": id}, []string{"*"})
	if err != nil {
		c.JSON(401, gin.H{"error": "build select failed"})
		return
	}
	rows, err = db.Query(cond, val...)
	if err != nil {
		c.JSON(401, gin.H{"error": "database query failed"})
		return
	}
	if rows.Next() {
		c.JSON(401, gin.H{"error": "you have replied this order,can't reply repeatedly!"})
		return
	}
	data := map[string]interface{}{
		"id":         id,
		"message":    reply.Reply,
		"reply_time": current,
	}
	cond, val, err = builder.BuildInsert("reply", []map[string]interface{}{data})
	if err != nil {
		c.JSON(401, gin.H{"error": "build insert failed"})
		return
	}
	_, err = tx.Exec(cond, val...)
	if err != nil {
		//c.JSON(401, gin.H{"error": "database execute failed"})
		err1 := tx.Rollback()
		if err1 != nil {
			c.JSON(401, gin.H{"error": err1.Error()})
			return
		}
		c.JSON(401, gin.H{"error": "tx execute failed"})
		return
	}
	err = updateComplete(id, tx)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	err = increaseCoins(id, advisor, charge, tx)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	err = increaseOrders(advisor, "complete_order", tx)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
	}
	err = tx.Commit()
	if err != nil {
		c.JSON(401, gin.H{"error": "commit failed"})
	}
	c.JSON(200, gin.H{"message": "reply successfully!"})
}

func isExpired(id string) error {
	var isCompleted string
	cond, val, err := builder.BuildSelect("orders", map[string]interface{}{"id": id}, []string{"is_completed"})
	if err != nil {
		return err
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(&isCompleted)
		if err != nil {
			return err
		}
		if isCompleted == "expired" || isCompleted == "expired (refunded)" {
			return errors.New("this order is expired,please check again")
		}
		if isCompleted == "completed" || isCompleted == "Completed" {
			return errors.New("this order is completed,you can't reply repeatedly")
		}
	}
	return nil
}

func increaseCoins(id, phone string, charge string, tx *sql.Tx) error {
	coin := 0.0
	increasedCoins := 0.0
	serviceType := ""
	cond, val, err := builder.BuildSelect("advisor", map[string]interface{}{"phone": phone}, []string{"coin"})
	if err != nil {
		return errors.New("build select failed--increaseCoins")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return errors.New("database query failed--increaseCoins")
	}
	if rows.Next() {
		err = rows.Scan(&coin)
		if err != nil {
			return errors.New("rows scan failed--increaseCoins")
		}
	} else {
		return errors.New("phone doesn't exist--increaseCoins")
	}

	cond, val, err = builder.BuildSelect("orders", map[string]interface{}{"id": id}, []string{"type"})
	if err != nil {
		return errors.New("build select failed")
	}
	rows, err = db.Query(cond, val...)
	if err != nil {
		return errors.New("database query failed")
	}
	if rows.Next() {
		err = rows.Scan(&serviceType)
		if err != nil {
			return errors.New("rows scan failed")
		}
	}
	current := time.Now()
	layout := "2006-01-02 15:04:05"
	currentTime := current.Format(layout)
	chargeCoins, err := strconv.ParseFloat(charge, 64)
	if err != nil {
		return errors.New("parse float failed--2--increaseCoins")
	}
	expedite, err := whetherExpedite(id)
	if err != nil {
		return err
	}
	if expedite {
		increasedCoins = coin + chargeCoins + chargeCoins/2
	} else {
		increasedCoins = coin + chargeCoins
	}

	data := map[string]interface{}{
		"phone":   phone,
		"type":    serviceType,
		"credits": "+" + charge,
		"time":    currentTime,
	}
	cond, val, err = builder.BuildInsert("transactions", []map[string]interface{}{data})
	if err != nil {
		return errors.New("build insert failed")
	}
	_, err = tx.Exec(cond, val...)
	if err != nil {
		err1 := tx.Rollback()
		if err1 != nil {
			return errors.New("tx rollback failed")
		}
		return errors.New("database execute failed")
	}

	if expedite {
		current = time.Now()
		currentTime = current.Format(layout)
		data = map[string]interface{}{
			"phone":   phone,
			"type":    "Speed-up Order",
			"credits": "+" + fmt.Sprintf("%.1f", chargeCoins/2),
			"time":    currentTime,
		}
		cond, val, err = builder.BuildInsert("transactions", []map[string]interface{}{data})
		if err != nil {
			return errors.New("build insert failed")
		}
		_, err = tx.Exec(cond, val...)
		if err != nil {
			err1 := tx.Rollback()
			if err1 != nil {
				return errors.New("tx rollback failed")
			}
			return errors.New("database execute failed")
		}
	}

	cond, val, err = builder.BuildUpdate("advisor", map[string]interface{}{"phone": phone}, map[string]interface{}{"coin": increasedCoins})
	if err != nil {
		return errors.New("build update failed--increaseCoins")
	}
	_, err = tx.Exec(cond, val...)
	if err != nil {
		err1 := tx.Rollback()
		if err1 != nil {
			return errors.New("tx rollback failed")
		}
		return errors.New("database execute failed--increaseCoins")
	}
	return nil
}

func whetherExpedite(id string) (bool, error) {
	var expedite sql.NullString
	cond, val, err := builder.BuildSelect("orders", map[string]interface{}{"id": id}, []string{"expedite"})
	if err != nil {
		return false, err
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return false, err
	}
	if rows.Next() {
		err = rows.Scan(&expedite)
		if err != nil {
			return false, err
		}
		if !expedite.Valid {
			return false, nil
		}
	}
	return true, nil
}

func updateComplete(id string, tx *sql.Tx) error {

	cond, val, err := builder.BuildUpdate("orders", map[string]interface{}{"id": id}, map[string]interface{}{"is_completed": "completed"})
	if err != nil {
		return errors.New("build update failed")
	}
	_, err = tx.Exec(cond, val...)
	if err != nil {
		err1 := tx.Rollback()
		if err1 != nil {
			return errors.New("tx rollback failed")
		}
		return errors.New("database execute failed")
	}
	return nil
}

func (a *Advisor) TransactionsDetails(c *gin.Context) {
	transactions := &[]Transaction{}
	transaction := Transaction{}
	cond, val, err := builder.BuildSelect("transactions", map[string]interface{}{"phone": a.Phone, "_orderby": "time desc"}, []string{"type", "credits", "time"})
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
		err = rows.Scan(&transaction.Type, &transaction.Credits, &transaction.Time)
		if err != nil {
			c.JSON(401, gin.H{"error": "rows scan failed"})
		}
		current := time.Now()
		layout := "2006-01-02 15:04:05"
		currentTimeFormat := current.Format(layout)
		currentTime, err := time.Parse(layout, currentTimeFormat)
		if err != nil {
			c.JSON(401, gin.H{"error": "time parse failed"})
			return
		}
		transacTime, err := time.Parse(layout, transaction.Time)
		if err != nil {
			c.JSON(401, gin.H{"error": "time parse error"})
			return
		}
		if currentTime.Sub(transacTime) > 3*24*time.Hour {
			continue
		}
		*transactions = append(*transactions, transaction)
	}
	if len(*transactions) == 0 {
		c.JSON(401, gin.H{"error": "there are no transactions in 3 days"})
		return
	}
	c.JSON(200, gin.H{"TransactionsDetails": transactions})
}
