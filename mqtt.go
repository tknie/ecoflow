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
	"fmt"
	reflect "reflect"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tknie/log"
	"github.com/tknie/services"
)

var ecoclient *MqttClient

var devices *DeviceListResponse

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

// displayHeader log output display message header of MQTT Ecoflow data
func displayHeader(msg *Header) {
	if !log.IsDebugLevel() {
		return
	}
	log.Log.Debugf("-> Header  %s", msg)
	log.Log.Debugf("-> SM      %s", msg.GetDeviceSn())
	log.Log.Debugf("-> Version %d", msg.GetVersion())
	log.Log.Debugf("-> PayloadVersion %d", msg.GetPayloadVer())
	log.Log.Debugf("-> SRC     %d", msg.GetSrc())
	log.Log.Debugf("-> Dest    %d", msg.GetDest())
	log.Log.Debugf("-> Datalen %d", msg.GetDataLen())
	log.Log.Debugf("-> CmdId   %d", msg.GetCmdId())
	log.Log.Debugf("-> CmdFunc %d", msg.GetCmdFunc())
	log.Log.Debugf("-> DSRC    %d", msg.GetDSrc())
	log.Log.Debugf("-> DDest   %d", msg.GetDDest())
	log.Log.Debugf("-> NeedAcl %d", msg.GetNeedAck())
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
