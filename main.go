package main

import (
	"encoding/json"
	"flag"
	"fmt"
	filter "github.com/antlinker/go-dirtyfilter"
	"github.com/antlinker/go-dirtyfilter/store"
	"github.com/pkg/profile"
	"gitlab.babeltime.com/packagist/blogger"
	"net/http"
	"os"
	"strconv"
	"time"
)

var FilterManage *filter.DirtyManager

type Result struct {
	Error      int        `json:"error"`
	Message    string     `json:"message"`
	Data       ResultData `json:"data"`
	Conclusion int        `json:"conclusionw"`
	Logid      int        `json:"logid"`
	Timestamp  int64      `json:"timestamp"`
}

type ResultData struct {
	OriginWord string `json:"origin_word"`
	Num   int      `json:"num"`
	Words []string `json:"words"`
	Replace  string `json:"replace"`
}

var (
	BFile = blogger.NewBFile("/tmp/godfa.log", blogger.L_DEBUG)
	DEBUG = flag.Bool("debug", false, "")
	dictFile = flag.String("file", "", "")
)

func main() {
	flag.Parse()
	if *DEBUG {
		defer profile.Start().Stop()
	}
	if len(*dictFile) == 0 {
		panic("please input file flag")
	}
	_f, _ := os.Open(*dictFile)
	defer _f.Close()
	memStore, err := store.NewMemoryStore(store.MemoryConfig{
		Reader: _f,
	})
	if err != nil {
		panic(err)
	}
	FilterManage = filter.NewDirtyManager(memStore)
	http.HandleFunc("/api/v1/word", startFilter)
	http.HandleFunc("/check/text", startFilter)
	http.HandleFunc("/getwordlist", getFilterWordList)

	err = http.ListenAndServe(":8000", nil)
	if err != nil {
		fmt.Println(err)
		panic("server down")
	}
}

//  获取所有敏感词列表
func getFilterWordList(w http.ResponseWriter, r *http.Request) {
	all, _ := FilterManage.Store().ReadAll()
	result := Result{
		Error:   0,
		Message: "ok",
		Data: ResultData{
			Num:   len(all),
			Words: all,
		},
		Conclusion: 0,
		Timestamp:  time.Now().Unix(),
		Logid:      blogger.GetLogid(),
	}
	_r, _ := json.Marshal(result)
	w.Write(_r)
}

//开始进行关键词过滤
func startFilter(w http.ResponseWriter, r *http.Request) {
	logger := blogger.NewBlogger(BFile)
	logid := blogger.GetLogid()
	logger.AddBase("logid", strconv.Itoa(logid))
	_begin := time.Now()
	var word string

	// 兼容老关键词
	if word = r.FormValue("word"); word == "" {
		word = r.FormValue("content")
	}

	//返回值数据
	var result = Result{
		Error: 0,
		Data: ResultData{
			OriginWord: word,
			Num:   0,
			Words: []string{},
		},
		Conclusion: 0,
		Message:    "ok",
		Timestamp:  time.Now().Unix(),
		Logid:      blogger.GetLogid(),
	}

	if word == "" {
		result.Error = 2
		result.Message = "word is empty"
	} else {
		//过滤
		wordList, err := FilterManage.Filter().Filter(word)
		if err != nil {
			result.Error = 1
		} else if num := len(wordList); num > 0 {
			result.Data.Num = num
			result.Data.Words = wordList
			result.Conclusion = 1
			//是否需要将关键词过滤
			if r.FormValue("replace") == "true" {
				// 是否有替换字母
				delim := []rune{'*'}
				if r.FormValue("delim") != "" {
					delim = []rune(r.FormValue("delim"))
				}
				result.Data.Replace, _ = FilterManage.Filter().Replace(word, delim[0])
			}
		}
	}

	//返回json格式
	_r, _ := json.Marshal(result)
	logger.Info(string(_r))
	logger.Info("total time :" + time.Since(_begin).String())
	logger.Flush()
	w.Write(_r)
}
