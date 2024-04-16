package model

import (
	"errors"
	"fmt"
	"github.com/didi/gendry/builder"
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
)

type CommentRequest struct {
	Rating string `json:"rating"`
	Review string `json:"review"`
	Tip    string `json:"tip"`
}
type Collect struct {
	AdvisorName string
	Bio         string
	ConnectNow  string
}
type Transaction struct {
	Type    string
	Credits string
	Time    string
}

func (u *User) RateReviewAndTip(c *gin.Context) {
	id := c.Param("id")
	var req CommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(401, gin.H{"error": "bind json failed"})
		return
	}
	if req.Rating == "" {
		c.JSON(401, gin.H{"error": "please input score"})
		return
	}
	s, err := strconv.ParseFloat(req.Rating, 64)
	if err != nil {
		c.JSON(401, gin.H{"error": "please input number"})
		return
	}
	if s < 0 || s > 5.0 {
		c.JSON(401, gin.H{"error": "score should be between 0 and 5"})
		return
	}
	if req.Rating == "" {
		req.Rating = "0"
	}
	if req.Review == "" || req.Review == "0" {
		c.JSON(401, gin.H{"error": "pleaser input review"})
		return
	}
	cond, val, err := builder.BuildSelect("review", map[string]interface{}{"id": id}, []string{"*"})
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	rows, err := db.Query(cond, val...)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	if rows.Next() {
		c.JSON(401, gin.H{"error": "you have reviewed this order,can't reply repeatedly"})
		return
	}

	err = rateAndReview(id, u.Phone, req)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	err = tipCoin(id, req.Tip)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "write review successfully"})
}

func tipCoin(id string, tip string) error {
	if tip == "" {
		return nil
	}
	var userPhone, advisorPhone string
	var userBalance, advisorBalance float64
	cond, val, err := builder.BuildSelect("orders", map[string]interface{}{"id": id}, []string{"user", "advisor"})
	if err != nil {
		return errors.New("build select failed")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return errors.New("database query failed--1--tipcoin")
	}
	if rows.Next() {
		err = rows.Scan(&userPhone, &advisorPhone)
		if err != nil {
			return errors.New("rows scan failed--1--tipcoin")
		}
	}
	cond, val, err = builder.BuildSelect("user", map[string]interface{}{"phone": userPhone}, []string{"coin"})
	if err != nil {
		return errors.New("build select failed")
	}
	rows, err = db.Query(cond, val...)

	if err != nil {
		return errors.New("database query failed--2--tipcoin")
	}
	if rows.Next() {
		err = rows.Scan(&userBalance)
		if err != nil {
			return errors.New("rows scan failed--2--tipcoin")
		}
	}
	tips, err := strconv.ParseFloat(tip, 64)
	if err != nil {
		return errors.New("please input number")
	}
	if tips < 0 {
		return errors.New("please input correct number")
	}
	if userBalance < tips {
		return errors.New("your balance is not enough,please recharge first")
	}
	remains := userBalance - tips
	cond, val, err = builder.BuildUpdate("user", map[string]interface{}{"phone": userPhone}, map[string]interface{}{"coin": remains})
	if err != nil {
		return errors.New("build update failed")
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return errors.New("database execute failed--1--tipcoin")
	}
	cond, val, err = builder.BuildSelect("advisor", map[string]interface{}{"phone": advisorPhone}, []string{"coin"})
	if err != nil {
		return errors.New("build select failed")
	}
	rows, err = db.Query(cond, val...)
	if err != nil {
		return errors.New("database query failed--3--tipcoin")
	}
	if rows.Next() {
		err = rows.Scan(&advisorBalance)
		if err != nil {
			return errors.New("rows scan failed--3--tipcoin")
		}
	}
	increasedCoin := advisorBalance + tips
	cond, val, err = builder.BuildUpdate("advisor", map[string]interface{}{"phone": advisorPhone}, map[string]interface{}{"coin": increasedCoin})
	if err != nil {
		return errors.New("build update failed")
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return errors.New("database execute failed--2--tipcoin")
	}

	current := time.Now()
	layout := "2006-01-02 15:04:05"
	currentTime := current.Format(layout)
	data := map[string]interface{}{
		"phone":   userPhone,
		"type":    "Tip",
		"credits": "-" + fmt.Sprintf("%.1f", tips),
		"time":    currentTime,
	}
	cond, val, err = builder.BuildInsert("transactions", []map[string]interface{}{data})
	if err != nil {
		return errors.New("build insert failed")
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return errors.New("database execute failed")
	}

	return nil
}

