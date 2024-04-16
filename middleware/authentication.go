package middleware

import (
	"advisorProject/config"
	"advisorProject/model"
	"database/sql"
	"errors"
	"fmt"
	"github.com/didi/gendry/builder"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
)

var db = config.InitDB()

func JwtUserAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		//The token has been stored into *gin.Context in the time when user logged in, now we can extract it from gin.Context
		token := c.Request.Header.Get("userToken")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is empty,please carry a token"})
			c.Abort()
			return
		}
		//fmt.Println("\n\n令牌：", token, "\n\n")
		claims, err := model.JWTParseUserToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"Error": err.Error()})
			c.Abort()
			return
		}
		c.Set("userClaims", claims) //store claims into gin.Context
	}
}

func JwtAdvisorAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		//The token has been stored into *gin.Context in the time when user logged in, now we can extract it from gin.Context
		token := c.Request.Header.Get("advisorToken")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is empty,please carry a token"})
			c.Abort()
			return
		}
		//fmt.Println("\n\n令牌：", token, "\n\n")
		claims, err := model.JWTParseAdvisorToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"Error": err.Error()})
			c.Abort()
			return
		}
		c.Set("advisorClaims", claims) //store claims into gin.Context
	}
}
func SaveAdvisor(ad *model.Visit) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.IsConsistent {
			c.Set("advisor", ad)
			c.Next()
		} else {
			c.JSON(401, gin.H{"error": "advisors are not consistent"})
			c.Abort()
		}
	}
}

func CheckExpiration() gin.HandlerFunc {
	return func(c *gin.Context) {
		var id, create string
		var expedite sql.NullString
		where := map[string]interface{}{"is_completed": "pending"}
		selectField := []string{"id", "create_time", "expedite"}
		cond, val, err := builder.BuildSelect("orders", where, selectField)
		if err != nil {
			c.JSON(401, gin.H{"error": "build select failed--middleware"})
			c.Abort()
		}
		rows, err := db.Query(cond, val...)
		if err != nil {
			c.JSON(401, gin.H{"error": "database query failed--middleware"})
			c.Abort()
		}
		for rows.Next() {
			err = rows.Scan(&id, &create, &expedite)
			if err != nil {
				c.JSON(401, gin.H{"error": "database query failed--middleware"})
				c.Abort()
			}
			t := time.Now()
			var duration time.Duration
			layout := "2006-01-02 15:04:05"
			format := t.Format(layout)
			currentTime, err := time.Parse(layout, format)
			if err != nil {
				c.JSON(401, gin.H{"error": "parse error"})
				c.Abort()
			}
			createTime, err := time.Parse(layout, create)

			//not expedited
			if !expedite.Valid {
				duration = currentTime.Sub(createTime)
				//err = judgeExpired(duration, 24*time.Hour, id)
				err = JudgeExpired(duration, 3*time.Minute, id)
				if err != nil {
					c.JSON(401, gin.H{"error": err.Error()})
					c.Abort()
				}
			}
			if expedite.Valid {
				expediteTime, err := time.Parse(layout, expedite.String)
				if err != nil {
					c.JSON(401, gin.H{"error": "parse expedite time error"})
					c.Abort()
				}

				//afterExpedite := expediteTime.Add(1 * time.Hour)
				//afterCreate := createTime.Add(24 * time.Hour)
				afterExpedite := expediteTime.Add(1 * time.Minute)
				afterCreate := createTime.Add(3 * time.Minute)

				//expedited in the last 1 hour
				if afterExpedite.Sub(afterCreate) > 0 {
					duration = currentTime.Sub(expediteTime)
					//err = judgeExpired(duration, 1*time.Hour, id)
					err = JudgeExpired(duration, 1*time.Minute, id)
					if err != nil {
						c.JSON(401, gin.H{"error": err.Error()})
						c.Abort()
					}
				} else { //expedited in the previous 23 hours
					duration = currentTime.Sub(createTime)
					//err = judgeExpired(duration, 24*time.Hour, id)
					err = JudgeExpired(duration, 3*time.Minute, id)
					if err != nil {
						c.JSON(401, gin.H{"error": err.Error()})
						c.Abort()
					}
				}
			}
		}
		c.Next()
	}
}

func JudgeExpired(duration time.Duration, permittedTime time.Duration, id string) error {
	if duration > permittedTime {
		cond, val, err := builder.BuildUpdate("orders", map[string]interface{}{"id": id}, map[string]interface{}{"is_completed": "expired"})
		if err != nil {
			return errors.New("build update failed")
		}
		_, err = db.Exec(cond, val...)
		if err != nil {
			return errors.New("database execute failed")
		}
	}
	return nil
}

func ReturnCoins() gin.HandlerFunc {
	return func(c *gin.Context) {
		var isExpedite sql.NullString
		var id, userPhone, cost string
		where := map[string]interface{}{"is_completed": "expired"}
		selectField := []string{"id", "user", "expedite"}
		cond, val, err := builder.BuildSelect("orders", where, selectField)
		if err != nil {
			c.JSON(401, gin.H{"error": "build select failed"})
			c.Abort()
		}
		rows, err := db.Query(cond, val...)
		if err != nil {
			c.JSON(401, gin.H{"error": "database query failed"})
			c.Abort()
		}
		for rows.Next() {
			err = rows.Scan(&id, &userPhone, &isExpedite)
			if err != nil {
				c.JSON(401, gin.H{"error": "rows scan failed"})
				c.Abort()
			}
			cost, err = totalCoinsNeedToReturn(id, isExpedite)
			if err != nil {
				c.JSON(401, gin.H{"error": err.Error()})
				c.Abort()
			}
			err = returnToUser(userPhone, cost)
			if err != nil {
				c.JSON(401, gin.H{"error": err.Error()})
				c.Abort()
			}
			err = updateOrderStatus(id)
			if err != nil {
				c.JSON(401, gin.H{"error": err.Error()})
				c.Abort()
			}
		}

		c.Next()
	}
}

