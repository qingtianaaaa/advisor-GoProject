package model

import (
	"context"
	"github.com/go-redis/redis/v8"
)

var Rdb *redis.Client

type Reviews struct {
	OrderId     string
	Rating      string
	User        string
	Advisor     string
	ServiceType string
	AllReview   []ReviewDisplay
}

var Ctx = context.Background()

//
//func Cache(rdb *redis.Client) {
//
//	CacheOrderInfo(rdb)
//	CacheOrderReview(rdb)
//}
//func CacheOrderInfo(rdb *redis.Client) {
//	orderInfo := OrderConcreteInfo{}
//	var userPhone, create, delivery, birth string
//
//	cond, val, err := builder.BuildSelect("orders", map[string]interface{}{"is_completed": []string{"expired", "completed"}}, []string{"id", "user", "type", "is_completed", "create_time"})
//	if err != nil {
//		log.Println("build select failed")
//		return
//	}
//	rows, err := db.Query(cond, val...)
//	if err != nil {
//		log.Println("database query failed--1")
//		return
//	}
//	layout := "2006-01-02 15:04:05"
//	for rows.Next() {
//		err = rows.Scan(&orderInfo.OrderId, &userPhone, &orderInfo.Type, &orderInfo.IsCompleted, &create)
//		if err != nil {
//			log.Println("rows scan failed")
//			return
//		}
//		t, err := time.Parse(layout, create)
//		if err != nil {
//			log.Println("time parse failed")
//			return
//		}
//		orderInfo.OrderTime = t.Format("Jan 2,2006 15:04:05")
//
//		cond, val, err = builder.BuildSelect("reply", map[string]interface{}{"id": orderInfo.OrderId}, []string{"message", "reply_time"})
//		if err != nil {
//			log.Println("build select failed")
//			return
//		}
//		rows1, err := db.Query(cond, val...)
//		if err != nil {
//			log.Println("database query failed--2")
//			return
//		}
//		if rows1.Next() {
//			err = rows1.Scan(&orderInfo.Reply, &delivery)
//			if err != nil {
//				log.Println("rows1 scan failed")
//				return
//			}
//			t, err = time.Parse(layout, delivery)
//			orderInfo.DeliveryTime = t.Format("Jan 02,2006 15:04:05")
//		} else {
//			orderInfo.Reply = "no reply"
//			if orderInfo.IsCompleted == "expired" {
//				orderInfo.DeliveryTime = "expired"
//			}
//		}
//		cond, val, err = builder.BuildSelect("user", map[string]interface{}{"phone": userPhone}, []string{"name", "birth", "gender"})
//		if err != nil {
//			log.Println("build select failed")
//			return
//		}
//		rows1, err = db.Query(cond, val...)
//		if err != nil {
//			log.Println("database query failed--3")
//			return
//		}
//		if rows1.Next() {
//			err = rows1.Scan(&orderInfo.Name, &birth, &orderInfo.Gender)
//			if err != nil {
//				log.Println("rows1 scan failed")
//				return
//			}
//			t, err := time.Parse("2006-01-02", birth)
//			if err != nil {
//
//				log.Println("time parse failed")
//				return
//			}
//			orderInfo.DateOfBirth = t.Format("Jan 02,2006")
//		}
//
//		cond, val, err = builder.BuildSelect("orders_info", map[string]interface{}{"id": orderInfo.OrderId}, []string{"general_situ", "specified_ques"})
//		if err != nil {
//			log.Println("build select failed")
//			return
//		}
//		rows1, err = db.Query(cond, val...)
//		if err != nil {
//			log.Println("database query failed--4")
//			return
//		}
//		if rows1.Next() {
//			err = rows1.Scan(&orderInfo.GeneralSitu, &orderInfo.SpecifiedQues)
//			if err != nil {
//				log.Println("rows1 scan failed")
//				return
//			}
//		}
//
//		orderInfoData, err := json.Marshal(orderInfo)
//		if err != nil {
//			log.Println("json marshal failed")
//			return
//		}
//		err = rdb.SetNX(Ctx, fmt.Sprintf("orderInfo:%s", orderInfo.OrderId), orderInfoData, 3*24*time.Hour).Err()
//		if err != nil {
//			log.Println("set key-value failed")
//			return
//		}
//
//	}
//}

func CacheOrderReview(rdb *redis.Client) {
	//reviews := Reviews{}
	//review := ReviewDisplay{}
	//var orderIdBefore, orderIdAfter, rvw, reviewTime string
	//cond, val, err := builder.BuildSelect("review", map[string]interface{}{}, []string{"order_id", "review", "rating", "review_time"})
	//if err != nil {
	//	log.Println("build select failed")
	//	return
	//}
	//rows, err := db.Query(cond, val...)
	//if err != nil {
	//	log.Println("database query failed")
	//	return
	//}
	//for rows.Next() {
	//	err = rows.Scan(&orderIdAfter, &rvw, &reviews.Rating, &reviewTime)
	//	if err != nil {
	//		log.Println("rows scan failed")
	//		return
	//	}
	//	if orderIdAfter == orderIdBefore {
	//
	//	}
	//}
	return
}