func rateAndReview(id string, phone string, req CommentRequest) error {
	var advisorPhone string
	exist, err := isOrderIdExist(id, phone)

	if err != nil {
		return err
	}
	if !exist {
		return errors.New("order id doesn't exist")
	}

	cond, val, err := builder.BuildSelect("orders", map[string]interface{}{"id": id}, []string{"advisor"})
	if err != nil {
		return errors.New("build select failed")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return errors.New("database query failed--1--rateandreview")
	}
	if rows.Next() {
		err = rows.Scan(&advisorPhone)
		if err != nil {
			return errors.New("rows scan failed--1--rateAndReview")
		}
	}

	data := map[string]interface{}{
		"id":     id,
		"tip":    req.Tip,
		"rating": req.Rating,
		"phone":  advisorPhone,
	}
	cond, val, err = builder.BuildInsert("review", []map[string]interface{}{data})
	if err != nil {
		return errors.New("build insert failed")
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return errors.New("database execute failed")
	}
	current := time.Now()
	layout := "2006-01-02 15:04:05"
	currentTime := current.Format(layout)
	data = map[string]interface{}{
		"order_id":    id,
		"phone":       phone,
		"review":      req.Review,
		"review_time": currentTime,
	}
	cond, val, err = builder.BuildInsert("answer", []map[string]interface{}{data})
	if err != nil {
		return errors.New("build insert failed")
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return errors.New("database execute failed")
	}

	res, err := builder.AggregateQuery(Ctx, db, "review", map[string]interface{}{"phone": advisorPhone}, builder.AggregateAvg("rating"))
	if err != nil {
		return errors.New("aggregate query failed")
	}
	avgRating := res.Float64()

	cond, val, err = builder.BuildUpdate("advisor", map[string]interface{}{"phone": advisorPhone}, map[string]interface{}{"rating": fmt.Sprintf("%.1f", avgRating)})
	if err != nil {
		return errors.New("build update failed")
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return errors.New("database query failed")
	}
	sumComment := 0
	cond, val, err = builder.BuildSelect("advisor", map[string]interface{}{"phone": advisorPhone}, []string{"sum_of_comments"})
	if err != nil {
		return errors.New("build select failed")
	}
	rows, err = db.Query(cond, val...)
	if err != nil {
		return errors.New("database query failed")
	}
	if rows.Next() {
		err = rows.Scan(&sumComment)
		if err != nil {
			return errors.New("rows scan failed")
		}
	}
	cond, val, err = builder.BuildUpdate("advisor", map[string]interface{}{"phone": advisorPhone}, map[string]interface{}{"sum_of_comments": sumComment + 1})
	if err != nil {
		return errors.New("build update failed")
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return errors.New("database execute failed")
	}
	return nil
	//ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	//defer cancel()
	//var advisorPhone string
	//exist, err := isOrderIdExist(id, phone)
	//if err != nil {
	//	return err
	//}
	//if !exist {
	//	return errors.New("order id doesn't exist")
	//}
	//s, _ := strconv.ParseFloat(req.Rating, 64)
	//cond, val, err := builder.BuildSelect("orders", map[string]interface{}{"id": id}, []string{"advisor"})
	//if err != nil {
	//	return errors.New("build select failed")
	//}
	//rows, err := db.Query(cond, val...)
	//if err != nil {
	//	return errors.New("database query failed--1--rateandreview")
	//}
	//if rows.Next() {
	//	err = rows.Scan(&advisorPhone)
	//	if err != nil {
	//		return errors.New("rows scan failed--1--rateAndReview")
	//	}
	//}
	//idSlice := []string{}
	//reviewIdSlice := []string{}
	//orderId := ""
	//cond, val, err = builder.BuildSelect("orders", map[string]interface{}{"advisor": advisorPhone}, []string{"id"})
	//if err != nil {
	//	return errors.New("build select failed")
	//}
	//rows, err = db.Query(cond, val...)
	//if err != nil {
	//	fmt.Println(err.Error())
	//	return errors.New("database query failed--2--rateandreview")
	//}
	//for rows.Next() {
	//	err = rows.Scan(&orderId)
	//	if err != nil {
	//		return errors.New("rows scan failed--2--rateAndReview")
	//	}
	//	idSlice = append(idSlice, orderId)
	//}
	//if len(idSlice) == 0 {
	//	return errors.New("the advisor has no orders")
	//} else {
	//	cond, val, err = builder.BuildSelect("review", map[string]interface{}{"order_id": idSlice, "_groupby": "order_id"}, []string{"order_id"})
	//	if err != nil {
	//		return errors.New("build select failed")
	//	}
	//	rows, err = db.Query(cond, val...)
	//	if err != nil {
	//		return errors.New("database query failed")
	//	}
	//	for rows.Next() {
	//		err = rows.Scan(&orderId)
	//		if err != nil {
	//			return errors.New("rows scan failed--3--rateAndReview")
	//		}
	//		reviewIdSlice = append(reviewIdSlice, orderId)
	//	}
	//	if len(reviewIdSlice) == 0 {
	//		err = updateAdvisorRating(advisorPhone, req.Rating)
	//		if err != nil {
	//			return err
	//		}
	//	} else {
	//		err = updateAdvisorAvgRating(id, advisorPhone, s, reviewIdSlice)
	//		if err != nil {
	//			return err
	//		}
	//	}
	//
	//}
	//
	//cond, val, err = builder.BuildSelect("advisor", map[string]interface{}{"phone": advisorPhone}, []string{"sum_of_comments"})
	//if err != nil {
	//	return errors.New("build select failed")
	//}
	//rows, err = db.Query(cond, val...)
	//if err != nil {
	//	return errors.New("database query failed")
	//}
	//var sumOfComments float64
	//if rows.Next() {
	//	err = rows.Scan(&sumOfComments)
	//	if err != nil {
	//		return errors.New("rows scan failed")
	//	}
	//}
	//increaseComments := sumOfComments + 1
	//cond, val, err = builder.BuildUpdate("advisor", map[string]interface{}{"phone": advisorPhone}, map[string]interface{}{"sum_of_comments": increaseComments})
	//if err != nil {
	//	return errors.New("build update failed")
	//}
	//_, err = db.Exec(cond, val...)
	//if err != nil {
	//	return errors.New("database execute failed")
	//}
	//
	//current := time.Now()
	//layout := "2006-01-02 15:04:05"
	//currentTime := current.Format(layout)
	//data := []map[string]interface{}{
	//	{"order_id": id, "review": req.Review, "tip": req.Tip, "rating": req.Rating, "review_time": currentTime},
	//}
	//cond, val, err = builder.BuildInsert("review", data)
	//if err != nil {
	//	return errors.New("build insert failed")
	//}
	//
	//_, err = db.Exec(cond, val...)
	//if err != nil {
	//	//return errors.New("database execute failed--rateAndReview")
	//	return errors.New("you have reviewed this order,can't review repeatedly")
	//}
	//return nil
}

