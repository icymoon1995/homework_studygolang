package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// 20 * 1024 b  = 20kb
// 最大速度
const MaxSpeed int = 20 * 1024

// 利用range 正常能处理返回206
const SuccessCode int = 206

// 不能处理range请求的还是返回200
const HalfSuccessCode int = 200

func main() {
	/**
	  大概思路:
	    利用 http request 中 header.range参数
	    类似于 分块请求

	  问题:
	  1.没有利用协程。。准确的说不知道这种情况下怎么用
	      感觉以下用法会有问题
	        countDownLatch := sync.waitGroup
	        for ...spendTime {
				countDownLatch.Add(1)

				go func () {
					// 请求下载分块
				} (start, end)

				countDownLatch.Done()
			}
		    countDownLatch.Wait()
	   2. 对于无法处理range的服务器。 对方依旧会返回200, 并直接附上整体文件
	   3. 对于巨大文件(没有进行测试)， 由于单线程会阻塞很久。
	*/
	var fileName string = "go-test-file.txt"
	var filePath string = "http://localhost:8082/"
	downloadFile(fileName, filePath)
}

/**
 * 下载文件
 *  @param fileName string 文件名
 *  @param filePath string 文件路径
 */
func downloadFile(fileName string, filePath string) {
	// 拼接file的真正path
	var requestFileUrl string = joinReallyPath(fileName, filePath)

	//	requestFileUrl = "https://studygolang.com/"
	res, _ := http.Head(requestFileUrl)
	headAttributes := res.Header
	contentLength, ok := headAttributes["Content-Length"]
	// 确定能取到 文件长度 Content-Length
	if ok {
		// fmt.Println("success")
	} else {
		log.Fatal("500 error , not found content-length from request")
	}

	// 文件长度
	fileSize, _ := strconv.Atoi(contentLength[0])

	// 计算以 最大速度下载， 需要请求多少次
	spendTime := fileSize / MaxSpeed
	if (fileSize % MaxSpeed) != 0 {
		spendTime += 1
	}

	// 用于保存最后的文件
	var file string

	// 用协程 无法控制每个协程最终的速度
	for index := 0; index < spendTime; index++ {
		// 由于利用了range  类似于断点续传
		// 每次需要请求 具体的文件块大小
		startByte := MaxSpeed * index
		endByte := MaxSpeed * (index + 1)
		if endByte >= fileSize {
			endByte = fileSize
		}

		// 具体的请求处理
		client := &http.Client{}
		request, _ := http.NewRequest("GET", requestFileUrl, nil)
		// range参数
		HeadRange := "bytes=" + strconv.Itoa(startByte) + "-" + strconv.Itoa(endByte)
		request.Header.Add("Range", HeadRange)

		//	fmt.Println(request)
		// 发送请求和处理响应
		response, _ := client.Do(request)
		handleResponse(*response)
		// 读取响应
		reader, _ := ioutil.ReadAll(response.Body)
		file += string(reader)

		// 控制一下速度?
		// 简略处理 直接暴力写成了1s
		// 大概想法是 记录请求到响应的处理时间, 如果 <= 1s ， 则sleep 1 - (消耗时间) 秒， 可能涉及一些精度处理问题
		// 大于1秒 则不休眠
		time.Sleep(1 * time.Second)

		// 最后一次关闭流
		if index == spendTime-1 {
			_ = response.Body.Close()
		}
	}

	fmt.Println(file)
}

/**
 * 拼接路径及文件名
 *	   判断路径中是否包含文件名
 *     判断结尾是否包含"/"
 *  @param fileName string 文件名
 *  @param filePath string 文件路径
 *  @param string
 */
func joinReallyPath(fileName string, filePath string) string {
	// 包含文件名 直接返回
	if strings.HasSuffix(filePath, fileName) {
		return filePath
	}
	// 判断结尾的 /
	var reallyPath string
	if strings.HasSuffix(filePath, "/") {
		reallyPath = filePath + fileName
	} else {
		reallyPath = filePath + "/" + fileName
	}
	return reallyPath
}

/**
 * 处理响应
 *  @param response http.Response 返回的响应
 */
func handleResponse(response http.Response) {
	// 处理响应
	if response.StatusCode != SuccessCode {
		if response.StatusCode == HalfSuccessCode {
			// 如果为200 会直接将整个文件返回 不需要继续请求
			log.Println("request success but server not support range, please try anther way")
			// 后续处理
		} else {
			log.Fatal("error")
			// 后续处理
		}
	}
}
