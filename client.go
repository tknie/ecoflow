/*
* Copyright 2025 Thorsten A. Knieling
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*    http://www.apache.org/licenses/LICENSE-2.0
*
 */

package ecoflow

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	ecoflowAPI            = "https://api.ecoflow.com"
	deviceListPath        = "/iot-open/sign/device/list"
	getAllQuotePath       = "/iot-open/sign/device/quota/all"
	setDeviceFunctionPath = "/iot-open/sign/device/quota"
	getDeviceFunctionPath = "/iot-open/sign/device/quota"
)

type ModuleType int

const (
	ModuleTypePd       ModuleType = 1
	ModuleTypeBms      ModuleType = 2
	ModuleTypeInv      ModuleType = 3
	ModuleTypeBmsSlave ModuleType = 4
	ModuleTypeMppt     ModuleType = 5
)

const (
	accessKeyHeader = "accessKey"
	nonceHeader     = "nonce"
	timestampHeader = "timestamp"
	signHeader      = "sign"
)

type Client struct {
	httpClient  *http.Client //can be customized if required
	accessToken string
	secretToken string
}

type DeviceListResponse struct {
	Code            string       `json:"code"`
	Message         string       `json:"message"`
	Devices         []DeviceInfo `json:"data"`
	EagleEyeTraceID string       `json:"eagleEyeTraceId"`
	Tid             string       `json:"tid"`
}

type DeviceInfo struct {
	SN     string `json:"sn"`
	Online int    `json:"online"`
}

type HttpRequest struct {
	httpClient        *http.Client
	method            string
	uri               string
	requestParameters map[string]interface{}
	accessKey         string
	secretKey         string
	getSignParameters func() *signParameters //required for unit testing
}

type signParameters struct {
	nonce       string
	timestamp   string
	accessKey   string
	sign        string
	queryParams string
}

type CmdSetResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (r *HttpRequest) generateSignParameters() *signParameters {
	nonce := generateNonce()
	timestamp := generateTimestamp()
	queryParams := generateQueryParams(r.requestParameters)
	return &signParameters{
		queryParams: queryParams,
		nonce:       nonce,
		timestamp:   timestamp,
		accessKey:   r.accessKey,
		sign:        r.generateSign(queryParams, nonce, timestamp),
	}
}

func generateQueryParams(data map[string]interface{}) string {
	var result []string

	// Process top-level map keys
	for k, v := range data {
		result = append(result, processValue(k, v)...)
	}

	// Sort results by ASCII value
	sort.Strings(result)

	// Concatenate results with & separator
	return strings.Join(result, "&")
}

func processValue(prefix string, value interface{}) []string {
	var result []string
	switch v := value.(type) {
	case map[string]interface{}:
		for k, nestedValue := range v {
			// Recursively process nested maps
			nestedPrefix := prefix + "." + k
			result = append(result, processValue(nestedPrefix, nestedValue)...)
		}
	case []interface{}:
		for i, item := range v {
			// Recursively process items in arrays
			nestedPrefix := prefix + "[" + strconv.Itoa(i) + "]"
			result = append(result, processValue(nestedPrefix, item)...)
		}
	case string:
		result = append(result, prefix+"="+v)
	case int:
		result = append(result, prefix+"="+strconv.Itoa(v))
	case float64:
		result = append(result, prefix+"="+strconv.FormatFloat(v, 'f', -1, 64))
	case bool:
		result = append(result, prefix+"="+strconv.FormatBool(v))
	}
	return result
}

func (r *HttpRequest) generateSign(queryString, nonce, timestamp string) string {
	keyValueString := r.getKeyValueString(queryString, nonce, timestamp)
	return encryptHmacSHA256(keyValueString, r.secretKey)
}

func encryptHmacSHA256(message string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	sha := hex.EncodeToString(h.Sum(nil))
	return sha
}

// timestamp is a UTC timestamp (in nano)
func generateTimestamp() string {
	return fmt.Sprint(time.Now().UnixNano())
}

// nonce is a random int with 6 digits
func generateNonce() string {
	return strconv.Itoa(rand.Intn(900000) + 100000)
}

func (r *HttpRequest) getKeyValueString(queryString string, nonce string, timestamp string) string {
	keyValueString := accessKeyHeader + "=" + r.accessKey + "&" +
		nonceHeader + "=" + nonce + "&" +
		timestampHeader + "=" + timestamp

	if queryString != "" {
		keyValueString = queryString + "&" + keyValueString
	}
	return keyValueString
}

// NewClient with default http client
func NewClient(accessToken, secretToken string) *Client {
	c := &Client{
		httpClient:  &http.Client{},
		accessToken: accessToken,
		secretToken: secretToken,
	}

	return c
}