//	func updateAdvisorAvgRating(id, advisorPhone string, rating float64, idSlice []string) error {
//		reviewCount := 0.0
//		advisorRating := 0.0
//		for _, val := range idSlice {
//			if id == val {
//				return errors.New("this order has been reviewed,can't review repeatedly")
//			}
//		}
//		cond, val, err := builder.BuildSelect("review", map[string]interface{}{"order_id in": idSlice, "_groupby": "order_id"}, []string{"order_id"})
//		if err != nil {
//			return errors.New("build select failed")
//		}
//		fmt.Println(cond)
//		fmt.Println(val)
//		fmt.Println("idSlice: ", idSlice)
//		rows, err := db.Query(cond, val...)
//		if err != nil {
//			fmt.Println(err.Error())
//			return errors.New("database query failed--1--updateAdvisorAvgRating")
//		}
//		for rows.Next() {
//			reviewCount++
//		}
//		fmt.Println("count", reviewCount)
//		cond, val, err = builder.BuildSelect("advisor", map[string]interface{}{"phone": advisorPhone}, []string{"rating"})
//		if err != nil {
//			return errors.New("build select failed")
//		}
//		rows, err = db.Query(cond, val...)
//		if err != nil {
//			fmt.Println(err.Error())
//			return errors.New("database query failed--2--updateAdvisorAvgRating")
//		}
//		if rows.Next() {
//			err = rows.Scan(&advisorRating)
//			if err != nil {
//				return errors.New("rows scan failed--1--updateAdvisorAvgRating")
//			}
//		}
//
//		updateRating := (reviewCount*advisorRating + rating) / (reviewCount + 1)
//		updateRatingFormat := fmt.Sprintf("%.1f", updateRating)
//		cond, val, err = builder.BuildUpdate("advisor", map[string]interface{}{"phone": advisorPhone}, map[string]interface{}{"rating": updateRatingFormat})
//		if err != nil {
//			return errors.New("build update failed")
//		}
//		_, err = db.Exec(cond, val...)
//		if err != nil {
//			return errors.New("database execute failed--updateAdvisorAvgRating")
//		}
//		return nil
//	}
//
//	func updateAdvisorRating(advisorPhone string, rating string) error {
//		cond, val, err := builder.BuildUpdate("advisor", map[string]interface{}{"phone": advisorPhone}, map[string]interface{}{"rating": rating})
//		if err != nil {
//			return errors.New("build update failed")
//		}
//		_, err = db.Exec(cond, val...)
//		if err != nil {
//			fmt.Println(err.Error())
//			return errors.New("database execute failed--updateAdvisorRating")
//		}
//		return nil
//	}
func isOrderIdExist(id string, phone string) (bool, error) {
	var orderId string
	var userPhone string
	cond, val, err := builder.BuildSelect("reply", map[string]interface{}{"id": id}, []string{"id"})
	if err != nil {
		return false, errors.New("build select failed")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return false, errors.New("database query failed--1--isOrderIdExist")
	}
	if rows.Next() {
		_ = rows.Scan(&orderId)
		cond, val, err = builder.BuildSelect("orders", map[string]interface{}{"id": orderId}, []string{"user"})
		if err != nil {
			return false, errors.New("build select failed--isorderidexist")
		}
		rows1, err := db.Query(cond, val...)
		if err != nil {
			return false, errors.New("database failed--isorderidexist")
		}
		if rows1.Next() {
			err = rows1.Scan(&userPhone)
			if err != nil {
				return false, errors.New("rows1 scan failed--isorderidexist")
			}

			if userPhone == phone {
				return true, nil
			} else {
				return false, errors.New("this is not your order,can't review")
			}
		}
	} else {
		return false, errors.New("order id doesn't exist")
	}
	return false, nil
}

