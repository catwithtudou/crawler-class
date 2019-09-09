package crawler

/**
 * user: ZY
 * Date: 2019/9/9 9:09
 */


import (
	"github.com/garyburd/redigo/redis"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type HashWeek [7][6]int


//<div id="kbStuTabs-list"[\s\S\w\W]*<div>*?

//<td>星期(.*)</td><td>.*?

//<td>星期(\d{1})[\s\S]?第(\d{1})[-.](\d{1})[\S\s]?(.*)周[\s\S]?</td><td>.*?

//(\d{1})[\s]?第(.*)节[\s]?(.*)周

//$1 星期数
//$2 节数
//$3 周数


//周数需考虑单周双周,和用','号隔开的周数

//存储该班级学号
func StoreClassStuId(classStuId []string,classId string,client redis.Conn)bool{
	var err error
	for _,v:=range classStuId{
		_,err=client.Do("SADD",classId,v)
		if err != nil {
			log.Println("redis sadd failed:",err)
			return false
		}
	}
	return true
}


//输入周数和班级号获得该班级的哈希表(若有一人有课就为1,若都没课为0)
func GetClassHash(week int,classId string,client redis.Conn)(hash HashWeek){

	classStuIds,err:=redis.Strings(client.Do("SMEMBERS",classId))
	if err != nil {
		log.Println("get the classStuIds failed:",err)
		return
	}
	var classHash []HashWeek
	classHash=make([]HashWeek,len(classStuIds))
	for k,v:=range classStuIds{
		classHash[k]=GetStuWeek(v,week)
	}
	for _,v:=range classHash{
		for kk,vv:=range v{
			for kkk,vvv:=range vv{
				hash[kk][kkk]+=vvv
				if hash[kk][kkk]>1{
					hash[kk][kkk]=1
				}
			}
		}
	}
	return
}


//通过输入学号和周数得到该学生该周的哈希课表
func GetStuWeek(stuId string,week int)(hash HashWeek){
	html:=GetHtml(stuId)
	list:=GetListHtml(html)
	td:=GetTdList(list)
	hash=GetWeek(week,td)
	return
}


func GetHtml(stuId string)string{
	resp,err:=http.Get("http://jwzx.cqu.pt/kebiao/kb_stu.php?xh="+stuId+"#kbStuTabs-list")
	if err != nil {
		panic(err.Error())
	}
	defer func(){
		resp.Body.Close()
	}()
	bytes,err:=ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err.Error())
	}
	return string(bytes)
}

func GetListHtml(html string)string{
	regList:=`<div id="kbStuTabs-list"[\s\S\w\W]*<div>*?`
	compile:=regexp.MustCompile(regList)
	subMatch:=compile.FindStringSubmatch(html)
	return subMatch[0]
}


func GetTdList(list string)[]string{
	regTd:=`<td>星期(.*)</td><td>.*?`
	compile:=regexp.MustCompile(regTd)
	subMatch:=compile.FindAllStringSubmatch(list,-1)
	var tdValue []string
	for _,v:=range subMatch{
		tdValue=append(tdValue,v[1])
	}
	return tdValue
}


func GetTdValue(td string)[]string{
	regTdValue:=`(\d{1})[\s]?第(.*)节[\s]?(.*)周`
	compile:=regexp.MustCompile(regTdValue)
	subMatch:=compile.FindStringSubmatch(td)
	return subMatch
}

