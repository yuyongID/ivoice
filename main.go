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

const (
	veedCookieName   = "veed_cookie"
	veedCookieDomain = ".veed.io"
	veedCookiePath   = "/"

	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"
)

var (
	client = &http.Client{
		Timeout: time.Second * 30,
	}
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
			cookieStr = cookieStr + cookie.Name + "=" + cookie.Value + "; "
		}
	}
	return cookieStr
}

func getVeedCookie() string {
	return getChromeCookie(veedCookieDomain)
}

func downloadAudio(text, outputDir string) error {
	url := fmt.Sprintf("https://www.veed.io/api/v1/subtitles/synthesize/preview?text=%s&voice=zh-CN-XiaoxiaoNeural", url.QueryEscape(text))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("Failed to get audio data for text \"%s\".\n", text)
	}
	req.Header.Set("authority", "www.veed.io")
	req.Header.Set("accept", "/")
	req.Header.Set("accept-language", "zh-CN,zh;q=0.9,ca;q=0.8,en;q=0.7")
	req.Header.Set("cookie", getVeedCookie())
	req.Header.Set("referer", "https://www.veed.io/")
	req.Header.Set("sec-ch-ua", "\"Google Chrome\";v=\"111\", \"Not(A:Brand\";v=\"8\", \"Chromium\";v=\"111\"")
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", "\"macOS\"")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("user-agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "Rate limit exceeded") {
			return fmt.Errorf("rate limit exceeded for text \"%s\"", text)
		} else {
			return fmt.Errorf("failed to get audio data for text \"%s\": %v", text, err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get audio data for text \"%s\". Http code is \"%d\"", text, resp.StatusCode)
	}

	outputFile := fmt.Sprintf("%s/%s.mp3", outputDir, strings.TrimSpace(text))
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create audio file \"%s\" for text \"%s\": %v", outputFile, text, err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write audio data for text \"%s\" to file \"%s\": %v", text, outputFile, err)
	}

	fmt.Printf("Successfully generated audio file \"%s\".\n", outputFile)
	return nil
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

	// 判断输出目录是否存在
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		fmt.Println("Output directory does not exist.")
		os.Mkdir(outputDir, os.ModePerm)
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

		err := downloadAudio(text, outputDir)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
	}
}
