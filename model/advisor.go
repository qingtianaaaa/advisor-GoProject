package model

import (
	"errors"
	"fmt"
	"github.com/didi/gendry/builder"
	"github.com/didi/gendry/scanner"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
)

const (
	Open  = 1
	Close = 0
)

type Advisor struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Phone         string `json:"phone"`
	Status        string `json:"status"`
	TotalOrder    string `json:"total_order"`
	SumOfComments string `json:"sum_of_comments"`
	Rating        string `json:"rating"`
	Coin          string `json:"coin"`
	About         string `json:"about"`
	Bio           string `json:"bio"`
}
type AdvisorUpdateRequest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Phone  string `json:"phone"`
	About  string `json:"about"`
	Bio    string `json:"bio"`
}
type SetPriceRequest struct {
	Price      string `json:"price"`
	Acceptance string `json:"acceptance"`
}
type Service struct {
	NumType     string `json:"num_type"`
	ServiceType string `json:"service_type"`
	Price       string `json:"price"`
	Acceptance  string `json:"acceptance"`
}

func (a *Advisor) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Register failed"})
		return
	}
	if req.Phone == "" || len(req.Phone) != 11 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please enter 11-digit phone number"})
		return
	}
	if req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please enter password"})
		return
	}
	whether, err := isRegistered("ad_register", req.Phone)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	whether, err = isRegistered("register", req.Phone)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if whether {
		err = insertRegis("ad_register", "advisor", req, c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Register information insert table advisor failed"})
		}
		err = insertTable("service", req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Register information insert table price failed"})
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "Advisor registered successfully!Please login!"})
}

func insertTable(table string, req RegisterRequest) error {
	where := map[string]interface{}{
		"phone": req.Phone,
	}
	var id int
	cond, val, err := builder.BuildSelect("advisor", where, []string{"id"})
	if err != nil {
		return err
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return err
	}
	for rows.Next() {
		if err = rows.Scan(&id); err != nil {
			return err
		}
	}
	data := []map[string]interface{}{
		{"phone": req.Phone, "service_type": "Text Reading", "num_type": 1},
		{"phone": req.Phone, "service_type": "Audio Reading", "num_type": 2},
		{"phone": req.Phone, "service_type": "Video Reading", "num_type": 3},
		{"phone": req.Phone, "service_type": "Live Text Chat", "num_type": 4},
	}
	cond, val, _ = builder.BuildInsert(table, data)
	_, err = db.Exec(cond, val...)
	if err != nil {
		return nil
	}
	return nil
}

func (a *Advisor) Update(c *gin.Context) {
	var req AdvisorUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Update failed"})
		return
	}
	claims, have := c.Get("advisorClaims")
	if !have {
		c.JSON(http.StatusBadRequest, gin.H{"error": "claims don't exist"})
		return
	}
	claimsDate := claims.(*AdvisorClaims)
	adv := claimsDate.Advisor

	exist, err := isExist("ad_register", adv.Phone)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if exist {
		err = updateAdvisor(*a, req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Phone == "" || req.Phone == a.Phone {
			c.JSON(http.StatusOK, gin.H{"message": "Advisor updated successfully"})

		} else {
			c.JSON(http.StatusOK, gin.H{"message": "Change binding phone number successfully,please login again"})
		}
		if req.Phone == "" {
			req.Phone = a.Phone
		}
		updateRealTimeAdvisor(req.Phone, a)
		return
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid number,please login again"})
		return
	}
}

func (a *Advisor) Login(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Login failed"})
		return
	}
	if req.Phone == "" || len(req.Phone) != 11 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please enter 11-digit phone number"})
		return
	}
	if req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please enter password"})
		return
	}

	exist, err := isExist("ad_register", req.Phone)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if exist {
		err = loginAdvisor(req, c)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		} else {
			updateRealTimeAdvisor(req.Phone, a) // remember to change a after user or advisor updated their personal information
			return
		}
	} else {
		exist, err = isExist("register", req.Phone)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if exist {
			c.JSON(http.StatusBadRequest, gin.H{"error": "This is a user account"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "This number is not registered.Please register first"})
		}
		return
	}
}

func loginAdvisor(login RegisterRequest, c *gin.Context) error {
	whether, err := isCorrect("ad_register", login)
	if err != nil {
		//c.JSON(http.StatusBadRequest, gin.H{"error": "Wrong password"})
		return errors.New("wrong password")
	}
	if whether {
		var advisor Advisor
		advisor, err = getAdvisor(login)
		if err != nil {
			return err
		}
		jwtStr := JWTGenerateAdvisorToken(advisor, time.Hour)
		c.JSON(http.StatusOK, gin.H{"Token": jwtStr, "ID": advisor.ID, "Name": advisor.Name, "Phone": advisor.Phone, "message": "Login successfully"})
		c.Set("token", jwtStr)
		return nil
	}
	return errors.New("login failed")
}