//通过查询的周数和得到的该学生的list来获得该学生的HashWeek即对应该周的课程表
func GetWeek(week int,list []string)(hash HashWeek){
	var tdValueList [][]string
	length:=len(list)
	tdValueList=make([][]string,length)
	var i=0
	for _,v:=range list{
		value:=GetTdValue(v)
		tdValueList[i]=value
		i++
	}
	for _,v:=range tdValueList{
		if len(v)<1{
			continue
		}
		reWeek:=v[3]
		var weekNum []int
		var weekSplit []string
		if strings.Contains(reWeek,","){
			weekSplit=strings.Split(reWeek,",")
		}else{
			weekSplit=append(weekSplit,reWeek)
		}
		for _,vv:=range weekSplit{
			if strings.Contains(vv,"-")&&strings.Contains(vv,"单"){
				index:=strings.Index(vv,"-")
				anoIndex:=strings.Index(vv,"周")
				left, _ :=strconv.Atoi(string(vv[index-1]))
				right,_:=strconv.Atoi(vv[index+1:anoIndex])
				if left%2==0{
					left++
				}
				for i:=left;i<=right;i+=2{
					weekNum=append(weekNum,(i))
				}
			}else if strings.Contains(vv,"-")&&strings.Contains(vv,"双"){
				index:=strings.Index(vv,"-")
				anoIndex:=strings.Index(vv,"周")
				left, _ :=strconv.Atoi(string(vv[index-1]))
				right,_:=strconv.Atoi(vv[index+1:anoIndex])
				if left%2!=0{
					left++
				}
				for i:=left;i<=right;i+=2{
					weekNum=append(weekNum,(i))
				}
			}else if strings.Contains(vv,"-"){
				index:=strings.Index(vv,"-")
				var right,left int
				if strings.Contains(vv,"周"){
					anoIndex:=strings.Index(vv,"周")
					right,_=strconv.Atoi(vv[index+1:anoIndex])
				}else{
					right,_=strconv.Atoi(vv[index+1:])
				}
				left, _ =strconv.Atoi(string(vv[index-1]))
				for i:=left;i<=right;i++{
					weekNum=append(weekNum,(i))
				}
			}else{
				var anoIndex int
				if strings.Contains(vv,"周"){
					anoIndex=strings.Index(vv,"周")
				}else{
					anoIndex=len(vv)
				}
				value, _ :=strconv.Atoi(vv[0:anoIndex])
				weekNum=append(weekNum,value)
			}
		}
		for _,vv:=range weekNum{
			if vv==week{
				day, _ :=strconv.Atoi(v[1])
				class,_:=strconv.Atoi(string(v[2][0]))
				hash[day-1][class/2]=1
			}
		}
	}
	return
}


//<div id="kbTabs-bj"[\s\S\w\W]*<div id="kbTabs-kc">

//bj=(\d){8}'


//获取学校班级html网页
func getClassHtml()string{
	resp,err:=http.Get("http://jwzx.cqu.pt/kebiao/index.php")
	if err != nil {
		panic(err.Error())
	}
	defer func(){
		resp.Body.Close()
	}()
	bytes,err:=ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err.Error())
	}
	return string(bytes)
}


//保存学校所有班级号
func GetAllClass(conn redis.Conn){
	html:=getClassHtml()
	part1:=`<div id="kbTabs-bj"[\s\S\w\W]*<div id="kbTabs-kc">`
	part3:=`bj=(\d{8})'`
	compile1:=regexp.MustCompile(part1)
	subMatch1:=compile1.FindStringSubmatch(html)
	compile3:=regexp.MustCompile(part3)
	subMatch3:=compile3.FindAllStringSubmatch(subMatch1[0],-1)
	for _,v:=range subMatch3{
		_,err:=conn.Do("SADD","schoolClass",v[1])
		if err != nil {
			log.Println("redis sadd failed:",err)
		}
	}
}


//<div id="kbStuTabs-list"[\s\S\w\W]*<div>*?

//此学号是否存在
func IsExistedStu(stuId string)bool{
	html:=GetHtml(stuId)
	part1:=`<div id="kbStuTabs-list"[\s\S\w\W]*<div>*?`
	compile1:=regexp.MustCompile(part1)
	subMatch1:=compile1.FindStringSubmatch(html)
	part2:=`<tbody>(.*?)</tbody>`
	compile2:=regexp.MustCompile(part2)
	subMatch2:=compile2.FindStringSubmatch(subMatch1[0])
	if len(subMatch2)==0{
		return true
	}
	return false

}