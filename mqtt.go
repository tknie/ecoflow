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
	"context"
	"encoding/json"
	"fmt"
	"math"
	reflect "reflect"
	"sort"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tknie/log"
	"github.com/tknie/services"
)

var ecoclient *MqttClient

var devices *DeviceListResponse

var MqttDisable = false

var MessageHandler mqtt.MessageHandler

// InitMqtt initialize MQTT listener
func InitMqtt(user, password string) {
	services.ServerMessage("Initialize MQTT client")
	configuration := MqttClientConfiguration{
		Email:            user,
		Password:         password,
		OnConnect:        OnConnect,
		OnConnectionLost: OnConnectionLost,
		OnReconnect:      OnReconnect,
	}
	var err error
	ecoclient, err = NewMqttClient(context.Background(), configuration)
	if err != nil {
		services.ServerMessage("Shuting down ... error creating MQTT client: %v", err)
		log.Log.Fatalf("Error creating new MQTT client connection: %v", err)
	}
	ecoclient.Connect()
	log.Log.Debugf("Wait for Ecoflow disconnect")
	services.ServerMessage("Waiting for MQTT data")

}

// insertMqttData prepare MQTT data into column data for database storage
func insertMqttData(data map[string]interface{}) ([]string, [][]any) {
	keys := make([]string, 0)
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	columns := make([]any, 0)
	// prefix := ""
	fields := make([]string, 0)
	for _, k := range keys {
		v := data[k]
		name := "eco_" + strings.ReplaceAll(k, ".", "_")
		fields = append(fields, name)
		log.Log.Debugf(" %s=%v %T -> %s\n", k, v, v, name)
		switch val := v.(type) {
		case string:
			columns = append(columns, val)
		case float64:
			if val == math.Trunc(val) {
				columns = append(columns, int64(val))
			} else {
				columns = append(columns, val)
			}
		case time.Time:
			columns = append(columns, val)
		case []interface{}, map[string]interface{}:
			b, err := json.Marshal(val)
			if err != nil {
				services.ServerMessage("Error marshal: %#v", val)
				columns = append(columns, nil)
			} else {
				s := string(b)
				columns = append(columns, s)
			}
		default:
			services.ServerMessage("Unknown type %s=%T\n", k, v)
			log.Log.Errorf("Unknown type %s=%T\n", k, v)
		}
	}
	return fields, [][]any{columns}
}

// displayHeader log output display message header of MQTT Ecoflow data
func displayHeader(msg *Header) {
	if !log.IsDebugLevel() {
		return
	}
	log.Log.Debugf("-> Header  %s\n", msg)
	log.Log.Debugf("-> SM      %s\n", msg.GetDeviceSn())
	log.Log.Debugf("-> Version %d\n", msg.GetVersion())
	log.Log.Debugf("-> PayloadVersion %d\n", msg.GetPayloadVer())
	log.Log.Debugf("-> SRC     %d\n", msg.GetSrc())
	log.Log.Debugf("-> Dest    %d\n", msg.GetDest())
	log.Log.Debugf("-> Datalen %d\n", msg.GetDataLen())
	log.Log.Debugf("-> CmdId   %d\n", msg.GetCmdId())
	log.Log.Debugf("-> CmdFunc %d\n", msg.GetCmdFunc())
	log.Log.Debugf("-> DSRC    %d\n", msg.GetDSrc())
	log.Log.Debugf("-> DDest   %d\n", msg.GetDDest())
	log.Log.Debugf("-> NeedAcl %d\n", msg.GetNeedAck())
}

// OnConnect on connect open handler called if connetion is done
func OnConnect(client mqtt.Client) {
	for _, d := range devices.Devices {
		services.ServerMessage("Subscribe for MQTT entries of device %s", d.SN)
		err := ecoclient.SubscribeForParameters(d.SN, MessageHandler)
		if err != nil {
			log.Log.Errorf("Unable to subscribe for parameters %s: %v", d.SN, err)
		} else {
			log.Log.Infof("Subscribed to receive parameters %s", d.SN)
		}
	}
}

func GetTypeName(myvar interface{}) string {
	if t := reflect.TypeOf(myvar); t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		return t.Name()
	}
}

// OnConnectionLost on connection lost happened
func OnConnectionLost(_ mqtt.Client, err error) {
	log.Log.Errorf("Error connection lost: %v", err)
}

// OnReconnect on connection reconnection
func OnReconnect(mqtt.Client, *mqtt.ClientOptions) {
	log.Log.Infof("Reconnecting...")
}

func (m *MqttClient) SubscribeForParameters(deviceSn string, callback mqtt.MessageHandler) error {
	topicParams := fmt.Sprintf("/app/device/property/%s", deviceSn)
	return m.SubscribeToTopics([]string{topicParams}, callback)
}

func (m *MqttClient) SubscribeToTopics(topics []string, callback mqtt.MessageHandler) error {
	topicsMap := make(map[string]byte, len(topics))

	for _, t := range topics {
		topicsMap[t] = 1
	}

	token := m.Client.SubscribeMultiple(topicsMap, callback)
	token.Wait()
	return nil
}
