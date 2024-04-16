package model

import (
	"database/sql"
	"fmt"
	"github.com/didi/gendry/builder"
	"log"
	"time"
)

func CronJob() {
	IsExpired()
	IsExpedite()
}

func IsExpired() {
	var id, phone, create string
	var expedite sql.NullString
	where := map[string]interface{}{"is_completed": "pending"}
	selectField := []string{"id", "user", "create_time", "expedite"}
	cond, val, err := builder.BuildSelect("orders", where, selectField)
	if err != nil {
		log.Println("build select failed")
		return
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		log.Println("database query failed")
		return
	}
	for rows.Next() {
		err = rows.Scan(&id, &phone, &create, &expedite)
		t := time.Now()
		var duration time.Duration
		layout := "2006-01-02 15:04:05"
		format := t.Format(layout)
		currentTime, err := time.Parse(layout, format)
		if err != nil {
			log.Println("parse error")
			return
		}
		createTime, err := time.Parse(layout, create)
		//not expedited
		if !expedite.Valid {
			duration = currentTime.Sub(createTime)
			fmt.Println("未加速，3min超时归还下单金币：", duration)
			//if duration > 24*time.Hour {
			if duration > 3*time.Minute {
				err = processStatusAndCoins(id, false, phone)
				if err != nil {
					log.Println(err.Error())
					return
				}
			}
		}
		if expedite.Valid {
			expediteTime, err := time.Parse(layout, expedite.String)
			if err != nil {
				log.Println(err.Error())
				return
			}
			//afterExpedite := expediteTime.Add(1 * time.Hour)
			//afterCreate := createTime.Add(24 * time.Hour)
			afterExpedite := expediteTime.Add(1 * time.Minute)
			afterCreate := createTime.Add(3 * time.Minute)
			//expedite in the last 1 hour
			if afterExpedite.Sub(afterCreate) > 0 {

				duration = currentTime.Sub(expediteTime)
				fmt.Println("最后1分钟加速，加速1min到期后归还下单和加急的总共金币:", duration)
				//if duration > 1*time.Hour {
				if duration > 1*time.Minute {
					err = processStatusAndCoins(id, true, phone)
					if err != nil {
						log.Println(err.Error())
						return
					}
				}
			} else { //expedite in the previous 23 hours
				duration = currentTime.Sub(createTime)
				fmt.Println("前2分钟加速,1min后归还加急金币：", currentTime.Sub(expediteTime))
				//if duration > 24*time.Hour {
				if duration > 3*time.Minute {
					err = processStatusAndCoins(id, false, phone)
					if err != nil {
						log.Println(err.Error())
						return
					}
				}
			}
		}
	}
}
func processStatusAndCoins(id string, inLastMinute bool, userPhone string) error {
	cost := 0.0
	returnCoin := 0.0
	cond, val, err := builder.BuildSelect("orders_info", map[string]interface{}{"id": id}, []string{"cost"})
	if err != nil {
		return err
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(&cost)
		if err != nil {
			return err
		}
	}

	err = updateStatus(id)
	if err != nil {
		return err
	}
	//err = obtainCost(id, &cost)
	//if err != nil {
	//	return err
	//}
	returnCoin = cost
	if inLastMinute {
		returnCoin = cost + cost/2
	}

	current := time.Now()
	layout := "2006-01-02 15:04:05"
	currentTime := current.Format(layout)
	data := map[string]interface{}{
		"phone":   userPhone,
		"type":    "Order Expired Refund",
		"credits": "+" + fmt.Sprintf("%.1f", cost),
		"time":    currentTime,
	}
	cond, val, err = builder.BuildInsert("transactions", []map[string]interface{}{data})
	if err != nil {
		return err
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return err
	}

	if inLastMinute {
		current = time.Now()
		currentTime = current.Format(layout)
		data = map[string]interface{}{
			"phone":   userPhone,
			"type":    "Speed-up Expired Refund",
			"credits": "+" + fmt.Sprintf("%.1f", cost/2),
			"time":    currentTime,
		}
		cond, val, err = builder.BuildInsert("transactions", []map[string]interface{}{data})
		if err != nil {
			return err
		}
		_, err = db.Exec(cond, val...)
		if err != nil {
			return err
		}
	}

	err = updateUserCoin(userPhone, returnCoin)
	if err != nil {
		return err
	}
	return nil
}

