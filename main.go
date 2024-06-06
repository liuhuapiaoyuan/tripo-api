package main

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
	"tripo-api/keymanager" // 替换为你的模块名称
)

type Response struct {
	Message string `json:"message"`
}

var km *keymanager.KeyManager // 全局变量

// 主函数
func main() {
	var err2 error
	km, err2 = keymanager.NewKeyManager("keys.db")
	if err2 != nil {
		fmt.Printf("项目初始化数据库失败了: %s\n", err2)
		return
	}

	http.HandleFunc("/task/query", queryTaskHandler)
	http.HandleFunc("/task/text_to_model", textToModelHandler)
	http.HandleFunc("/task/image_to_model", imageToModelHandler)
	http.HandleFunc("/upload/sync_url", syncURLHandler)

	http.HandleFunc("/create_key", km.CreateKeyHandler)
	http.HandleFunc("/remove_key", km.RemoveKeyHandler)
	http.HandleFunc("/", km.ListKeysHandler)

	fmt.Println("Server is listening on port 8000...")
	fmt.Println("访问： http://localhost:8000/")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}

func queryTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskId := r.URL.Query().Get("taskId")
	sync := r.URL.Query().Get("sync")

	// 从请求头获取授权码
	authorization, err := km.AllocateKey()
	if err != nil {
		fmt.Printf("无法分配到有效keyto allocate key: %s\n", err)
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	// 转发请求到目标API
	apiURL := fmt.Sprintf("https://api.tripo3d.ai/v2/openapi/task/%s", taskId)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", authorization)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to send request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)

	if sync == "1" {
		// {"data":{"output":{},"result":{},"progress":6,"status":"running","task_id":"40cf0f1a-08d0-42fd-87b3-488da51f7e88","type":"image_to_model","create_time":1717602310,"input":{}},"code":0}
		// 判断status是否为running
		if err != nil {
			println("queryTaskHandler: Failed to parse response body")
			http.Error(w, "Failed to parse response body", http.StatusInternalServerError)
			return
		}
		data, ok := response["data"].(map[string]interface{})
		if !ok {
			println("queryTaskHandler: Failed to parse response body cannot find data")
			http.Error(w, "Failed to parse response body", http.StatusInternalServerError)
			return
		}
		if data["status"] == "running" {
			//延迟1秒
			time.Sleep(2 * time.Second)
			queryTaskHandler(w, r)
			return
		}

	}
	// 将目标API的响应返回给前端
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	// 把response返回前端

	json.NewEncoder(w).Encode(response)
}

func textToModelHandler(w http.ResponseWriter, r *http.Request) {
	// 从请求头获取授权码
	authorization, err := km.AllocateKey()
	if err != nil {
		fmt.Printf("无法分配到有效keyto allocate key: %s\n", err)
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	// 读取请求体内容
	var requestBody map[string]interface{}
	err = json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	// 设置参数 type 为 text_to_model
	requestBody["type"] = "text_to_model"

	// 转换回 JSON 格式
	modifiedBody, err := json.Marshal(requestBody)
	if err != nil {
		http.Error(w, "Failed to encode modified request body", http.StatusInternalServerError)
		return
	}

	// 转发请求到目标API
	apiURL := "https://api.tripo3d.ai/v2/openapi/task"
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(string(modifiedBody)))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authorization)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to send request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	// 调用成功 加1
	// 判断响应状态是否是200
	if resp.StatusCode == http.StatusOK {
		km.IncreaseUsage(authorization, 20)
	}
	// 将目标API的响应返回给前端
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func imageToModelHandler(w http.ResponseWriter, r *http.Request) {
	fileToken := r.URL.Query().Get("file_token")

	// 从请求头获取授权码
	authorization, err := km.AllocateKey()
	if err != nil {
		fmt.Printf("无法分配到有效keyto allocate key: %s\n", err)
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	jsonStr := fmt.Sprintf(`{
		"type": "image_to_model",
		"file": {
			"type": "png",
			"file_token": "%s"
		}
	}`, fileToken)

	// 转发请求到目标API
	apiURL := "https://api.tripo3d.ai/v2/openapi/task"
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(jsonStr))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authorization)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to send request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		km.IncreaseUsage(authorization, 30)
	}
	// 将目标API的响应返回给前端
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

}

func syncURLHandler(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	Authorization := r.Header.Get("Authorization")
	if url == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}

	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to fetch the image", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	file, err := os.CreateTemp("", "upload-*.jpeg")
	if err != nil {
		http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(file.Name())

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		http.Error(w, "Failed to copy image to file", http.StatusInternalServerError)
		return
	}

	file.Seek(0, 0)

	// Prepare form data
	body := &strings.Builder{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", file.Name())
	if err != nil {
		http.Error(w, "Failed to create form file", http.StatusInternalServerError)
		return
	}
	_, err = io.Copy(part, file)
	if err != nil {
		http.Error(w, "Failed to copy file content", http.StatusInternalServerError)
		return
	}
	writer.Close()

	// Send POST request
	req, err := http.NewRequest("POST", "https://api.tripo3d.ai/v2/openapi/upload", strings.NewReader(body.String()))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", Authorization)

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		http.Error(w, "Failed to send request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// 将目标API的响应返回给前端
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