func getAdvisor(login RegisterRequest) (Advisor, error) {
	var advisor Advisor
	where := map[string]interface{}{
		"phone": login.Phone,
	}
	cond, val, err := builder.BuildSelect("advisor", where, []string{"*"})
	if err != nil {
		return advisor, err
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return advisor, err
	}
	//defer rows.Close()
	scanner.Scan(rows, &advisor)
	//fmt.Println("getAdvisor中是：", advisor)
	return advisor, nil
}

func updateRealTimeAdvisor(phone string, a *Advisor) {
	where := map[string]interface{}{
		"phone": phone,
	}
	cond, val, err := builder.BuildSelect("advisor", where, []string{"*"})
	if err != nil {
		panic(err.Error())
	}
	rows, err := db.Query(cond, val...)
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()
	err = scanner.Scan(rows, a)
	if a.Status == "in-service" {
		a.Status = "1"
	}
	if a.Status == "out-of-service" {
		a.Status = "0"
	}
}
func changeAdvisorPhoneNumber(updTab1, updTab2, phBefore, phAfter string) error {
	no, err := isRegistered("ad_register", phAfter)
	if err != nil {
		return errors.New("this number has been registered,please provide a different " +
			"number if you want to change your binding number")
	}
	no, err = isRegistered("register", phAfter)
	if err != nil {
		return errors.New("this number has been registered,please provide a different " +
			"number if you want to change your binding number")
	}
	if no { //dont register  ,then change the number
		where := map[string]interface{}{"phone": phBefore}
		update := map[string]interface{}{"phone": phAfter}
		cond, val, err := builder.BuildUpdate(updTab1, where, update)

		if err != nil {
			return errors.New("build update failed")
		}
		_, err = db.Exec(cond, val...)
		//fmt.Println("========err", err.Error())
		if err != nil {
			return errors.New("database execution failed")
		}

		where = map[string]interface{}{"phone": phBefore}
		update = map[string]interface{}{"phone": phAfter}
		cond, val, err = builder.BuildUpdate(updTab2, where, update)

		if err != nil {
			return errors.New("build update failed")
		}
		_, err = db.Exec(cond, val...)
		if err != nil {
			return errors.New("database execution failed")
		}
		return nil

	}
	return errors.New("the number has benn registered,please change a number if you want to change your binding number")
}

// Written:by lwj
// Function:Update user information
// Return:error
func updateAdvisor(adv Advisor, req AdvisorUpdateRequest) error {
	if req.Phone == "" {
		req.Phone = adv.Phone
	}
	if len(req.Phone) != 11 {
		return errors.New("please enter 11-digit number if you want to change your binding number")
	}

	if req.Phone != adv.Phone {
		err := changeAdvisorPhoneNumber("price", "ad_register", adv.Phone, req.Phone)
		if err != nil {
			//return errors.New("change number failed")
			return err
		}

	}
	if req.About == "" {
		req.About = adv.About
	}
	if req.Bio == "" {
		req.Bio = adv.Bio
	}
	if req.Name == "" {
		req.Name = adv.Name
	}
	if req.Status == "" {
		req.Status = adv.Status
	}

	data := struct2Map(req)

	if req.Status == "0" {
		data["Status"] = "out-of-service"
	} else {
		data["Status"] = "in-service"
	}

	where := map[string]interface{}{
		"phone": req.Phone,
	}
	cond, val, err := builder.BuildUpdate("advisor", where, data)
	if err != nil {
		return errors.New("build select failed")
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return errors.New("database execution failed")
	}
	return nil
}

