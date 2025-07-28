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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	sync "sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tknie/log"
	"github.com/tknie/services"
	"google.golang.org/protobuf/proto"
)

const layout = "2006-01-02 15:04:05.000"

type statMqtt struct {
	mu          sync.Mutex
	mqttCounter uint64
	httpCounter uint64
}

type Entry struct {
	object       interface{}
	serialNumber string
}

type ProtocolHandler interface {
	CallHandler(*Entry)
}

var caller ProtocolHandler
var mqttStatMap = sync.Map{}
var mapStatMqtt = make(map[string]*statMqtt)
var Callback func(serialNumber string, data map[string]interface{})

func StatMqtt() string {
	var buffer bytes.Buffer
	for k, v := range mapStatMqtt {
		buffer.WriteString(fmt.Sprintf("  %s got http=%03d mqtt=%03d messages\n", k, v.httpCounter, v.mqttCounter))
	}
	return buffer.String()
}

func DisplayPayload(sn string, payload []byte) bool {
	log.Log.Debugf("Base64: %s", base64.RawStdEncoding.EncodeToString(payload))
	log.Log.Debugf("Payload %s", FormatByteBuffer("MQTT Body", payload))

	platform := &SendHeaderMsg{}
	err := proto.Unmarshal(payload, platform)
	if err != nil {
		log.Log.Errorf("Unable to parse message message %v: %v", payload, err)
	} else {
		switch platform.Msg.GetCmdId() {
		case 1:

			ih := &InverterHeartbeat{}
			err := proto.Unmarshal(platform.Msg.Pdata, ih)
			if err != nil {
				log.Log.Errorf("Unable to parse pdata message: %v", err)
			} else {
				log.Log.Debugf("-> InverterHearbeat %s\n", ih)
				caller.CallHandler(&Entry{object: ih, serialNumber: sn})

				if log.IsDebugLevel() {
					log.Log.Debugf("DynamicWatts   %v", ih.GetDynamicWatts())
					log.Log.Debugf("LowerLimit     %v", ih.GetLowerLimit())
					log.Log.Debugf("PermanentWatts %v", ih.GetPermanentWatts())
					log.Log.Debugf("UpperLimit     %v", ih.GetUpperLimit())
					log.Log.Debugf("InstallCountry %v", ih.GetInstallCountry())
					log.Log.Debugf("InvOnOff       %v", ih.GetInvOnOff())
					log.Log.Debugf("Pv10pVolt      %v", ih.GetPv1OpVolt())
					log.Log.Debugf("Pv1InputVolt   %v", ih.GetPv1InputVolt())
					log.Log.Debugf("Pv1InputWatts  %v", ih.GetPv1InputWatts())
					log.Log.Debugf("Pv20pVolt      %v", ih.GetPv2OpVolt())
					log.Log.Debugf("Pv2InputVolt   %v", ih.GetPv2InputVolt())
					log.Log.Debugf("Pv2InputWatts  %v", ih.GetPv2InputWatts())
					log.Log.Debugf("Timestamp      %v", ih.GetTimestamp())
					log.Log.Debugf("Time           %v", time.Unix(int64(ih.GetTimestamp()), 0))
				}
			}
		case 32:
			pp := &PowerPack{}
			err := proto.Unmarshal(platform.Msg.Pdata, pp)
			if err != nil {
				log.Log.Errorf("Unable to parse pdata message: %v", err)
			} else {
				log.Log.Debugf("Power Pack: %#v", pp)
				for _, p := range pp.SysPowerStream {
					caller.CallHandler(&Entry{object: p, serialNumber: sn})
				}
			}
		default:
			displayHeader(platform.Msg)
			log.Log.Infof("Unknown Cmd ID %d -> %s\n", platform.Msg.GetCmdId(), sn)
			log.Log.Infof("Base64: %s\n", base64.RawStdEncoding.EncodeToString(payload))
			return false
		}
	}
	return true
}

func GetStatEntry(serialNumber string) *statMqtt {
	if s, ok := mapStatMqtt[serialNumber]; ok {
		return s
	} else {
		stat := &statMqtt{}
		mapStatMqtt[serialNumber] = stat
		return stat
	}

}

// MessageHandler message handle called if MQTT event entered
func MessageHandler(_ mqtt.Client, msg mqtt.Message) {
	serialNumber := getSnFromTopic(msg.Topic())
	stat := GetStatEntry(serialNumber)
	stat.mu.Lock()
	defer stat.mu.Unlock()

	stat.mqttCounter++

	if e, ok := mqttStatMap.Load(msg.Topic()); ok {
		mqttStatMap.Store(msg.Topic(), e.(int)+1)
	} else {
		mqttStatMap.Store(msg.Topic(), 1)
	}
	if stat.mqttCounter%350 == 0 {
		services.ServerMessage("Received MQTT msgs: %04d", stat.mqttCounter)
		mqttStatMap.Range(func(key, value any) bool {
			log.Log.Infof("Received message of device %s = %d at %v\n", key, value.(int), time.Now().Format(layout))
			return true
		})
	}

	log.Log.Debugf("received message on topic %s; body (retain: %t):\n%s", msg.Topic(),
		msg.Retained(), FormatByteBuffer("MQTT Body", msg.Payload()))
	payload := msg.Payload()

	data := make(map[string]interface{})
	err := json.Unmarshal(payload, &data)
	if err == nil {
		log.Log.Debugf("JSON: %v", string(payload))
		if log.IsDebugLevel() {
			cmdId := int(data["cmdId"].(float64))
			log.Log.Debugf("-> CmdId   %03d", cmdId)
			log.Log.Debugf("-> CmdFunc %f", data["cmdFunc"].(float64))
			log.Log.Debugf("-> Version %s", data["version"].(string))
			log.Log.Debugf("ID           : %f", data["id"].(float64))
		}
		if _, ok := data["params"]; ok {
			data = data["params"].(map[string]interface{})
		}
		if _, ok := data["serial_number"]; !ok {
			data["serial_number"] = serialNumber
		}
		if _, ok := data["timestamp"]; !ok {
			data["timestamp"] = time.Now()
		}
		Callback(serialNumber, data)

		return
	}

	start := 0
	end := len(payload)
	index := bytes.Index(payload, []byte(serialNumber))
	if index != -1 {
		end = index + len(serialNumber)
	}
	log.Log.Debugf("Serial index 1: %d/%d %d:%d", index, len(payload), start, end)
	DisplayPayload(serialNumber, payload[start:end])
	start = end
	if len(payload) > index+len(serialNumber) {
		index = bytes.Index(payload[end:], []byte(serialNumber))
		if index != -1 {
			end = end + index + len(serialNumber)
		} else {
			end = len(payload)
		}
		log.Log.Debugf("Serial index 2: %d", index)
		DisplayPayload(serialNumber, payload[start:end])
	}

}

// getSnFromTopic extract serial number from topic
func getSnFromTopic(topic string) string {
	topicStr := strings.Split(topic, "/")
	return topicStr[len(topicStr)-1]
}
