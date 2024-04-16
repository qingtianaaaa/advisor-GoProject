package model

import (
	"advisorProject/config"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/didi/gendry/builder"
	"github.com/didi/gendry/scanner"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"net/http"
	"reflect"
	"strconv"
	"time"
)

// User
type User struct {
	ID     int    `json:"id"`
	Phone  string `json:"phone"`
	Name   string `json:"name"`
	Birth  string `json:"birth"`
	Gender string `json:"gender"`
	Bio    string `json:"bio"`
	About  string `json:"about"`
	Coin   string `json:"coin"`
}
type RegisterRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}
type UpdateRequest struct {
	Phone  string `json:"phone"`
	Name   string `json:"name"`
	Birth  string `json:"birth"`
	Gender string `json:"gender"`
	Bio    string `json:"bio"`
	About  string `json:"about"`
}
type showService struct {
	Num         string
	ServiceType string
	Price       string
	Consult     string
}

type UserReviewDisplay struct {
	OrderId     int
	OrderRating string
	ServiceType string
	//UserReviewTime string
	//UserReview     string
	AllReview []ReviewDisplay
}

type ReviewDisplay struct {
	Name       string
	Review     string
	ReviewTime string
}

type Visit struct {
	Phone    string
	Name     string
	Tag      string
	About    string
	Rating   string
	Reviews  string
	Readings float64
	OnTime   string
	Collect  string
	Service  []showService
}

var db = config.InitDB()

// var Ad = &Advisor{}
var Ad = &Visit{}

// Login Written by lwj
// Function:Realize login function
func (u *User) Login(c *gin.Context) {
	config.IsConsistent = false
	var req RegisterRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	if req.Phone == "" || len(req.Phone) != 11 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please enter 11-digit phone number"})
		return
	}
	if req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password is not provided"})
		return
	}
	exist, err := isExist("register", req.Phone)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Login failed"})
		return
	}
	if exist {
		err = loginUser(req, c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		} else {
			//c.JSON(http.StatusOK, gin.H{"message": "Login successfully"})
			updateRealTimeUser(req.Phone, u)
			return
		}
	} else {
		exist, err = isExist("ad_register", req.Phone)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if exist {
			c.JSON(http.StatusBadRequest, gin.H{"error": "This is a advisor account"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "This number is not registered.Please register first"})
		}
		return
	}
}

// Written by lwj
// Function: Update information of user in real-time (when sign in and updating)
func updateRealTimeUser(phone string, usr *User) {
	where := map[string]interface{}{
		"phone": phone,
	}
	cond, val, err := builder.BuildSelect("user", where, []string{"*"})
	if err != nil {
		panic(err.Error())
	}
	rows, err := db.Query(cond, val...)
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()
	err = scanner.Scan(rows, usr)
	if err != nil {
		panic(err.Error())
	}
}

// Written by lwj
// Function:Realize login function,using token to authenticate
func loginUser(login RegisterRequest, c *gin.Context) error {
	correct, err := isCorrect("register", login)
	if err != nil {

		return errors.New("wrong password")
	}
	if correct {
		//if a user login successfully, then generate a token to the user
		var user User
		user, err = getUser(login)

		jwtStr := JWTGenerateUserToken(user, time.Hour)
		c.JSON(http.StatusOK, gin.H{"Token": jwtStr, "ID": user.ID, "Name": user.Name, "Phone": user.Phone, "message": "Login successfully"})
		c.Set("token", jwtStr) //store token into *gin.context,making it more convenient for middleware to check the token validity

		return nil
	}
	return errors.New("login failed")
}

// Written by lwj
// Function:Get user information correspond to the login phone number
func getUser(login RegisterRequest) (User, error) {
	var user User
	where := map[string]interface{}{
		"phone": login.Phone,
	}
	cond, val, err := builder.BuildSelect("user", where, []string{"*"})
	if err != nil {
		return user, err
	}
	//fmt.Println("cond:", cond)
	//fmt.Println("val", val)
	rows, err := db.Query(cond, val...)
	if err != nil {
		return user, err
	}
	defer rows.Close()
	err = scanner.Scan(rows, &user) //read rows into user
	if err != nil {
		return user, err
	}
	fmt.Println(user)
	return user, nil

}

func isCorrect(table string, login RegisterRequest) (bool, error) {
	where := map[string]interface{}{
		"phone": login.Phone,
	}
	cond, val, err := builder.BuildSelect(table, where, []string{"password"})
	if err != nil {
		return false, err
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var password string
		if err = rows.Scan(&password); err != nil {
			return false, err
		}
		if password == login.Password {
			return true, nil
		}
	}
	return false, errors.New("wrong password")
}