func (u *User) CollectAd(c *gin.Context) {
	id := c.Param("id")
	i, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(401, gin.H{"error": "a to i failed"})
		return
	}
	exist, err := isIdExist(i)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	if !exist {
		c.JSON(401, gin.H{"error": "advisor doesnt't exist"})
	}
	advisorPhone := ""
	cond, val, err := builder.BuildSelect("advisor", map[string]interface{}{"id": id}, []string{"phone"})
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
	if rows.Next() {
		err = rows.Scan(&advisorPhone)
		if err != nil {
			c.JSON(401, gin.H{"error": "rows scan failed"})
			return
		}
	}
	var status, collectId string
	exist, err = isCollected(u.Phone, advisorPhone, &status, &collectId)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	if status == "y" {
		c.JSON(401, gin.H{"error": "you have collected this advisor, can't collect repeatedly"})
		return
	}
	if !exist {
		data := map[string]interface{}{
			"user":      u.Phone,
			"advisor":   advisorPhone,
			"collected": "y",
		}
		cond, val, err = builder.BuildInsert("collect", []map[string]interface{}{data})
		if err != nil {
			c.JSON(401, gin.H{"error": "build insert failed"})
			return
		}
		_, err = db.Exec(cond, val...)
		if err != nil {
			c.JSON(401, gin.H{"error": "database execute failed"})
			return
		}
	} else {
		cond, val, err = builder.BuildUpdate("collect", map[string]interface{}{"id": collectId}, map[string]interface{}{"collected": "y"})
		if err != nil {
			c.JSON(401, gin.H{"error": "build update failed"})
			return
		}
		_, err = db.Exec(cond, val...)
		if err != nil {
			c.JSON(401, gin.H{"error": "database execute failed"})
			return
		}
	}

	c.JSON(200, gin.H{"message": "collect successfully"})
}

