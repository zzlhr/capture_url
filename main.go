package main

import (
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"fmt"
)

type UrlObject struct {
	status int
	url    string
}

const FILE_TYPE = "jpg|ico|jpeg|gif|bmp|svg|ipa|apk|dmg|do|action|png|css|js|json|mp3|zip|exe|pdf|rm|avi|xls|mdf|doc|MID|ppt|wps|rmvb|wma|wav|wfs|torrent"

var over = false
var OnlyThisDomain = true
var maxThreadNumber = 20
var useThreadNumber = 0
var urlList [] UrlObject
var l sync.Mutex

func getHtml(url string) (string, string, error) {
	resp, err := http.Get(url)

	if err != nil {
		// log.Error(err)
		//fmt.Println(err)
		return url, "", err
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err == nil {
		result := string(body[:])

		url = strings.Split(url, "?")[0]

		return url, result, nil
	}else {
		// fmt.Println(err)
		return url, "", err
	}

}

/**
 *
 * @param baseUrl 读取页面数据所属的url
 * @param html    页面数据
 * @return        string数组
 */
func getUrl(baseUrl string, html string) []string {
	r, _ := regexp.Compile("src=\"(.*?)\" | href=\"(.*?)\"")
	array := r.FindAllStringSubmatch(html, -1)

	result := make([]string, 0, len(array))

	// 去掉baseUrl 的参数 ?后的内容
	baseUrl = strings.Split(baseUrl, "?")[0]
	baseUrls := strings.Split(baseUrl, "/")
	baseUrl = ""
	for i := 0; i < len(baseUrls)-1; i++ {
		// 去除路径的最后的xxx.html等等。
		baseUrl += baseUrls[i]+"/"
	}
	for i := 0; i < len(array); i++ {
		var url string
		// fmt.Println(array)
		if array[i][1] != "" {
			url = array[i][1]
		} else if array[i][2] != "" {
			url = array[i][2]
		}

		if url[0] == '#' || url == "/" {
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
		// 如果下面循环发现文件是静态文件该变量赋值为true，跳出循环
		br := false
		for j := 0; j < len(res); j++ {
			// 过滤静态资源
			str := res[j]
			//fmt.Println(url)
			//fmt.Println(str)
			urlTypes := strings.Split(url, ".")
			var urlType string
			if len(urlTypes) > 0 {
				urlType = urlTypes[len(urlTypes)-1]

			}
			//fmt.Println(str + "|" + urlType)
			if strings.EqualFold(str, urlType) {
				// 该文件为静态文件
				br = true
				continue
			}
		}
		if br {
			// 静态文件跳出循环
			continue
		}

		if strings.Index(url, "://") == -1 {
			if len(url) > 0{
				if url[0] == '/' {
					// 再次如果是/开头url是相对于域名根的urls[0]为https: urls[1]空字符串  urls[2]为域名
					// 这是url从根开始的真是路径 例如:{/admin/index.html},再此转换相对路径绝对路径
					urls := strings.Split(baseUrl, "/")
					url = urls[0] +"/"+ urls[1] +"/"+ urls[2] + url
				} else {
					// 相对当前路径
					if url[0:3] == "../" {
						url = formatGoBackUri(baseUrl, url)
					}else {
						url = baseUrl + url
					}
				}
				if url[0:2] == "./" {
					url = url[2:len(url)]
				}
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




func formatGoBackUri(baseUrl string, addUrl string) string {

	i := 0
	for addUrl[0:3] == "../" {
		i++
		addUrl = addUrl[3 : len(addUrl)]
	}

	u := ""
	baseUrls := strings.Split(baseUrl, "/")
	urls := baseUrls[0:len(baseUrls) - i]
	for j := 0; j < len(urls); j++{
		u += urls[j] + "/"
		if j == len(urls) {
			u += addUrl
		}
	}
	//fmt.Println(u)

	return u
}




func saveUrl(url string) int {
	ex := true

	for i := 0; i < len(urlList); i++ {
		if url == urlList[i].url {
			ex = false
			break
		}
	}

	if ex {
		urlList = append(urlList, UrlObject{0, url})
		return 1
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

func getNoGetUrl() (string, *UrlObject) {
	for i := 0; i < len(urlList); i++ {
		if 0 == urlList[i].status {
			l.Lock()
			urlList[i].status = 1
			l.Unlock()
			return urlList[i].url, &urlList[i]
		}
	}
	// 如果执行到此还未return说明所有地址已经爬完，等待所有urlList的status=1结束
	return "", &UrlObject{}
}

func runThreadGet() {
	url, urlObj := getNoGetUrl()
	urlObjIp := &urlObj.status
	*urlObjIp = 2
	if url != "" {
		fmt.Println("正在抓取:"+url)
		url, result, err := getHtml(url)

		if err == nil {
			urls := getUrl(url, result)
			if saveUrls(urls) == 0 {
				// urlObj.status = 2
				// fmt.Println(urlList)
				useThreadNumber--
				// 此处true没卵用，就是预声明一下，这里没有拿到数据。下方会进行任务完成情况检查，全部完成才会关闭程序
				over = true
				// 在结束程序前检查urlList的所有url是否状态都为完成
				for i := 0; i < len(urlList); i++ {
					if 2 != urlList[i].status {
						// 如果一个不为2就说明程序中有未完成的任务，等待程序继续.
						over = false
						break
					}
				}

			}


		}else {
			fmt.Println(err)
		}
	} else {
		// 当所有url都被获取完毕开始检测是否全部读取完毕
		// 当程序发现url获取失败时，对urllist中的数据进行检查看是否存在状态为正在获取的url如果没有就退出
		for i := 0; i < len(urlList); i++ {
			if 2 != urlList[i].status {
				over = false
				break
			}
		}
		if over == false {
			//fmt.Println("sleep")
			//fmt.Println(urlList)
			time.Sleep(100 * time.Millisecond)
			useThreadNumber--
			return
		} else {
				// fmt.Println(urlList)
			useThreadNumber--
			over = true
			// fmt.Println("程序结束184")
		}

	}
	useThreadNumber--
	return
}

func main() {

	url, result, err := getHtml("http://www.soft2005.com/Web/index.aspx")
	 fmt.Println(err)
	if err == nil {
		urls := getUrl(url, result)
		saveUrls(urls)
		for !over {
			if useThreadNumber < maxThreadNumber {
				l.Lock()
				maxThreadNumber++
				l.Unlock()
				go runThreadGet()
			}
			time.Sleep(10 * time.Millisecond)
			// fmt.Println(over)
		}
		fmt.Println(urlList)
	}


	//url := "http://www.hntxrj.com/"
	//fmt.Println(strings.Split(url, "?")[0])

	//url := "http://www.hntxrj.com/"
	//fmt.Println(len(strings.Split(url, "/")))

}
