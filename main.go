package main

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

type Response struct {
	Message string `json:"message"`
}

// 主函数
func main() {
	http.HandleFunc("/task/query", queryTaskHandler)
	http.HandleFunc("/task/text_to_model", textToModelHandler)
	http.HandleFunc("/task/image_to_model", imageToModelHandler)
	http.HandleFunc("/upload/sync_url", syncURLHandler)

	fmt.Println("Server is listening on port 8000...")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}

func queryTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskId := r.URL.Query().Get("taskId")

	// 从请求头获取授权码
	authorization := r.Header.Get("Authorization")

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

	// 将目标API的响应返回给前端
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func textToModelHandler(w http.ResponseWriter, r *http.Request) {
	// 从请求头获取授权码
	authorization := r.Header.Get("Authorization")

	// 读取请求体内容
	var requestBody map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&requestBody)
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

	// 将目标API的响应返回给前端
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func imageToModelHandler(w http.ResponseWriter, r *http.Request) {

	// 从请求头获取授权码
	authorization := r.Header.Get("Authorization")

	// 读取请求体内容
	var requestBody map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	// 设置参数 type 为 image_to_model
	requestBody["type"] = "image_to_model"

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