// Update Written:by lwj
// Function: Realize update function
func (u *User) Update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claims, have := c.Get("userClaims")
	if !have {
		c.JSON(http.StatusBadRequest, gin.H{"error": "claims don't exist"})
		return
	}

	claimData := claims.(*UserClaims)
	usr := claimData.User

	exist, err := isExist("register", usr.Phone)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if exist {
		err := updateUser(*u, &req)
		fmt.Println(u)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Phone == "" || req.Phone == u.Phone {
			c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
		} else {
			c.JSON(http.StatusOK, gin.H{"message": "Change binding phone number successfully,please login again"})
		}
		fmt.Println("----req.phone", req.Phone)
		updateRealTimeUser(req.Phone, u)
		return
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid number,please login again"})
		return
	}
}

// Written:by lwj
// Function: Check the number whether has been registered or not
// Return:error
func isExist(table string, phone string) (bool, error) {
	where := map[string]interface{}{
		"phone": phone,
	}
	cond, val, err := builder.BuildSelect(table, where, []string{"phone"})
	if err != nil {
		return false, errors.New("build select failed--isexist")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return false, errors.New("database query failed--isExist")
	}
	if rows.Next() {
		return true, nil
	}
	return false, nil
}

// Written:by lwj
// Function:Update user information
// Return:error
func updateUser(usr User, req *UpdateRequest) error {
	if req.Phone == "" {
		req.Phone = usr.Phone
	}
	if len(req.Phone) != 11 {
		return errors.New("please enter 11-digit number if you want to change you binding phone")
	}
	if req.Phone != usr.Phone {
		err := changePhoneNumber("register", usr.Phone, req.Phone)
		if err != nil {
			return err
		}
	}

	if req.Name == "" {
		req.Name = usr.Name
	}
	if req.Bio == "" {
		req.Bio = usr.Bio
	}
	if req.About == "" {
		req.About = usr.About
	}
	if req.Birth == "" {
		req.Birth = usr.Birth
	}
	if req.Gender == "" {
		req.Gender = usr.Gender
	}
	data := struct2Map(*req)
	where := map[string]interface{}{
		"phone": req.Phone,
	}
	cond, val, err := builder.BuildUpdate("user", where, data)
	if err != nil {
		return errors.New("build update failed")
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return errors.New("database execution failed")
	}
	return nil
}