type CmdSetRequest struct {
	Id          string                 `json:"id"`
	OperateType string                 `json:"operateType,omitempty"`
	ModuleType  ModuleType             `json:"moduleType,omitempty"`
	CmdCode     string                 `json:"cmdCode,omitempty"`
	Sn          string                 `json:"sn"`
	Params      map[string]interface{} `json:"params"`
}

func NewHttpRequest(httpClient *http.Client, method string, uri string, params map[string]interface{}, accessKey, secretKey string) *HttpRequest {
	r := &HttpRequest{
		httpClient:        httpClient,
		method:            method,
		uri:               uri,
		requestParameters: params,
		accessKey:         accessKey,
		secretKey:         secretKey,
	}

	//required for unit testing
	r.getSignParameters = func() *signParameters {
		return r.generateSignParameters()
	}

	return r
}

// GetDeviceList executes a request to get the list of devises linked to the user account. Shared devices are not included
// If the response parameter "code" is not 0, then there is an error. Error code and error message are returned
func (c *Client) GetDeviceList(ctx context.Context) (*DeviceListResponse, error) {
	request := NewHttpRequest(c.httpClient, "GET", ecoflowAPI+deviceListPath, nil, c.accessToken, c.secretToken)
	response, err := request.Execute(ctx)
	if err != nil {
		return nil, err
	}
	var deviceResponse DeviceListResponse

	err = json.Unmarshal(response, &deviceResponse)
	if err != nil {
		return nil, err
	}

	if deviceResponse.Code != "0" {
		return &deviceResponse, fmt.Errorf("can't get device list, error code: %s, error message: %s", deviceResponse.Code, deviceResponse.Message)
	}
	return &deviceResponse, nil
}

func (r *HttpRequest) Execute(ctx context.Context) ([]byte, error) {
	signParams := r.getSignParameters()
	requestURI := r.uri + "?" + signParams.queryParams

	var reqBody bytes.Buffer

	if r.requestParameters != nil {
		reqBytes, _ := json.Marshal(r.requestParameters)
		reqBody.Write(reqBytes)
	}

	var httpReq *http.Request
	var err error

	switch r.method {
	case http.MethodGet:
		httpReq, err = http.NewRequestWithContext(ctx, http.MethodGet, requestURI, nil)
		if err != nil {
			return nil, err
		}
	case http.MethodPost:
		httpReq, err = http.NewRequestWithContext(ctx, http.MethodPost, r.uri, &reqBody)
		if err != nil {
			return nil, err
		}
		httpReq.Header.Add("Content-Type", "application/json;charset=UTF-8")
	case http.MethodPut:
		httpReq, err = http.NewRequestWithContext(ctx, http.MethodPut, r.uri, &reqBody)
		if err != nil {
			return nil, err
		}
		httpReq.Header.Add("Content-Type", "application/json;charset=UTF-8")

	default:
		slog.Error("Only POST and GET methods are supported so far")
		return nil, errors.New("unsupported http method")
	}

	httpReq.Header.Add(accessKeyHeader, r.accessKey)
	httpReq.Header.Add(nonceHeader, signParams.nonce)
	httpReq.Header.Add(timestampHeader, signParams.timestamp)
	httpReq.Header.Add(signHeader, signParams.sign)

	client := r.httpClient
	if client == nil {
		client = &http.Client{}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response status is failed|url=%s, statusCode=%s", requestURI, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// GetDevice get all device parameters for a specific device
// Use HTTP request to get the parameter information
func (c *Client) GetDeviceAllParameters(ctx context.Context, deviceSn, specific string) (map[string]interface{}, error) {
	return c.GetDeviceInfo(ctx, deviceSn, "data")
}

// GetDevice get all device parameters for a specific device
// Use HTTP request to get the parameter information
func (c *Client) GetDeviceInfo(ctx context.Context, deviceSn, specific string) (map[string]interface{}, error) {
	requestParams := make(map[string]interface{})
	requestParams["sn"] = deviceSn

	request := NewHttpRequest(c.httpClient, "GET", ecoflowAPI+getAllQuotePath, requestParams, c.accessToken, c.secretToken)
	response, err := request.Execute(ctx)
	if err != nil {
		fmt.Println("Error ... http request:", err)
		return nil, err
	}

	var jsonData map[string]interface{}
	err = json.Unmarshal(response, &jsonData)
	if err != nil {
		return nil, err
	}

	if code, ok := jsonData["code"].(string); !ok || code != "0" {
		return nil, fmt.Errorf("can't get parameters, error code %s", code)
	}

	if specific != "" {
		if tmpMap, ok := jsonData[specific].(map[string]interface{}); ok {
			return tmpMap, nil
		}
		return nil, errors.New("response is not valid, can't process it")
	}

	return jsonData, err
}
