package packages

import (
	"net/http"

	"encoding/json"
	"fmt"
	"net/url"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/storage"
)

type OssUploader struct {
	// TODO: implement OssUploader
}
type RequestBody struct {
	Bucket    string `json:"bucket"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	Region    string `json:"region"`
	CDN       string `json:"cdn"`
	TargetUrl string `json:"url"`
}

// 1. 读取oss的配置
func NewOssUploader() *OssUploader {
	km := &OssUploader{}
	return km
}

func (km *OssUploader) Sync_url(w http.ResponseWriter, r *http.Request) {
	// 获取参数 AccessKey，SecretKey和Bucket
	var requestBody RequestBody

	// 读取请求体并解析 JSON 数据
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		// 处理解析错误
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{"code": 1,"msg":"parse request body fail", "data": {"err": "%s"}}`, err.Error())))
		return
	}

	// 使用解析后的参数
	bucket := requestBody.Bucket
	accessKey := requestBody.AccessKey
	secretKey := requestBody.SecretKey
	region := requestBody.Region
	cdn := requestBody.CDN
	targetUrl := requestBody.TargetUrl

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

func fetchPathnameFromURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return parsedURL.Path, nil
}
