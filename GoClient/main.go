package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"srunClient/encryptlib"
	"syscall"
	"time"

	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

/*
rad_user_info: 获取用户信息的地址
get_challenge: 获取加密 token 的地址
srun_portal: 身份认证地址
*/
var targets = map[string]string{
	"rad_user_info": "https://netauth.xjtlu.edu.cn/cgi-bin/rad_user_info",
	"get_challenge": "https://netauth.xjtlu.edu.cn/cgi-bin/get_challenge",
	"srun_portal":   "https://netauth.xjtlu.edu.cn/cgi-bin/srun_portal",
}

const (
	// banner
	Banner = `
   ________  __  __  ____                ________          __ 
  / __/_  / / / / / / __/_____ _____    / ___/ (_)__ ___  / /_
 _\ \  / /_/ /_/ / _\ \/ __/ // / _ \  / /__/ / / -_) _ \/ __/
/___/ /___/\____/ /___/_/  \_,_/_//_/  \___/_/_/\__/_//_/\__/ 
`
	// jsonp 标志
	callback string = "jQueryCallback"
	// 模拟 UA 头
	userAgent string = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	// 加密常量
	TYPE       string = "1"
	N          string = "200"
	ENC        string = "srun_bx1"
	ACID       string = "0"
	ConfigFile        = "config.yml"
	//
	ModeLogin  = "1"
	ModeLogout = "2"
	//
	httpErrorMessage = `[!] 访问 %s 的过程中出现网络问题, 请检查:
1. 网络环境是否正常
2. 配置的 URL 是否正确
`
)

type getIpResp struct {
	Ip string `json:"online_ip"`
}

type getChallengeResp struct {
	Challenge string `json:"challenge"`
}

type srunPortalRes struct {
	Res string `json:"res"`
}

type srunPortalErr struct {
	ErrMsg string `json:"err_msg"`
}

