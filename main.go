package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/zellyn/kooky/browser/chrome"
)

func getChromeCookie(webName string) string {
	dir, _ := os.UserConfigDir() // "/<USER>/Library/Application Support/"
	cookiesFile := dir + "/Google/Chrome/Default/Cookies"
	cookies, err := chrome.ReadCookies(cookiesFile)
	if err != nil {
		log.Fatal(err)
	}
	cookieStr := ""
	for _, cookie := range cookies {
		if cookie.Domain == webName {
			// fmt.Println(cookie)
			cookieStr = cookieStr + cookie.Name + "=" + cookie.Value + "; "
		}
	}
	// fmt.Println(cookieStr)
	return cookieStr
}

func main() {
	// 获取命令行参数
	args := os.Args[1:]
	if len(args) != 2 {
		fmt.Println("Usage: ivoice <input_file> <output_dir>")
		os.Exit(1)
	}
	inputFile := args[0]
	outputDir := args[1]
	veed_cookie := getChromeCookie(".veed.io")

	// 判断输出目录是否存在
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		fmt.Println("Output directory does not exist.")
		os.Mkdir("result", os.ModePerm)
	}

	// 打开输入文件并按行读取文本
	file, err := os.Open(inputFile)
	if err != nil {
		fmt.Println("Failed to open input file.")
		os.Exit(1)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// 获取当前行文本
		text := scanner.Text()

		// 发送HTTP GET请求，获取音频数据
		url := fmt.Sprintf("https://www.veed.io/api/v1/subtitles/synthesize/preview?text=%s&voice=zh-CN-XiaoxiaoNeural", url.QueryEscape(text))
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Printf("Failed to get audio data for text \"%s\".\n", text)
			continue
		}
		req.Header.Set("authority", "www.veed.io")
		req.Header.Set("accept", "*/*")
		req.Header.Set("accept-language", "zh-CN,zh;q=0.9,ca;q=0.8,en;q=0.7")
		req.Header.Set("cookie", veed_cookie)
		// req.Header.Set("range", "bytes=0-")
		req.Header.Set("referer", "https://www.veed.io/")
		req.Header.Set("sec-ch-ua", "\"Google Chrome\";v=\"111\", \"Not(A:Brand\";v=\"8\", \"Chromium\";v=\"111\"")
		req.Header.Set("sec-ch-ua-mobile", "?0")
		req.Header.Set("sec-ch-ua-platform", "\"macOS\"")
		req.Header.Set("sec-fetch-dest", "empty")
		req.Header.Set("sec-fetch-mode", "cors")
		req.Header.Set("sec-fetch-site", "same-origin")
		req.Header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36")

		time.Sleep(2000)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			if strings.Contains(err.Error(), "Rate limit exceeded") {
				fmt.Printf("Rate limit exceeded for text \"%s\".\n", text)
			} else {
				fmt.Printf("Failed to get audio data for text \"%s\".\n", text)
			}
			os.Exit(1)
		}
		if resp.StatusCode != 200 {
			fmt.Printf("Failed to get audio data for text \"%s\". Http code is \"%d\".\n", text, resp.StatusCode)
			os.Exit(1)
		}
		defer resp.Body.Close()

		// 读取音频数据并保存到文件
		outputFile := fmt.Sprintf("%s/%s.mp3", outputDir, strings.TrimSpace(text))
		file, err := os.Create(outputFile)
		if err != nil {
			fmt.Printf("Failed to create audio file \"%s\" for text \"%s\".\n", outputFile, text)
			continue
		}
		defer file.Close()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			fmt.Printf("Failed to write audio data for text \"%s\" to file \"%s\".\n", text, outputFile)
			continue
		}

		// 输出提示信息
		fmt.Printf("Successfully generated audio file \"%s\".\n", outputFile)
	}
}
