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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
)

const (
	ecoflowLoginUrl         = "https://api.ecoflow.com/auth/login"
	ecoflowScene            = "IOT_APP"
	ecoflowUserType         = "ECOFLOW"
	ecoflowCertificationUrl = "https://api.ecoflow.com/iot-auth/app/certification"
)

type MqttClientConfiguration struct {
	Email                string
	Password             string
	OnConnect            mqtt.OnConnectHandler
	OnConnectionLost     mqtt.ConnectionLostHandler
	OnReconnect          mqtt.ReconnectHandler
	MaxReconnectInterval time.Duration
}

type MqttClient struct {
	Client           mqtt.Client
	connectionConfig *MqttConnectionConfig
}

type MqttConnectionConfig struct {
	CertificateAccount  string `json:"certificateAccount"`
	CertificatePassword string `json:"certificatePassword"`
	Url                 string `json:"url"`
	Port                string `json:"port"`
	Protocol            string `json:"protocol"`
	UserId              string `json:"userId"`
}

type MqttLoginResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		User struct {
			UserId        string `json:"userId"`
			Email         string `json:"email"`
			Name          string `json:"name"`
			Icon          string `json:"icon"`
			State         int    `json:"state"`
			Regtype       string `json:"regtype"`
			CreateTime    string `json:"createTime"`
			Destroyed     string `json:"destroyed"`
			RegisterLang  string `json:"registerLang"`
			Source        string `json:"source"`
			Administrator bool   `json:"administrator"`
			Appid         int    `json:"appid"`
			CountryCode   string `json:"countryCode"`
		} `json:"user"`
		Token string `json:"token"`
	} `json:"data"`
}

type MqttCredentialsResponse struct {
	Code    string               `json:"code"`
	Message string               `json:"message"`
	Data    MqttConnectionConfig `json:"data"`
}

func NewMqttClient(ctx context.Context, config MqttClientConfiguration) (*MqttClient, error) {
	c, err := getMqttCredentials(ctx, config.Email, config.Password)
	if err != nil {
		return nil, err
	}
	var protocol = c.Protocol
	var broker = c.Url
	var port = c.Port
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%s", protocol, broker, port))
	opts.SetClientID(fmt.Sprintf("ANDROID_%s_%s", uuid.New(), c.UserId))
	opts.SetUsername(c.CertificateAccount)
	opts.SetPassword(c.CertificatePassword)
	opts.SetConnectRetry(true)
	if config.OnConnect != nil {
		opts.OnConnect = config.OnConnect
	}
	if config.OnConnectionLost != nil {
		opts.OnConnectionLost = config.OnConnectionLost
	}
	if config.OnReconnect != nil {
		opts.OnReconnecting = config.OnReconnect
	}
	//default value is 10 minutes
	if config.MaxReconnectInterval != 0 {
		opts.MaxReconnectInterval = config.MaxReconnectInterval
	}
	return &MqttClient{Client: mqtt.NewClient(opts), connectionConfig: c}, nil
}

func getMqttCredentials(ctx context.Context, email, password string) (*MqttConnectionConfig, error) {
	mqttLoginResponse, err := getLoginResponse(ctx, email, password)
	if err != nil {
		return nil, err
	}

	var params = make(map[string]string)
	params["userId"] = mqttLoginResponse.Data.User.UserId

	jsonParams, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	certReq, err := http.NewRequestWithContext(ctx, "GET", ecoflowCertificationUrl, bytes.NewReader(jsonParams))
	if err != nil {
		return nil, err
	}

	certReq.Header.Set("Authorization", "Bearer "+mqttLoginResponse.Data.Token)
	certReq.Header.Add("lang", "en_US")

	client := http.Client{}
	resp, err := client.Do(certReq)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var mqttConn *MqttCredentialsResponse
	err = json.Unmarshal(responseBody, &mqttConn)
	if err != nil {
		return nil, err
	}

	c := &mqttConn.Data
	c.UserId = mqttLoginResponse.Data.User.UserId

	return c, nil
}

func getLoginResponse(ctx context.Context, email string, password string) (*MqttLoginResponse, error) {
	var params = make(map[string]string)
	params["email"] = email
	params["password"] = base64.StdEncoding.EncodeToString([]byte(password))
	params["scene"] = ecoflowScene
	params["userType"] = ecoflowUserType
	jsonParams, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	loginReq, err := http.NewRequestWithContext(ctx, "POST", ecoflowLoginUrl, bytes.NewReader(jsonParams))
	if err != nil {
		return nil, err
	}

	loginReq.Header.Add("lang", "en_US")
	loginReq.Header.Add("content-type", "application/json")

	client := http.Client{}
	resp, err := client.Do(loginReq)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var mqttLoginResponse *MqttLoginResponse
	err = json.Unmarshal(responseBody, &mqttLoginResponse)
	if err != nil {
		return nil, err
	}

	return mqttLoginResponse, nil
}

func (m *MqttClient) Connect() error {
	if token := m.Client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}
