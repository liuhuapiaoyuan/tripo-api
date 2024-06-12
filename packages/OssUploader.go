package packages

import (
	"net/http"

	"bytes"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/storage"
)

type OssUploader struct {
	// TODO: implement OssUploader
}

// 1. 读取oss的配置
func NewOssUploader() *OssUploader {
	km := &OssUploader{}
	return km
}

func (km *OssUploader) Sync_url(w http.ResponseWriter, r *http.Request) {
	// 获取参数 AccessKey，SecretKey和Bucket
	bucket := r.FormValue("bucket")
	accessKey := r.FormValue("accessKey")
	secretKey := r.FormValue("secretKey")
	region := r.FormValue("region")
	cdn := r.FormValue("cdn")
	targetUrl := r.FormValue("url")
	pathName, err := fetchPathnameFromURL(targetUrl)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		// 进行错误处理或者日志记录
		w.Write([]byte(fmt.Sprintf(`{"code": 1,"msg":"upload fail", "data": {"path": "%s"}}`, pathName)))
		return
	}
	mac := auth.New(accessKey, secretKey)
	realRegion, res := storage.GetRegionByID(storage.RegionID(region))

	if !res {
		// 进行错误处理或者日志记录
		w.Write([]byte(fmt.Sprintf(`{"code": 1,"msg":"region unknown", "data": {"region": "%s"}}`, region)))
		return
	}
	bucketManager := storage.NewBucketManager(mac, &storage.Config{
		Region: &realRegion,
	})
	// 指定保存的key
	fetchRet, err := bucketManager.Fetch(targetUrl, bucket, pathName)
	if err != nil {
		w.Write([]byte(fmt.Sprintf(`{"code": 1,"msg":"upload fail: %s", "data": {"path": "%s"}}`, err.Error(), pathName)))
	} else {
		//fetchRet转成json
		fetchRetJsonStr, err := json.Marshal(fetchRet)
		if err != nil {
			w.Write([]byte(fmt.Sprintf(`{"code": 1,"msg":"upload fail: %s", "data": {"path": "%s"}}`, err.Error(), pathName)))
			return
		}

		// 如果cdn存在，则修改path
		if cdn != "" {
			pathName = fmt.Sprintf("%s/%s", cdn, pathName)
		}
		w.Write([]byte(fmt.Sprintf(`{"code": 0,"msg":"upload success ", "data": {"path": "%s" , "result": %s}}`, pathName, fetchRetJsonStr)))
	}
	// // 不指定保存的key，默认用文件hash作为文件名
	// fetchRet, err = bucketManager.FetchWithoutKey(resURL, bucket)
	// if err != nil {
	// 	fmt.Println("fetch error,", err)
	// } else {
	// 	fmt.Println(fetchRet.String())
	// }

}

type UploadRequest struct {
	URL    string `json:"url"`
	Host   string `json:"host"`
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
}

func fetchHostFromURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return parsedURL.Host, nil
}
func fetchPathnameFromURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return parsedURL.Path, nil
}

// 上传完enjain
func uploadFile(accessToken string, uploadURL, region string, bucket string, key string) error {
	host, err := fetchHostFromURL(uploadURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %v", err)
	}

	requestBody := UploadRequest{
		URL:    uploadURL,
		Host:   host,
		Bucket: bucket,
		Key:    key,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("https://api-%s.qiniuapi.com/sisyphus/fetch", region), bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("UpToken %s", accessToken))

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad response status: %s", resp.Status)
	}

	return nil
}
