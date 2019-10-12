package crawler

/**
 * user: ZY
 * Date: 2019/9/9 9:10
 */

import (
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var RedisConn redis.Conn

//存储路由信息
func RouteInit() {
	r := gin.Default()
	r.POST("/storeClass", StoreClass)
	r.POST("/getClass", GetClassCourse)
	r.POST("/getAll",GetAllCourse)
	_ = r.Run(":8080")
	defer RedisConn.Close()
}

func RedisInit() {
	var err error
	RedisConn, err = redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		log.Println("redis connect is failed", err)
		return
	}
}

//存储班级学生信息
//参数
//classId
//stuIds
//其中stuIds的格式是每个分割用',',不能用空格,学号必须为10位
//其中classId必须为8位//被我删除
func StoreClass(context *gin.Context) {
	classId := context.PostForm("classId")
	stuIds := context.PostForm("stuIds")
	isClass, err := redis.Int(RedisConn.Do("SISMEMBER", "schoolClass", classId))
	if err != nil {
		log.Println("get the classId failed", err)
		isClass = 0
	}
	if isClass != 1 {
		context.JSON(http.StatusBadRequest, gin.H{"message": "please input the right classId"})
		return
	}
	stuIds = strings.ReplaceAll(stuIds, " ", "")
	reStuId := strings.Split(stuIds, ",")
	for _, v := range reStuId {
		if !IsExistedStu(v) {
			context.JSON(http.StatusBadRequest, gin.H{"message": v + " is not existed"})
			return
		}
	}
	if StoreClassStuId(reStuId, classId, RedisConn) {
		context.JSON(http.StatusOK, gin.H{"message": "store the information successfully"})
		return
	} else {
		context.JSON(http.StatusBadGateway, gin.H{"error": "store the information failed"})
		return
	}
}

//得到该周该班级都没课的信息
//参数
//classId
//week
func GetClassCourse(context *gin.Context) {
	classId := context.PostForm("classId")
	week := context.PostForm("week")
	isClass, err := redis.Int(RedisConn.Do("SISMEMBER", "schoolClass", classId))
	if err != nil {
		log.Println("get the classId failed", err)
		isClass = 0
	}
	if len(classId) != 8 && isClass != 1 {
		context.JSON(http.StatusBadRequest, gin.H{"message": "please input the right classId"})
		return
	}
	var hashWeek HashWeek
	wk, _ := strconv.Atoi(week)
	hashWeek = GetClassHash(wk, classId, RedisConn)
	for k, v := range hashWeek {
		for kk, vv := range v {
			if vv == 0 {
				hashWeek[k][kk] = 1
			} else {
				hashWeek[k][kk] = 0
			}
		}
	}
	var hashCourse HashCourse
	hashCourse.NewHashCourse(hashWeek[0],hashWeek[1],hashWeek[2],hashWeek[3],hashWeek[4])
	context.JSON(http.StatusOK,hashCourse)
	return
}

//stuIds
//week
//根据学号直接获得该week的课表信息
func GetAllCourse(context *gin.Context) {
	stuIds := context.PostForm("stuIds")
	week := context.PostForm("week")
	stuIds = strings.ReplaceAll(stuIds, " ", "")
	reStuId := strings.Split(stuIds, ",")
	for _, v := range reStuId {
		if !IsExistedStu(v) {
			context.JSON(http.StatusBadRequest, gin.H{"message": v + " is not existed"})
			return
		}
	}
	var hashWeek HashWeek
	wk, _ := strconv.Atoi(week)
	hashWeek = GetAllHash(wk, reStuId)
	for k, v := range hashWeek {
		for kk, vv := range v {
			if vv == 0 {
				hashWeek[k][kk] = 1
			} else {
				hashWeek[k][kk] = 0
			}
		}
	}
	var hashCourse HashCourse
	hashCourse.NewHashCourse(hashWeek[0],hashWeek[1],hashWeek[2],hashWeek[3],hashWeek[4])
	context.JSON(http.StatusOK,hashCourse)
	return
}

type HashCourse struct {
	Monday    [6]int `json:"星期一"`
	Tuesday   [6]int `json:"星期二"`
	Wednesday [6]int `json:"星期三"`
	Thursday  [6]int `json:"星期四"`
	Friday    [6]int `json:"星期五"`
}

func (hash *HashCourse) NewHashCourse(mon, tues, wed, thur, fri [6]int) {
	hash.Monday=mon
	hash.Wednesday=wed
	hash.Thursday=thur
	hash.Friday=fri
	hash.Tuesday=tues

	return
}