type config struct {
	Mode      string `yaml:"mode"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	OS        string `yaml:"os"`
	IP        string `yaml:"ip"`
	Interface string `yaml:"interface"`
	AcID      string `yaml:"ac_id"`
	EncVer    string `yaml:"EncVer"`
	EncVerV2  string `yaml:"enc_ver"`
}

type arguments struct {
	config
	cliConfig  config
	configPath string
}

func loadConfig(path string) (config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return config{}, nil
	}
	if err != nil {
		return config{}, err
	}
	var result config
	if err := yaml.Unmarshal(data, &result); err != nil {
		return config{}, err
	}
	return result, nil
}

func parseArguments() (arguments, error) {
	var result arguments
	flag.StringVar(&result.configPath, "config", ConfigFile, "YAML config file")
	flag.StringVar(&result.Mode, "mode", "", "1 for login, 2 for logout")
	flag.StringVar(&result.Username, "username", "", "username")
	flag.StringVar(&result.Password, "password", "", "password")
	flag.StringVar(&result.OS, "os", "", "device OS")
	flag.StringVar(&result.IP, "ip", "", "login IP")
	flag.StringVar(&result.Interface, "interface", "", "network interface (Python client only)")
	flag.StringVar(&result.AcID, "ac_id", "", "access controller ID")
	flag.StringVar(&result.EncVer, "EncVer", "", "encryption version")
	flag.StringVar(&result.EncVerV2, "enc_ver", "", "encryption version")
	flag.Parse()
	cliConfig := result.config

	fileConfig, err := loadConfig(result.configPath)
	if err != nil {
		return arguments{}, err
	}
	result.config = fileConfig
	result.cliConfig = cliConfig
	return result, nil
}

func applyArguments(args arguments) config {
	result := args.config
	if args.cliConfig.Mode != "" {
		result.Mode = args.cliConfig.Mode
	}
	if args.cliConfig.Username != "" {
		result.Username = args.cliConfig.Username
	}
	if args.cliConfig.Password != "" {
		result.Password = args.cliConfig.Password
	}
	if args.cliConfig.OS != "" {
		result.OS = args.cliConfig.OS
	}
	if args.cliConfig.IP != "" {
		result.IP = args.cliConfig.IP
	}
	if args.cliConfig.Interface != "" {
		result.Interface = args.cliConfig.Interface
	}
	if args.cliConfig.AcID != "" {
		result.AcID = args.cliConfig.AcID
	}
	if args.cliConfig.EncVer != "" {
		result.EncVer = args.cliConfig.EncVer
	}
	if args.cliConfig.EncVerV2 != "" {
		result.EncVer = args.cliConfig.EncVerV2
	}
	return result
}

func checkErrMsg(body []byte) (string, error) {
	var respJsonErr srunPortalErr
	fmt.Println("[*] 可能 srun 服务发生了异常, 正在检查报错码...")
	err := json.Unmarshal(body, &respJsonErr)
	if err != nil {
		fmt.Println("[!] 获取报错码失败")
		return "", err
	}
	fmt.Printf("[*] srun 报错码: %s", respJsonErr.ErrMsg)
	return respJsonErr.ErrMsg, nil
}

func checkCallback(body []byte, callback string) {
	fmt.Println("[*] 正在检查是否存在 callback 信息...")
	if string(body[:len(callback)]) == callback {
		fmt.Println("[!] 正常获取了 callback 信息, 请人工检查返回内容是否符合 json 格式:")
		fmt.Print(string(body[:100]))
		if len(body) < 100 {
			fmt.Println("")
		} else {
			fmt.Println("...")
		}
	} else {
		fmt.Println("[!] 没有 callback 信息, 请检查配置的 URL 是否正确")
	}
}

func getIp(callback, path string) (string, error) {
	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return "", err
	}
	query := req.URL.Query()
	query.Add("callback", callback)
	req.URL.RawQuery = query.Encode()
	req.Header.Add("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Printf(httpErrorMessage, path)
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var respJson getIpResp
	err = json.Unmarshal(body[len(callback)+1:len(body)-1], &respJson)
	if err != nil {
		fmt.Println("[!] 成功获取了返回的数据, 但是不存在 online_ip 值")
		errMsg, err := checkErrMsg(body[len(callback)+1 : len(body)-1])
		if err != nil {
			checkCallback(body, callback)
			return "", err
		}
		return "", fmt.Errorf("srun err: %s", errMsg)
	}
	fmt.Printf("[*] 成功获取登录 ip: %s\n", respJson.Ip)
	return respJson.Ip, nil
}

func getChallenge(callback, username, ip, path string) (string, error) {
	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return "", err
	}
	query := req.URL.Query()
	query.Add("callback", callback)
	query.Add("username", username)
	query.Add("ip", ip)
	req.URL.RawQuery = query.Encode()
	req.Header.Add("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Printf(httpErrorMessage, path)
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var respJson getChallengeResp
	err = json.Unmarshal(body[len(callback)+1:len(body)-1], &respJson)
	if err != nil {
		fmt.Println("[!] 成功获取了返回的数据, 但是不存在 challenge 值")
		errMsg, err := checkErrMsg(body[len(callback)+1 : len(body)-1])
		if err != nil {
			checkCallback(body, callback)
			return "", err
		}
		return "", fmt.Errorf("srun err: %s", errMsg)
	}
	fmt.Printf("[*] 成功获取加密 token: %s\n", respJson.Challenge)
	return respJson.Challenge, nil
}

func srunPortalLogin(callback, username, password, path, token, ip, os, acID, encVer string) {
	fmt.Println("[*] 正在加密用户信息...")
	hmd5_password := encryptlib.Hmd5(password, token)
	info := encryptlib.GetInfo(encryptlib.Info{
		Username: username,
		Password: password,
		Ip:       ip,
		Acid:     acID,
		EncVer:   encVer,
	}, token)
	chksum := encryptlib.Sha1(
		encryptlib.Chkstr(token, username, hmd5_password, ACID, ip, N, TYPE, info))
	fmt.Println("[*] 已完成用户信息加密, 准备进入身份认证")
	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return
	}
	current_time := fmt.Sprintf("%d000", time.Now().Unix())
	var loginQueryParams = map[string]string{
		"action":       "login",
		"callback":     callback,
		"username":     username,
		"password":     "{MD5}" + hmd5_password,
		"os":           os,
		"name":         os,
		"nas_ip":       "",
		"double_stack": "0",
		"chksum":       chksum,
		"info":         info,
		"ac_id":        ACID,
		"ip":           ip,
		"n":            N,
		"type":         TYPE,
		"captchaVal":   "",
		"_":            current_time,
	}
	query := req.URL.Query()
	for k, v := range loginQueryParams {
		query.Add(k, v)
	}
	req.URL.RawQuery = query.Encode()
	req.Header.Add("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Printf(httpErrorMessage, path)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var respJsonRes srunPortalRes
	err = json.Unmarshal(body[len(callback)+1:len(body)-1], &respJsonRes)
	if err != nil || respJsonRes.Res != "ok" {
		fmt.Println("[!] 登陆失败")
		_, err = checkErrMsg(body[len(callback)+1 : len(body)-1])
		if err != nil {
			checkCallback(body, callback)
		}
		return
	}
	fmt.Println("[*] 登录成功")
}

func srunPortalLogout(callback, username, ip, path string) {
	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return
	}
	var logoutQueryParams = map[string]string{
		"action":   "logout",
		"callback": callback,
		"username": username,
		"ip":       ip,
	}
	query := req.URL.Query()
	for k, v := range logoutQueryParams {
		query.Add(k, v)
	}
	req.URL.RawQuery = query.Encode()
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Println("[*] 登出失败")
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var respJson srunPortalRes
	err = json.Unmarshal(body[len(callback)+1:len(body)-1], &respJson)
	if err != nil || respJson.Res != "ok" {
		fmt.Println("[*] 登出失败")
		return
	}
	fmt.Println("[*] 登出成功")
}

func login(username, password, ip, os, acID, encVer string) {
	if username == "" {
		fmt.Print("[+] 请输入您的学号: ")
		fmt.Scanln(&username)
	}
	if password == "" {
		fmt.Print("[+] 请输入您的密码: ")
		passwordBytes, _ := term.ReadPassword(int(syscall.Stdin))
		password = string(passwordBytes)
		fmt.Println()
	}
	if ip == "" {
		var err error
		ip, err = getIp(callback, targets["rad_user_info"])
		if err != nil {
			fmt.Println("获取 IP 失败, 请检查配置的 URL 或手动指定 ip")
			return
		}
	}
	token, err := getChallenge(callback, username, ip, targets["get_challenge"])
	if err != nil {
		return
	}
	srunPortalLogin(callback, username, password, targets["srun_portal"], token, ip, os, acID, encVer)
}

func logout(username, ip string) {
	if username == "" {
		fmt.Print("[+] 请输入您的学号: ")
		fmt.Scanln(&username)
	}
	if ip == "" {
		var err error
		ip, err = getIp(callback, targets["rad_user_info"])
		if err != nil {
			fmt.Println("获取 IP 失败, 请检查配置的 URL")
			return
		}
	}
	srunPortalLogout(callback, username, ip, targets["srun_portal"])
}

func main() {
	fmt.Print(Banner)
	args, err := parseArguments()
	if err != nil {
		fmt.Printf("[!] 读取配置失败: %v\n", err)
		return
	}
	settings := applyArguments(args)
	if settings.Interface != "" {
		fmt.Fprintln(os.Stderr, "[!] --interface 仅支持 Linux 上的 Python 客户端")
		os.Exit(1)
	}
	if settings.OS == "" {
		settings.OS = "Windows"
	}
	if settings.AcID == "" {
		settings.AcID = ACID
	}
	if settings.EncVer == "" {
		settings.EncVer = ENC
	}
	if settings.Mode == "" {
		fmt.Print("[+] 请选择工作模式 [1]登录 [2]登出: ")
		fmt.Scanln(&settings.Mode)
	}
	switch settings.Mode {
	case ModeLogin:
		login(settings.Username, settings.Password, settings.IP, settings.OS, settings.AcID, settings.EncVer)
	case ModeLogout:
		logout(settings.Username, settings.IP)
	default:
		fmt.Println("[*] BYE")
	}
}