func isCollected(uPh, adPh string, status, collectId *string) (bool, error) {
	cond, val, err := builder.BuildSelect("collect", map[string]interface{}{"user": uPh, "advisor": adPh}, []string{"id", "collected"})
	if err != nil {
		return false, errors.New("build select failed")
	}
	rows, err := db.Query(cond, val...)
	if err != nil {
		return false, errors.New("database query failed")
	}
	if rows.Next() {
		err = rows.Scan(collectId, status)
		if err != nil {
			return false, errors.New("rows scan failed")
		}
		return true, nil
	}
	return false, nil
}

func (u *User) CollectList(c *gin.Context) {
	var advisors []Collect
	var advisor Collect
	var phone, id, isCollected string
	cond, val, err := builder.BuildSelect("collect", map[string]interface{}{"user": u.Phone}, []string{"advisor", "collected"})
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
		err = rows.Scan(&phone, &isCollected)
		if err != nil {
			c.JSON(401, gin.H{"error": "rows scan failed"})
			return
		}
		if isCollected == "n" {
			continue
		}
		cond, val, err = builder.BuildSelect("advisor", map[string]interface{}{"phone": phone}, []string{"id", "name", "bio"})
		if err != nil {
			c.JSON(401, gin.H{"error": "build select failed"})
			return
		}
		rows1, err := db.Query(cond, val...)
		if err != nil {
			c.JSON(401, gin.H{"error": "database query failed"})
			return
		}
		if rows1.Next() {
			err = rows1.Scan(&id, &advisor.AdvisorName, &advisor.Bio)
			if err != nil {
				c.JSON(401, gin.H{"error": "rows1 scan failed"})
				return
			}
			advisor.ConnectNow = "http://localhost:8080/user/auth/visit/" + id
		}
		advisors = append(advisors, advisor)
	}
	if len(advisors) == 0 {
		c.JSON(401, gin.H{"error": "there are no advisors collected"})
		return
	}
	c.JSON(200, gin.H{"collect": advisors})
}

func (u *User) TransactionsDetails(c *gin.Context) {
	transactions := &[]Transaction{}
	transaction := Transaction{}
	cond, val, err := builder.BuildSelect("transactions", map[string]interface{}{"phone": u.Phone, "_orderby": "time desc"}, []string{"type", "credits", "time"})
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