func updateUserCoin(phone string, returnCoins float64) error {
	original := 0.0
	cond, val, err := builder.BuildSelect("user", map[string]interface{}{"phone": phone}, []string{"coin"})
	if err != nil {
		return err
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(&original)
		if err != nil {
			return err
		}
	}
	refund := original + returnCoins
	fmt.Println("归还下单金币:", returnCoins)
	fmt.Println("归还后金币:", refund)
	cond, val, err = builder.BuildUpdate("user", map[string]interface{}{"phone": phone}, map[string]interface{}{"coin": refund})
	if err != nil {
		return err
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return err
	}

	return nil
}

func updateStatus(id string) error {
	cond, val, err := builder.BuildUpdate("orders", map[string]interface{}{"id": id}, map[string]interface{}{"is_completed": "expired"})
	if err != nil {
		return err
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return err
	}
	return nil
}
func obtainCost(id string, cost *float64) error {
	cond, val, err := builder.BuildSelect("orders_info", map[string]interface{}{"id": id}, []string{"cost"})
	if err != nil {
		return err
	}
	rows, err := db.Query(cond, val...)
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(cost)
		if err != nil {
			return err
		}
	}
	return nil
}

func IsExpedite() {
	var expedite sql.NullString
	var create string
	var id, userPhone string
	cond, val, err := builder.BuildSelect("orders", map[string]interface{}{"is_completed": "pending"}, []string{"id", "user", "expedite", "create_time"})
	if err != nil {
		log.Println(err.Error())
		return
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		log.Println(err.Error())
		return
	}
	for rows.Next() {
		err = rows.Scan(&id, &userPhone, &expedite, &create)
		if err != nil {
			log.Println(err.Error())
			return
		}
		if !expedite.Valid {
			continue
		}
		layout := "2006-01-02 15:04:05"
		expediteTime, err := time.Parse(layout, expedite.String)

		if err != nil {
			log.Println(err.Error())
			return
		}
		current := time.Now()
		transform := current.Format(layout)
		currentTime, err := time.Parse(layout, transform)
		if err != nil {
			log.Println(err.Error())
			return
		}
		//if currentTime.Sub(expediteTime) > 1*time.Hour {
		if currentTime.Sub(expediteTime) > 1*time.Minute {
			fmt.Println("返还加急金币执行了")
			err = returnExpediteCoins(id, userPhone)
			if err != nil {
				log.Println(err.Error())
				return
			}
			createTime, err := time.Parse(layout, create)
			if err != nil {
				log.Println(err.Error())
				return
			}
			afterCreate := createTime.Add(3 * time.Minute)
			afterExpedite := expediteTime.Add(1 * time.Minute)
			if afterExpedite.Sub(afterCreate) < 0 {
				err = updateExpediteStatus(id)
				if err != nil {
					log.Println(err.Error())
					return
				}
			}
		}
	}
}

func updateExpediteStatus(id string) error {
	fmt.Println("修改expedite执行了")
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

func returnExpediteCoins(id, phone string) error {
	originalCoin := 0.0
	cost := 0.0
	cond, val, err := builder.BuildSelect("user", map[string]interface{}{"phone": phone}, []string{"coin"})
	if err != nil {
		return err
	}
	rows, err := db.Query(cond, val...)
	defer rows.Close()
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(&originalCoin)
		if err != nil {
			return err
		}
	}
	cond, val, err = builder.BuildSelect("orders_info", map[string]interface{}{"id": id}, []string{"cost"})
	if err != nil {
		return err
	}
	rows, err = db.Query(cond, val...)
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(&cost)
		if err != nil {
			return err
		}
	}
	returnCoins := originalCoin + cost/2
	fmt.Println("加急订单之后的coin:", originalCoin)
	fmt.Println("订单价", cost, ",归还一半")
	fmt.Println("加急订单过期金币归还后coin:", returnCoins)
	cond, val, err = builder.BuildUpdate("user", map[string]interface{}{"phone": phone}, map[string]interface{}{"coin": returnCoins})
	if err != nil {
		return err
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return err
	}
	current := time.Now()
	layout := "2006-01-02 15:04:05"
	currentTime := current.Format(layout)
	data := map[string]interface{}{
		"phone":   phone,
		"type":    "Speed-up Expired Refund",
		"credits": "+" + fmt.Sprintf("%.1f", cost/2),
		"time":    currentTime,
	}
	cond, val, err = builder.BuildInsert("transactions", []map[string]interface{}{data})
	if err != nil {
		return err
	}
	_, err = db.Exec(cond, val...)
	if err != nil {
		return err
	}
	return nil
}