func (a *Advisor) DisplayService(c *gin.Context) {
	services := []Service{}
	service := Service{}
	selectField := []string{"num_type", "service_type", "price", "acceptance"}
	cond, val, err := builder.BuildSelect("service", map[string]interface{}{"phone": a.Phone}, selectField)
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
		err = rows.Scan(&service.NumType, &service.ServiceType, &service.Price, &service.Acceptance)
		if err != nil {
			c.JSON(401, gin.H{"error": "rows scan failed"})
			return
		}
		services = append(services, service)
	}
	c.JSON(200, gin.H{"service": services})

	//var req SetPriceRequest
	//if err := c.ShouldBindJSON(&req); err != nil {
	//	c.JSON(http.StatusBadRequest, gin.H{"error": "Modify price failed"})
	//	return
	//}
	//claims, exists := c.Get("claims")
	//if !exists {
	//	c.JSON(http.StatusBadRequest, gin.H{"error": "Claims don't exist"})
	//	return
	//}
	//claimsData := claims.(*AdvisorClaims) //extract advisor information from gin.Context
	//advisor := claimsData.Advisor
	//if a.Phone != advisor.Phone {
	//	c.JSON(http.StatusBadRequest, gin.H{"error": "The login number doesn't match the number in token"})
	//	return
	//}
	//whether, err := isExist("price", a.Phone) //notice: Should pass *a instead of a, because a is a pointer and *a is a struct what the function wants
	//if err != nil {
	//	c.JSON(http.StatusBadRequest, gin.H{"error": "The number doesn't exist"})
	//	return
	//}
	//if whether {
	//	err = setPrice(req, a.Phone)
	//	if err != nil {
	//		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	//		return
	//	}
	//	c.JSON(http.StatusOK, gin.H{"message": "Set price successfully"})
	//	return
	//}
	//c.JSON(http.StatusBadRequest, gin.H{"error": "Set price failed"})
}
func (a *Advisor) SetPrice(c *gin.Context) {
	var req SetPriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(401, gin.H{"error": "set price failed"})
		return
	}
	num := c.Param("id")
	var id, price, acc string
	cond, val, err := builder.BuildSelect("service", map[string]interface{}{"phone": a.Phone}, []string{"num_type", "price", "acceptance"})
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
	exist := false
	for rows.Next() {
		err = rows.Scan(&id, &price, &acc)
		if err != nil {
			c.JSON(401, gin.H{"error": "rows scan failed"})
			return
		}
		if num == id {
			exist = true
			if req.Price == "" {
				req.Price = price
			}
			if req.Acceptance == "" {
				req.Acceptance = acc
			}
			break
		}
	}
	if !exist {
		c.JSON(401, gin.H{"error": "the service type doesn't exist"})
	}

	err = truncate(&req.Price)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	where := map[string]interface{}{
		"phone":    a.Phone,
		"num_type": num,
	}
	update := map[string]interface{}{
		"price":      req.Price,
		"acceptance": req.Acceptance,
	}
	cond, val, err = builder.BuildUpdate("service", where, update)
	if err != nil {
		c.JSON(401, gin.H{"error": "build update failed"})
		return
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		c.JSON(401, gin.H{"error": "database execute failed"})
		return
	}
	//c.JSON(200, gin.H{"message": "set price successfully"})
	//c.Redirect(http.StatusMovedPermanently, "http://localhost:8080/advisor/auth/setprice")
	//c.Request.URL.Path = "http://localhost:8080/advisor/auth/setprice"
	a.DisplayService(c)
}

func truncate(price *string) error {
	p, err := strconv.ParseFloat(*price, 64)
	if err != nil {
		return errors.New("parse float failed")
	}
	if p < 3.0 || p > 36.0 {
		return errors.New("the price should be set between 3.0 and 36.0")
	}
	*price = fmt.Sprintf("%.1f", p)
	return nil
}

//
//func setPrice(req SetPriceRequest, phone string) error {
//	//req.Phone = a.Phone
//	where := map[string]interface{}{
//		"phone": phone,
//	}
//	var t, a, v, l string
//	selectFieled := []string{"textreading", "audioreading", "videoreading", "livetextchat"}
//	cond, val, err := builder.BuildSelect("price", where, selectFieled)
//	rows, err := db.Query(cond, val...)
//	if err != nil {
//		return errors.New("database query failed")
//	}
//	for rows.Next() {
//		err = rows.Scan(&t, &a, &v, &l)
//		if err != nil {
//			return errors.New("scan failed")
//		}
//	}
//	if req.LiveTextChat == "" {
//		req.LiveTextChat = l
//	}
//	if req.AudioReading == "" {
//		req.AudioReading = a
//	}
//	if req.TextReading == "" {
//		req.TextReading = t
//	}
//	if req.VideoReading == "" {
//		req.VideoReading = v
//	}
//	err = truncate(&req)
//	if err != nil {
//		return err
//	}
//	data := struct2Map(req)
//	where = map[string]interface{}{
//		"phone": phone,
//	}
//	cond, val, err = builder.BuildUpdate("price", where, data)
//	if err != nil {
//		return errors.New("set price error")
//	}
//	//fmt.Println("cond:", cond, "val:", val)
//	_, err = db.Exec(cond, val...)
//
//	if err != nil {
//		return errors.New("database execution error")
//	}
//	return nil
//}
//
//func truncate(req *SetPriceRequest) error {
//	t := reflect.TypeOf(*req)
//	v := reflect.ValueOf(req).Elem()
//	for i := 0; i < t.NumField(); i++ {
//		//fieldName := t.Field(i).Name
//		fieldVal := v.Field(i).String()
//		f, err := strconv.ParseFloat(fieldVal, 64)
//		if f <= 3.0 || f >= 36.0 {
//			return errors.New("the price should be set between 3.0 and 36.0")
//		}
//		if err != nil {
//			return errors.New("parse float failed")
//		}
//		fdecimal := fmt.Sprintf("%.1f", f)
//		v.Field(i).SetString(fdecimal)
//	}
//	return nil
//}