func updateOrderStatus(id string) error {
	cond, val, err := builder.BuildUpdate("orders", map[string]interface{}{"id": id}, map[string]interface{}{"is_completed": "expired (refunded)"})
	if err != nil {
		return err
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return err
	}
	return nil
}

func returnToUser(phone string, cost string) error {
	var coin float64
	cond, val, err := builder.BuildSelect("user", map[string]interface{}{"phone": phone}, []string{"coin"})
	if err != nil {
		return err
	}
	rows, err := db.Query(cond, val...)
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(&coin)
		if err != nil {
			return err
		}
		coinReturn, err := strconv.ParseFloat(cost, 64)
		if err != nil {
			return err
		}
		coin += coinReturn
	}
	cond, val, err = builder.BuildUpdate("user", map[string]interface{}{"phone": phone}, map[string]interface{}{"coin": coin})
	if err != nil {
		return err
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return err
	}

	return nil
}

func totalCoinsNeedToReturn(id string, isExpedite sql.NullString) (string, error) {
	var cost string
	cond, val, err := builder.BuildSelect("orders_info", map[string]interface{}{"id": id}, []string{"cost"})
	if err != nil {
		return "", err
	}
	rows, err := db.Query(cond, val...)
	if err != nil {
		return "", err
	}
	if rows.Next() {
		err = rows.Scan(&cost)
		if err != nil {
			return "", err
		}
	}
	s, err := strconv.ParseFloat(cost, 64)
	if err != nil {
		return "", err
	}
	c := fmt.Sprintf("%.1f", s*3/2)
	if isExpedite.Valid {
		return c, nil
	}
	return cost, nil
}

func ReturnExpediteCoins() gin.HandlerFunc {
	return func(c *gin.Context) {
		var id, user string
		var cost float64
		var isExpedite sql.NullString
		where := map[string]interface{}{"is_completed": "pending"}
		selectField := []string{"id", "user", "expedite"}
		cond, val, err := builder.BuildSelect("orders", where, selectField)
		if err != nil {
			c.JSON(401, gin.H{"error": err.Error()})
			c.Abort()
		}
		rows, err := db.Query(cond, val...)
		if err != nil {
			c.JSON(401, gin.H{"error": err.Error()})
			c.Abort()
		}
		for rows.Next() {
			err = rows.Scan(&id, &user, &isExpedite)
			if err != nil {
				c.JSON(401, gin.H{"error": err.Error()})
				c.Abort()
			}
			if !isExpedite.Valid {
				continue
			}
			isExpired, err := isExpediteExpired(id, isExpedite.String)
			if err != nil {
				c.JSON(401, gin.H{"error": err.Error()})
				c.Abort()
			}
			if !isExpired {
				continue
			}
			cost, err = OrderCost(id)
			if err != nil {
				c.JSON(401, gin.H{"error": err.Error()})
				c.Abort()
			}
			err = returnCoinToUser(user, cost)
			if err != nil {
				c.JSON(401, gin.H{"error": err.Error()})
				c.Abort()
			}
			err = setExpediteNull(id)
			if err != nil {
				c.JSON(401, gin.H{"error": err.Error()})
				c.Abort()
			}
		}
		c.Next()
	}
}

func setExpediteNull(id string) error {
	cond, val, err := builder.BuildUpdate("orders", map[string]interface{}{"id": id}, map[string]interface{}{"expedite": sql.NullString{}})
	if err != nil {
		return err
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return err
	}
	return nil
}

func returnCoinToUser(phone string, cost float64) error {
	var coin float64
	cond, val, err := builder.BuildSelect("user", map[string]interface{}{"phone": phone}, []string{"coin"})
	if err != nil {
		return err
	}
	rows, err := db.Query(cond, val...)
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(&coin)
		if err != nil {
			return err
		}
	}
	coin = coin + cost/2
	afterReturn := fmt.Sprintf("%.1f", coin)
	cond, val, err = builder.BuildUpdate("user", map[string]interface{}{"phone": phone}, map[string]interface{}{"coin": afterReturn})
	if err != nil {
		return err
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return err
	}
	return nil
}

func OrderCost(id string) (float64, error) {
	var cost float64
	cond, val, err := builder.BuildSelect("orders_info", map[string]interface{}{"id": id}, []string{"cost"})
	if err != nil {
		return 0, err
	}
	rows, err := db.Query(cond, val...)
	if err != nil {
		return 0, err
	}
	if rows.Next() {
		err = rows.Scan(&cost)
		if err != nil {
			return 0, err
		}
	}
	return cost, nil
}

func isExpediteExpired(id string, expedite string) (bool, error) {
	current := time.Now()
	layout := "2006-01-02 15:04:05"
	f := current.Format(layout)
	currentTime, err := time.Parse(layout, f)
	if err != nil {
		return false, err
	}
	expediteTime, err := time.Parse(layout, expedite)
	if err != nil {
		return false, err
	}
	duration := currentTime.Sub(expediteTime)
	//if duration > 1*time.Hour {
	if duration > 1*time.Minute {
		return true, nil
	}
	return false, nil
}
