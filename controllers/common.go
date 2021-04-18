package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"tesla/config"
	"tesla/globalvar"
	svc "tesla/service"
	"tesla/utils"
	"time"
)

type TrafficParam struct {
	Username   string `json:"username"`
	ServerAddr string `json:"server_addr"`
	ClientAddr string `json:"client_addr"`
	TargetAddr string `json:"target_addr"`
	Bytes      string `json:"bytes"`
}

type KickParam struct {
	User string `form:"user" json:"user"`
	Ip string `form:"ip" json:"ip"`
}

func KickController(c *gin.Context){
	var info KickParam
	err := c.Bind(&info)
	if err != nil{
		fmt.Println(err.Error())
	}
	if info.User != ""{
		users := strings.Split(info.User, ",")
		return_users_str := ""
		for i := 0; i < len(users); i++ {
			infos := strings.Split(users[i], "-")
			user_username := infos[0]
			//country := infos[1]
			level := infos[2]
			//session := infos[3]
			//itype := infos[4]
			//rate := infos[5]
			if level == "basic" {
				key := "userBaseAuthOf" + user_username
				value, err := utils.GetRedisValueByPrefix(key)
				if err == redis.Nil {
					utils.Log.WithField("key", key).Error("redis cache value is null")
					continue
				}
				//redis get value success
				res := strings.Split(value, ":")
				//用多了
				total, _ := strconv.ParseFloat(res[1], 8)
				used, _ := strconv.ParseFloat(res[2], 8)
				if used > total {
					return_users_str += users[i] + ","
				}
			} else if level == "super" {
				key := "userSuperAuthOf" + user_username
				value, err := utils.GetRedisValueByPrefix(key)
				if err == redis.Nil {
					utils.Log.WithField("key", key).Error("redis cache value is null")
					continue
				}
				//redis get value success
				res := strings.Split(value, ":")
				//用多了
				total, _ := strconv.ParseFloat(res[1], 8)
				used, _ := strconv.ParseFloat(res[2], 8)
				if used > total {
					return_users_str += users[i] + ","
				}
			}else if level == "light" {
				key := "userLightAuthOf" + user_username
				value, err := utils.GetRedisValueByPrefix(key)
				if err == redis.Nil {
					utils.Log.WithField("key", key).Error("redis cache value is null")
					continue
				}
				//redis get value success
				res := strings.Split(value, ":")
				//用多了
				total, _ := strconv.ParseFloat(res[1], 8)
				used, _ := strconv.ParseFloat(res[2], 8)
				if used > total {
					return_users_str += users[i] + ","
				}
			}
		}
		sz := len(return_users_str)
		if sz > 0{
			c.JSON(http.StatusOK, gin.H{
				"user": return_users_str[:sz-1],
				"ip":"",
			})
			return
		}else{
			c.JSON(http.StatusOK, gin.H{
				"user": "",
				"ip":"",
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"user": "",
		"ip":"",
	})
}

func AuthController(c *gin.Context) {
	user := c.Query("user")
	password := c.Query("pass")
	//client_addr := c.Query("client_addr")
	//service := c.Query("service")
	//sps := c.Query("sps")
	target := c.Query("target")
	//fmt.Println(user, password, client_addr, service, sps, target)
	flag := utils.GetSneakerMap(target)
	infos := strings.Split(user, "-")
	user_username := infos[0]
	//user_password := password
	country := infos[1]
	level := infos[2]
	session := infos[3]
	itype := infos[4]
	rate := infos[5]

	//fmt.Println(user_username, user_password, country, level, session, itype, rate)

	key := ""
	if level == "basic" {
		key = "userBaseAuthOf" + user_username
	} else if level == "super" {
		key = "userSuperAuthOf" + user_username
	} else if level == "light" {
		key = "userLightAuthOf" + user_username
	} else {
		utils.Log.WithField("level", level).Error("level is not basic or super")
		c.JSON(http.StatusCreated, "level is not basic or super")
		return
	}

	session_number, err := strconv.Atoi(session)
	session_number = session_number
	if err != nil {
		utils.Log.WithField("session", session).Error("session parse to int err")
		c.JSON(http.StatusCreated, "session parse to int err")
		return
	}

	value, err := utils.GetRedisValueByPrefix(key)
	//redis value is not found
	if err == redis.Nil {
		utils.Log.WithField("key", key).Error("redis cache value is null")
		c.JSON(http.StatusCreated, "redis cache value is null, redis key is  "+key)
		return
	}

	//redis server error
	if err != nil {
		c.JSON(http.StatusInternalServerError, "redis server is not available")
		return
	}

	//redis get value success
	res := strings.Split(value, ":")
	//密码不正确
	if password != res[0] {
		utils.Log.WithField("password", res[0]).Error("password is not right")
		c.JSON(http.StatusCreated, "password is not right")
		return
	}

	//用多了
	total, _ := strconv.ParseFloat(res[1], 8)
	used, _ := strconv.ParseFloat(res[2], 8)
	if used > total {
		c.JSON(http.StatusCreated, "current traffic is oversize")
		return
	}

	//优化版本
	t := ""
	key = user_username + session
	val, err := utils.GetRedisValueByPrefix(key)
	//redis value is not found

	//redis server error
	if err != nil && err != redis.Nil {
		c.JSON(http.StatusInternalServerError, "redis server is not available")
		return
	}

	// value is nil
	if err == redis.Nil {
		if level == "basic" {
			key = "BasicAccountInfo" + user_username
			val, err := utils.GetRedisValueByPrefix(key)
			if err == redis.Nil {
				c.JSON(http.StatusCreated, "redis value is nil , key is "+key)
				return
			}
			if err != nil {
				c.JSON(http.StatusInternalServerError, "redis server is not available")
				return
			}
			accounts_value := strings.Split(val, ":")
			totalNumber, err := strconv.Atoi(accounts_value[0])
			if err != nil {
				c.JSON(http.StatusInternalServerError, "accounts_value[0] can not parse to int, accounts_value[0] is"+accounts_value[0])
				return
			}
			if totalNumber == 0 {
				c.JSON(http.StatusCreated, "totalNumber is 0")
				return
			}
			pick := session_number % totalNumber
			accounts_info := accounts_value[pick+1]
			accounts_array := strings.Split(accounts_info, "-")
			if accounts_array[0] == "geo"{
				t = svc.CreateOneGeo(country, itype, session, accounts_array[1], accounts_array[2])
			}
			if accounts_array[0] == "lumi" {
				if itype == "Rotate" || country == "usf" || !flag{
					accounts_info := accounts_value[1]
					accounts_array := strings.Split(accounts_info, "-")
					t = svc.CreateOneGeo(country, itype, session, accounts_array[1], accounts_array[2])
				} else{
						t = svc.CreateLumi(accounts_array[3], session, country, accounts_array[1], accounts_array[2])
				}
			}
			if accounts_array[0] == "oxy" {
				if itype == "Rotate" || country == "usf" || country == "mo" || country == "cn"  || country == "hk" || country == "cz"  {
					accounts_info := accounts_value[1]
					accounts_array := strings.Split(accounts_info, "-")
					t = svc.CreateOneGeo(country, itype, session, accounts_array[1], accounts_array[2])
				}else{
					rand.Seed(time.Now().UnixNano())
					number := rand.Intn(3)
					if number != 1{
						accounts_info := accounts_value[1]
						accounts_array := strings.Split(accounts_info, "-")
						t = svc.CreateOneGeo(country, itype, session, accounts_array[1], accounts_array[2])
					} else{
						t = svc.CreateOneOxy(country, itype, session, accounts_array[1], accounts_array[2])
					}
				}
			}
			if accounts_array[0] == "smart" {
				if itype == "Rotate" || country == "usf" || country == "mo" || country == "cn"  || country == "hk" || country == "cz"  {
					accounts_info := accounts_value[1]
					accounts_array := strings.Split(accounts_info, "-")
					t = svc.CreateOneGeo(country, itype, session, accounts_array[1], accounts_array[2])
				}else{
						t = svc.CreateOneSmart(country, itype, session, accounts_array[1], accounts_array[2])
				}
			}
		} else if level == "super" {
			key = "SuperAccountInfo" + user_username
			val, err := utils.GetRedisValueByPrefix(key)
			if err == redis.Nil {
				c.JSON(http.StatusCreated, "redis value is nil , key is "+key)
				return
			}
			if err != nil {
				c.JSON(http.StatusInternalServerError, "redis server is not available")
				return
			}
			accounts_value := strings.Split(val, ":")
			totalNumber, err := strconv.Atoi(accounts_value[0])
			if err != nil {
				c.JSON(http.StatusInternalServerError, "accounts_value[0] can not parse to int, accounts_value[0] is"+accounts_value[0])
				return
			}
			if totalNumber == 0 {
				c.JSON(http.StatusCreated, "totalNumber is 0")
				return
			}
			pick := session_number % totalNumber
			accounts_info := accounts_value[pick+1]
			accounts_array := strings.Split(accounts_info, "-")
			if accounts_array[0] == "geo" {
				t = svc.CreateOneGeo(country, itype, session, accounts_array[1], accounts_array[2])
			}
			if accounts_array[0] == "lumi"{
				if itype == "Rotate" || country == "usf"  || !flag{
					accounts_info := accounts_value[1]
					accounts_array := strings.Split(accounts_info, "-")
					t = svc.CreateOneGeo(country, itype, session, accounts_array[1], accounts_array[2])
				} else{
						t = svc.CreateLumi(accounts_array[3], session, country, accounts_array[1], accounts_array[2])
					}
			}
			if accounts_array[0] == "oxy" {
				if itype == "Rotate" || country == "usf" || country == "mo" || country == "cn"  || country == "hk" || country == "cz"  {
					accounts_info := accounts_value[1]
					accounts_array := strings.Split(accounts_info, "-")
					t = svc.CreateOneGeo(country, itype, session, accounts_array[1], accounts_array[2])
				}else{
					rand.Seed(time.Now().UnixNano())
					number := rand.Intn(3)
					if number != 1{
						accounts_info := accounts_value[1]
						accounts_array := strings.Split(accounts_info, "-")
						t = svc.CreateOneGeo(country, itype, session, accounts_array[1], accounts_array[2])
					} else{
						t = svc.CreateOneOxy(country, itype, session, accounts_array[1], accounts_array[2])
					}
				}
			}
			if accounts_array[0] == "smart" {
				if itype == "Rotate" || country == "usf" || country == "mo" || country == "cn"  || country == "hk" || country == "cz"  {
					accounts_info := accounts_value[1]
					accounts_array := strings.Split(accounts_info, "-")
					t = svc.CreateOneGeo(country, itype, session, accounts_array[1], accounts_array[2])
				}else{
					t = svc.CreateOneSmart(country, itype, session, accounts_array[1], accounts_array[2])
				}
			}
		}else if level == "light" {
			key = "LightAccountInfo" + user_username
			val, err := utils.GetRedisValueByPrefix(key)
			if err == redis.Nil {
				c.JSON(http.StatusCreated, "redis value is nil , key is "+key)
				return
			}
			if err != nil {
				c.JSON(http.StatusInternalServerError, "redis server is not available")
				return
			}
			accounts_value := strings.Split(val, ":")
			totalNumber, err := strconv.Atoi(accounts_value[0])
			if err != nil {
				c.JSON(http.StatusInternalServerError, "accounts_value[0] can not parse to int, accounts_value[0] is"+accounts_value[0])
				return
			}
			if totalNumber == 0 {
				c.JSON(http.StatusCreated, "totalNumber is 0")
				return
			}
			pick := session_number % totalNumber
			accounts_info := accounts_value[pick+1]
			accounts_array := strings.Split(accounts_info, "-")
			if accounts_array[0] == "geo" {
				t = svc.CreateOneGeo(country, itype, session, accounts_array[1], accounts_array[2])
			}
			if accounts_array[0] == "lumi"{
				if itype == "Rotate" || country == "usf"  || !flag{
					accounts_info := accounts_value[1]
					accounts_array := strings.Split(accounts_info, "-")
					t = svc.CreateOneGeo(country, itype, session, accounts_array[1], accounts_array[2])
				} else{
					t = svc.CreateLumi(accounts_array[3], session, country, accounts_array[1], accounts_array[2])
				}
			}
			if accounts_array[0] == "oxy" {
				if itype == "Rotate" || country == "usf" || country == "mo" || country == "cn"  || country == "hk" || country == "cz"  {
					accounts_info := accounts_value[1]
					accounts_array := strings.Split(accounts_info, "-")
					t = svc.CreateOneGeo(country, itype, session, accounts_array[1], accounts_array[2])
				}else{
					rand.Seed(time.Now().UnixNano())
					number := rand.Intn(3)
					if number != 1{
						accounts_info := accounts_value[1]
						accounts_array := strings.Split(accounts_info, "-")
						t = svc.CreateOneGeo(country, itype, session, accounts_array[1], accounts_array[2])
					} else{
						t = svc.CreateOneOxy(country, itype, session, accounts_array[1], accounts_array[2])
					}
				}
			}
			if accounts_array[0] == "smart" {
				if itype == "Rotate" || country == "usf" || country == "mo" || country == "cn"  || country == "hk" || country == "cz"  {
					accounts_info := accounts_value[1]
					accounts_array := strings.Split(accounts_info, "-")
					t = svc.CreateOneGeo(country, itype, session, accounts_array[1], accounts_array[2])
				}else{
					t = svc.CreateOneSmart(country, itype, session, accounts_array[1], accounts_array[2])
				}
			}
		}
		redis_key := user_username + session
		err = utils.SetRedisValueByPrefix(redis_key, t, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, "redis set value error key is "+redis_key+", value is "+t)
			return
		}
	} else {
		t = val
	}

	c.Header("userconns", config.AppConfig.UserConns)
	c.Header("ipconns", config.AppConfig.IPConns)
	c.Header("userrate", rate)
	c.Header("iprate", rate)
	c.Header("upstream", "http://"+t)
	c.JSON(http.StatusNoContent, "success")

}

func TrafficController(c *gin.Context) {
	//server_addr := c.Query("server_addr")
	//client_addr := c.Query("client_addr")
	//target_addr := c.Query("target_addr")
	username := c.Query("username")
	bytes := c.Query("bytes")

	//这里是拿Key
	infos := strings.Split(username, "-")
	user_username := infos[0]
	level := infos[2]
	userkey := ""
	if (level == "basic") {
		userkey = "userBaseAuthOf" + user_username
	} else {
		userkey = "userSuperAuthOf" + user_username
	}
	//计算
	byteUse, err := strconv.ParseFloat(bytes, 8)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "res[2] cannot parse to float, res[2] is "+ bytes)
		return
	}
	usage := 1.05 * byteUse / 10000000
	globalvar.UpdateUSERARRAYVal(userkey, usage)
	//上传
	if globalvar.AddCOUNT() > 1000 {
		UploadToKafka()
	}
	c.JSON(http.StatusNoContent, "success")
}

func UploadToKafka(){
	userArray := globalvar.CopyMap()
	go func() {
		message := ""
		for key, value := range userArray {
			rspon, _ := strconv.ParseFloat(fmt.Sprintf("%.4f", value), 64)
			rsponStr := strconv.FormatFloat(rspon, 'E', -1, 64) //float64
			message += key + ":"+rsponStr + ","
		}
		//push to kafka
		err := svc.PushTrafficParamToKafka(message)
		if err != nil {
			utils.Log.WithField("err", err).Error("push to kafka err")
			return
		}
	}()
}