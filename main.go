package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"github.com/murlokswarm/log"
	"fmt"
)

type UrlObject struct {
	status int
	url    string
}

const FILE_TYPE = "jpg|ico|jpeg|gif|bmp|svg|ipa|apk|dmg|do|action|png|css|js|json|mp3|zip|exe|pdf|rm|avi|xls|mdf|doc|MID|ppt|wps|rmvb|wma|wav|wfs|torrent"

var OnlyThisDomain = true
var maxThreadNumber = 20
var useThreadNumber = 0
var urlList [] UrlObject
var l sync.Mutex

func getHtml(url string) (string, string) {
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil {
		log.Error(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	result := string(body[:])

	url = strings.Split(url, "?")[0]

	return url, result
}

func getUrl(baseUrl string, html string) []string {
	r, _ := regexp.Compile("src=\"(.*?)\" | href=\"(.*?)\"")
	array := r.FindAllStringSubmatch(html, -1)

	result := make([]string, 0, len(array))

	// 去掉baseUrl 的参数 ?后的内容
	baseUrl = strings.Split(baseUrl, "?")[0]
	for i := 0; i < len(array); i++ {
		var url string
		fmt.Println(array)
		if array[i][1] != "" {
			url = array[i][1]
		} else if array[i][2] != "" {
			url = array[i][2]
		}

		if url == "#" || url == "/" {
			// 过滤#
			continue
		}
		if len(url) > 10 && strings.EqualFold(url[0:10], "javascript") {
			// 过滤javascript脚本
			continue
		}
		if len(url) > 7 && strings.EqualFold(url[0:7], "tencent") {
			// 过滤javascript脚本
			continue
		}

		res := strings.Split(FILE_TYPE, "|")
		br := false
		for j := 0; j < len(res); j++ {
			// 过滤静态资源
			str := res[j]
			fmt.Println(url)
			fmt.Println(str)
			urlTypes := strings.Split(url, ".")
			var urlType string
			if len(urlTypes) > 0 {
				urlType = urlTypes[len(urlTypes)-1]

			}
			//fmt.Println(str + "|" + urlType)
			if strings.EqualFold(str, urlType) {
				br = true
				continue
			}
		}
		if br {
			continue
		}
		if strings.Index(url, "://") == -1 {
			if url[0] == '/' {
				// 这是url从根开始的真是路径 例如:{/admin/index.html}
				urls := strings.Split(baseUrl, "/")
				url = urls[0] + urls[1] + urls[2] + url
			} else {
				url = baseUrl + url
			}
		}

		//fmt.Println(url)
		if len(url) > len(baseUrl){
			if OnlyThisDomain && url[0:len(baseUrl)] == baseUrl {
				result = append(result, url)
			}
			if !OnlyThisDomain {
				result = append(result, url)
			}
		}

	}
	return result
}

func saveUrl(url string) int {
	if len(urlList) == 0 {
		urlList = append(urlList, UrlObject{0, url})
	}
	for i := 0; i < len(urlList); i++ {
		if url != urlList[i].url {
			urlList = append(urlList, UrlObject{0, url})
			return 1
		}
	}
	return 0
}

func saveUrls(urls []string) int {
	i := 0
	for i := 0; i < len(urls); i++ {
		i += saveUrl(urls[i])
	}
	return i
}

func getNoGetUrl() (string, UrlObject) {
	for i := 0; i < len(urlList); i++ {
		if 0 == urlList[i].status {
			l.Lock()
			urlList[i].status = 1
			l.Unlock()
			return urlList[i].url, urlList[i]
		}
	}
	// 如果执行到此还未return说明所有地址已经爬完，等待所有urlList的status=1结束
	return "", UrlObject{}
}

func runThreadGet() {
	url, urlObj := getNoGetUrl()
	if url != "" {
		url, result := getHtml(url)
		urls := getUrl(url, result)
		if saveUrls(urls) == 0 {
			urlObj.status = 2
			fmt.Println(urlList)
			useThreadNumber--
			os.Exit(1)
		}
		urlObj.status = 2
	} else {
		// 当所有url都被获取完毕开始检测是否全部读取完毕
		over := true
		for i := 0; i < len(urlList); i++ {
			if 1 == urlList[i].status {
				over = false
				break
			}
		}
		if over == false {
			time.Sleep(500)
			useThreadNumber--
			return
		} else {
			fmt.Println(urlList)
			useThreadNumber--
			os.Exit(1)
		}

	}
	useThreadNumber--
	return
}

func main() {

	url, result := getHtml("http://www.hntxrj.com/")
	urls := getUrl(url, result)
	saveUrls(urls)

	for true {
		if useThreadNumber < maxThreadNumber {
			l.Lock()
			maxThreadNumber++
			l.Unlock()
			go runThreadGet()
		}
	}

	//url := "http://www.hntxrj.com/"
	//fmt.Println(strings.Split(url, "?")[0])

	//url := "http://www.hntxrj.com/"
	//fmt.Println(len(strings.Split(url, "/")))

}