func changePhoneNumber(updTab, phBefore, phAfter string) error {
	no, err := isRegistered("register", phAfter)
	if err != nil {
		return errors.New("this number has been registered,please provide a different " +
			"number if you want to change your binding number")
	}
	no, err = isRegistered("ad_register", phAfter)
	if err != nil {
		return errors.New("this number has been registered,please provide a different " +
			"number if you want to change your binding number")
	}
	if no { //dont register  ,then change the number
		where := map[string]interface{}{"phone": phBefore}
		update := map[string]interface{}{"phone": phAfter}
		cond, val, err := builder.BuildUpdate(updTab, where, update)
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

// Register Written: by lwj
// Function: Realize register function
func (u *User) Register(c *gin.Context) {

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil { //ShouldBindJSON用于POST请求，从请求中读取json数据 并将其解析并填充到传入的
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) //结构体变量中
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
	whether, err := isRegistered("register", req.Phone)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	whether, err = isRegistered("ad_register", req.Phone)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if whether {
		err = insertRegis("register", "user", req, c)
	}
	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully!Please login!"})
}

// by lwj
// Function: Judge the phone number whether has been registered or not
// Return: 1.true:has not been registered. false:has been registered  2.error
func isRegistered(table string, phone string) (bool, error) {
	where := map[string]interface{}{"phone": phone}
	cond, val, err := builder.BuildSelect(table, where, []string{"phone"})
	if err != nil {
		return false, errors.New("build select failed--isregistered")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return false, errors.New("database query failed--isRegistered")
	}

	if rows.Next() {
		return false, errors.New("number has been registered")
	}
	return true, nil
}

// by lwj
// Function:Insert phone number into database (into table Register and table User)
// Return: error
func insertRegis(table1 string, table2 string, regis RegisterRequest, c *gin.Context) error {
	data := struct2Map(regis)
	//insert into table register or ad_register
	cond, val, err := builder.BuildInsert(table1, []map[string]interface{}{data})
	if err != nil {
		return err
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return err
	}
	//insert into table user or advisor
	cond, val, err = builder.BuildInsert(table2, []map[string]interface{}{{"phone": regis.Phone}})

	if err != nil {
		return err
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return err
	}
	return nil
}

// by lwj
// Function:Convert struct type to map type, making building mysql select sentence more convenient
// return: none
func struct2Map(s interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	t := reflect.TypeOf(s)
	v := reflect.ValueOf(s)
	for i := 0; i < v.NumField(); i++ {
		ft := t.Field(i)
		fv := v.Field(i)
		res[ft.Name] = fv.Interface()
	}
	return res
}

func (u *User) Visit(c *gin.Context) {

	id := c.Param("id")
	userReview := &[]UserReviewDisplay{}
	err := searchAd(Ad, id, userReview, u)

	if err != nil {
		if err.Error() == "the advisor have no orders yet" {
			config.IsConsistent = true
			c.JSON(200, gin.H{"advisor": Ad})
			return
		}
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}

	config.IsConsistent = true
	if len(*userReview) == 0 {
		c.JSON(200, gin.H{"advisor": Ad})
		return
	}
	c.JSON(200, gin.H{"advisor": Ad, "reviews": *userReview})
	//c.JSON(200, gin.H{"advisor": Ad, "reviews": userReview})

	return
}

func searchAd(v *Visit, id string, userReview *[]UserReviewDisplay, u *User) error {
	v.Service = []showService{}
	var service showService
	var status string
	var acc string
	rating := 0.0
	n, _ := strconv.Atoi(id)
	_, err := isIdExist(n)
	if err != nil {
		return err
	}
	selectField := []string{"name", "bio", "about", "phone", "status", "rating", "sum_of_comments", "total_order", "complete_order"}
	cond, val, err := builder.BuildSelect("advisor", map[string]interface{}{"id": id}, selectField)
	if err != nil {
		return errors.New("build select failed--1--searchAd")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		fmt.Println(err.Error())
		return errors.New("database query failed--searchAd")
	}
	if rows.Next() {
		var completed float64
		err = rows.Scan(&v.Name, &v.Tag, &v.About, &v.Phone, &status, &rating, &v.Reviews, &v.Readings, &completed)
		v.Collect = "http://localhost:8080/user/auth/collect/" + id
		if v.Readings == 0 {
			v.OnTime = "0.0%"
		} else {
			v.OnTime = fmt.Sprintf("%.1f", completed/v.Readings*100) + "%"
		}

		if err != nil {
			return errors.New("rows scan failed--1")
		}
		v.Rating = fmt.Sprintf("%.1f", rating)
	} else {
		return errors.New("advisor id doesn't exist")
	}
	if status == "out-of-service" {
		return errors.New("the advisor is out of service")
	}
	selectField = []string{"service_type", "price", "acceptance", "num_type"}
	cond, val, err = builder.BuildSelect("service", map[string]interface{}{"phone": v.Phone}, selectField)
	if err != nil {
		return errors.New("build select failed--2--searchAd")
	}
	rows, err = db.Query(cond, val...)
	if err != nil {
		return errors.New("database query failed--2--searchAd")
	}

	for rows.Next() {
		err = rows.Scan(&service.ServiceType, &service.Price, &acc, &service.Num)
		if err != nil {
			return errors.New("rows scan failed--2")
		}
		if acc == "ON" {
			service.Consult = "http://localhost:8080/user/auth/creat/" + service.Num
			v.Service = append(v.Service, service)
		}
	}
	idSlice := []string{}
	i := ""
	cond, val, err = builder.BuildSelect("review", map[string]interface{}{"phone": v.Phone}, []string{"id"})
	if err != nil {
		return errors.New("build select failed")
	}
	rows, err = db.Query(cond, val...)
	if err != nil {
		return errors.New("database query failed")
	}
	for rows.Next() {
		err = rows.Scan(&i)
		if err != nil {
			return errors.New("rows scan failed")
		}
		idSlice = append(idSlice, i)
	}
	if len(idSlice) == 0 {
		return errors.New("the advisor have no orders yet")
	}
	review := UserReviewDisplay{}
	for _, vl := range idSlice {
		reviewJson, err := Rdb.Get(Ctx, fmt.Sprintf("review:%s", vl)).Result()
		if errors.Is(err, redis.Nil) {
			fmt.Println("=======")
			fmt.Println("review:" + vl + "在redis中不存在，序列化后存入redis")
			fmt.Println("=======")
			review.AllReview = []ReviewDisplay{}
			err = AddReview(vl, u, &review)
			if err != nil {
				return err
			}
			*userReview = append(*userReview, review)

			reviewData, err := json.Marshal(review)
			if err != nil {
				return errors.New("json marshal failed")
			}
			err = Rdb.Set(Ctx, fmt.Sprintf("review:%s", vl), reviewData, 3*24*time.Hour).Err()
			if err != nil {
				return errors.New("redis set failed")
			}
		} else if err != nil {
			return err
		} else {
			fmt.Println("=======")
			fmt.Println("review:" + vl + "在redis中存在，从缓存取出")
			fmt.Println("=======")
			err = json.Unmarshal([]byte(reviewJson), &review)
			if err != nil {
				return errors.New("json unmarshal failed")
			}
			*userReview = append(*userReview, review)
		}
	}
	//inRedis, err := isInRedis(userReview, v.Phone)
	//err = AddReview(idSlice, userReview, u)
	//if err != nil {
	//	return err
	//}
	return nil
}

func AddReview(id string, u *User, userReview *UserReviewDisplay) error {
	review := ReviewDisplay{}
	cond, val, err := builder.BuildSelect("answer", map[string]interface{}{"order_id": id, "_orderby": "review_time"}, []string{"phone", "review", "review_time"})
	if err != nil {
		return errors.New("build select failed")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return errors.New("database query failed")
	}
	var reviewTime, phone, rvw string
	layout := "2006-01-02 15:04:05"
	for rows.Next() {
		err = rows.Scan(&phone, &rvw, &reviewTime)
		if err != nil {
			return errors.New("rows scan failed")
		}
		t, err := time.Parse(layout, reviewTime)
		if err != nil {
			return errors.New("time parse failed")
		}
		rvwTime := t.Format("Jan 02,2006 15:04:05")
		err = obtainReview(userReview, id)
		if err != nil {
			return err
		}
		isuser, err := isUser(phone)
		if err != nil {
			return err
		}
		if isuser {
			review.Name, err = obtain("user", phone)
			if err != nil {
				return err
			}
			if phone == u.Phone {
				review.Name = review.Name + "(Me)"
			}
			review.Review = rvw
			review.ReviewTime = rvwTime
		} else {
			review.Name, err = obtain("advisor", phone)
			review.Name = review.Name + "(Advisor)"
			review.Review = rvw
			review.ReviewTime = rvwTime
		}
		userReview.AllReview = append(userReview.AllReview, review)

	}
	return nil
}

func obtain(table string, phone string) (name string, err error) {
	cond, val, err := builder.BuildSelect(table, map[string]interface{}{"phone": phone}, []string{"name"})
	if err != nil {
		return
	}
	rows, err := db.Query(cond, val...)
	if err != nil {
		return
	}
	if rows.Next() {
		err = rows.Scan(&name)
		if err != nil {
			return
		}
	}
	return
}
func isUser(phone string) (bool, error) {
	cond, val, err := builder.BuildSelect("user", map[string]interface{}{"phone": phone}, []string{"*"})
	if err != nil {
		return false, errors.New("build select failed--isUser")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return false, errors.New("database query failed")
	}
	if rows.Next() {
		return true, nil
	}
	return false, nil
}

func obtainReview(userReview *UserReviewDisplay, orderId string) error {

	cond, val, err := builder.BuildSelect("orders", map[string]interface{}{"id": orderId}, []string{"type"})
	if err != nil {
		return errors.New("build select failed-1-obtainReview")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return errors.New("database query failed")
	}
	if rows.Next() {
		err = rows.Scan(&userReview.ServiceType)
		if err != nil {
			return errors.New("rows scan failed-3")
		}
	}

	cond, val, err = builder.BuildSelect("review", map[string]interface{}{"id": orderId}, []string{"rating"})
	if err != nil {
		return errors.New("build select failed-2-obtainReview")
	}
	rows, err = db.Query(cond, val...)
	if err != nil {
		return errors.New("database query failed")
	}
	if rows.Next() {
		err = rows.Scan(&userReview.OrderRating)
		if err != nil {
			return errors.New("rows scan failed-4")
		}
	}
	userReview.OrderId, err = strconv.Atoi(orderId)
	if err != nil {
		return errors.New("convert failed")
	}
	return nil

}

//	func AddReview(idSlice []string, userReviews *[]UserReviewDisplay, u *User) error {
//		userReview := UserReviewDisplay{}
//		review := &ReviewDisplay{}
//
//		cond, val, err := builder.BuildSelect("answer", map[string]interface{}{"order_id": idSlice, "_orderby": "order_id,review_time"}, []string{"order_id", "phone", "review", "review_time"})
//		if err != nil {
//			return errors.New("build select failed-2-AddReview")
//		}
//		rows, err := db.Query(cond, val...)
//		defer rows.Close()
//		if err != nil {
//			return errors.New("database query failed")
//		}
//		var orderIdBefore, orderIdAfter, reviewTime, phone, rvw string
//		layout := "2006-01-02 15:04:05"
//		for rows.Next() {
//			err = rows.Scan(&orderIdAfter, &phone, &rvw, &reviewTime)
//			if err != nil {
//				fmt.Println(err.Error())
//				return errors.New("rows scan failed-6")
//			}
//			if orderIdAfter != orderIdBefore {
//				*userReviews = append(*userReviews, userReview)
//				userReview.AllReview = []ReviewDisplay{}
//				err = obtainReview(&userReview, orderIdAfter)
//				if err != nil {
//					return err
//				}
//			}
//
//			t, err := time.Parse(layout, reviewTime)
//			if err != nil {
//				return errors.New("time parse failed")
//			}
//			rvwTime := t.Format("Jan 02,2006 15:04:05")
//			isuser, err := isUser(phone)
//			if err != nil {
//				return err
//			}
//			if isuser {
//				cond, val, err = builder.BuildSelect("user", map[string]interface{}{"phone": phone}, []string{"name"})
//				if err != nil {
//					return errors.New("build select failed-3-AddReview")
//				}
//				rows1, err := db.Query(cond, val...)
//				if err != nil {
//					return errors.New("database query failed")
//				}
//				if rows1.Next() {
//					err = rows1.Scan(&review.Name)
//					if err != nil {
//						return errors.New("rows1 scan failed")
//					}
//				}
//				if phone == u.Phone {
//					review.Name = review.Name + "(Me)"
//				}
//				review.Review = rvw
//				review.ReviewTime = rvwTime
//			} else {
//				cond, val, err = builder.BuildSelect("advisor", map[string]interface{}{"phone": phone}, []string{"name"})
//				if err != nil {
//					return errors.New("build select failed-4-AddReview")
//				}
//				rows1, err := db.Query(cond, val...)
//				if err != nil {
//					return errors.New("database query failed")
//				}
//				if rows1.Next() {
//					err = rows1.Scan(&review.Name)
//					if err != nil {
//						return errors.New("rows1 scan failed")
//					}
//				}
//				review.Name = review.Name + "(Advisor)"
//				review.Review = rvw
//				review.ReviewTime = rvwTime
//			}
//			userReview.AllReview = append(userReview.AllReview, *review)
//			orderIdBefore = orderIdAfter
//		}
//		*userReviews = append(*userReviews, userReview)
//		return nil
//	}
func isIdExist(id int) (bool, error) {
	cond, val, err := builder.BuildSelect("advisor", map[string]interface{}{"id": id}, []string{"id"})
	if err != nil {
		return false, errors.New("build select failed--isIdExist")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return false, errors.New("database query failed--isIdExist")
	}
	if rows.Next() {
		return true, nil
	}
	return false, errors.New("the advisor does not exist")
}

func (u *User) Cancel(c *gin.Context) {
	id := c.Param("id")
	var collectId, status string
	exist, err := isAdIdExist(u.Phone, id, &collectId, &status)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	if !exist {
		c.JSON(401, gin.H{"error": "advisor doesn't exist"})
		return
	}
	if status == "n" {
		c.JSON(401, gin.H{"error": "you haven't collected this advisor,so you can't cancel collection"})
		return
	}
	cond, val, err := builder.BuildUpdate("collect", map[string]interface{}{"id": collectId}, map[string]interface{}{"collected": "n"})
	if err != nil {
		c.JSON(401, gin.H{"error": "build update failed"})
		return
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		c.JSON(401, gin.H{"eroor": "database execute failed"})
		return
	}
	c.JSON(200, gin.H{"msg": "cancel collection successfully"})
}

func isAdIdExist(userphone, id string, collectId, status *string) (bool, error) {
	var ph, adPh string

	cond, val, err := builder.BuildSelect("advisor", map[string]interface{}{"id": id}, []string{"phone"})
	if err != nil {
		return false, errors.New("build select failed")
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return false, errors.New("database query failed")
	}
	if rows.Next() {
		err = rows.Scan(&ph)
		if err != nil {
			return false, errors.New("rows scan failed")
		}
	}
	cond, val, err = builder.BuildSelect("collect", map[string]interface{}{"user": userphone}, []string{"advisor", "id", "collected"})
	if err != nil {
		return false, errors.New("build select failed")
	}
	rows, err = db.Query(cond, val...)
	if err != nil {
		return false, errors.New("database query failed")
	}
	if rows.Next() {
		err = rows.Scan(&adPh, collectId, status)
		if err != nil {
			return false, errors.New("rows scan failed")
		}
	}
	if adPh == ph {
		return true, nil
	}
	return false, nil
}
