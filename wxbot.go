package main

import (
	"encoding/json"
	"fmt"
	"github.com/eatmoreapple/openwechat"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Payload struct {
	Message string `json:"message"`
	SendTo  string `json:"to"`
}

type Response struct {
	Phone string `json:"phone"`
}

type SendMessagePayload struct {
	Message string `json:"message"`
	SendTo  string `json:"to"`
	Number  string `json:"number"`
}

// vars
var bot = openwechat.DefaultBot(openwechat.Desktop)
var friends openwechat.Friends
var phone = make(map[string]string, 5)
var client = http.Client{}

func main() {
	http.HandleFunc("/api/sendMessage", sendMessage)
	http.HandleFunc("/api/sendMessage1", sendMessage1)
	getDevices()

	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl
	err := bot.Login()
	if err != nil {
		log.Fatalf("登录失败: %v", err)
	}

	user, err := bot.GetCurrentUser()
	if err != nil {
		fmt.Println(err)
		return
	}
	friends, err = user.Friends()
	if err != nil {
		fmt.Println(err)
		return
	}

	// send sms
	bot.MessageHandler = func(msg *openwechat.Message) {
		if msg.IsText() {
			if strings.Contains(msg.Content, "发短信") || strings.Contains(msg.Content, "发信") || strings.Contains(msg.Content, "fdx") {
				fmt.Println(msg.Content)
				cmd := strings.Split(msg.Content, "\n")
				phoneNumber := cmd[1]
				sendTo := cmd[2]
				message := cmd[3]
				for key, value := range phone {
					if strings.Contains(key, phoneNumber) {
						url := fmt.Sprintf("http://%s:801/api/sendMessage?number=%s&message=%s", value, sendTo, message)
						HttpGet(url)
					}
				}
			}
		}
	}

	http.ListenAndServe(":802", nil)
	select {}

}

func sendMessage1(writer http.ResponseWriter, request *http.Request) {
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		fmt.Println(err)
	}
	defer request.Body.Close()
	var payload SendMessagePayload
	err = json.Unmarshal([]byte(body), &payload)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(payload.Message)
	fmt.Println(payload.SendTo)
	fmt.Println(payload.Number)
	for key, value := range phone {
		if strings.Contains(key, payload.Number) {
			fmt.Println(payload.Number, value)
			url := fmt.Sprintf("http://%s:801/api/sendMessage?number=%s&message=%s", value, payload.SendTo, payload.Message)
			HttpGet(url)

		}
	}
}

func sendMessage(writer http.ResponseWriter, request *http.Request) {
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		fmt.Println(err)
	}
	defer request.Body.Close()
	var payload Payload
	err = json.Unmarshal([]byte(body), &payload)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(payload.Message)
	fmt.Println(payload.SendTo)

	target := friends.GetByNickName(payload.SendTo)
	target.SendText(payload.Message)
}

func getDevices() {
	localIP, err := getLocalIP()
	if err != nil {
		fmt.Println("Error getting local IP address:", err)
		os.Exit(1)
	}

	baseIPs := strings.Split(localIP, ".")
	baseIP := baseIPs[0] + "." + baseIPs[1] + "." + baseIPs[2]
	port := 801
	ips := generateIPs(baseIP)

	var wg sync.WaitGroup
	results := make(chan string, len(ips))

	for _, ip := range ips {
		wg.Add(1)
		go scanPort(ip, port, &wg, results)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		fmt.Println(result)
		url := fmt.Sprintf("http://%s:801/api/getNumber", result)
		ret := HttpGet(url)
		var responseData Response
		err = json.Unmarshal([]byte(ret), &responseData)
		if err != nil {
			log.Fatalf("Failed to unmarshal JSON: %v", err)
		}
		phone[responseData.Phone] = result
	}
	fmt.Println(phone)
}

// getLocalIP  get local ip
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("unable to find local IP address")
}

// generateIPs all ips
func generateIPs(baseIP string) []string {
	var ips []string
	for i := 1; i < 255; i++ {
		ip := fmt.Sprintf("%s.%d", baseIP, i)
		ips = append(ips, ip)
	}
	return ips
}

// scanPort test 801
func scanPort(ip string, port int, wg *sync.WaitGroup, results chan<- string) {
	defer wg.Done()
	address := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", address, time.Second*1)
	if err != nil {
		return
	}
	conn.Close()
	results <- fmt.Sprintf("%s", ip)
}

// HttpGet get
func HttpGet(url string) string {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	return httpClient(req)
}

//func HttpPost(url string, payload string) string {
//	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
//	if err != nil {
//		log.Fatal(err)
//	}
//	req.Header.Set("Content-Type", "application/json")
//
//	return httpClient(req)
//}

func httpClient(req *http.Request) string {

	resp, _ := client.Do(req)
	defer resp.Body.Close()

	bodyText, _ := io.ReadAll(resp.Body)
	return string(bodyText)
}
