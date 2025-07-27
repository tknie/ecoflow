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
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tknie/log"
	"google.golang.org/protobuf/proto"
)

func TestProtoDecode(t *testing.T) {
	StartLog("ecoflow-test.log")
	test := os.Getenv("ECOFLOW_TESTPROTO")
	dst := make([]byte, 1000)
	b, err := base64.RawStdEncoding.Decode(dst, []byte(test))
	assert.NoError(t, err)
	assert.Equal(t, 136, b)
	dst = dst[0:136]
	platform := &SendHeaderMsg{}
	err = proto.Unmarshal(dst, platform)
	assert.NoError(t, err)
	log.Log.Debugf("%#v\n", platform)
	ih := &InverterHeartbeat{}
	err = proto.Unmarshal(platform.Msg.Pdata, ih)
	assert.NoError(t, err)
	log.Log.Debugf("%#v\n", ih)
	assert.Equal(t, uint32(1743087465), uint32(*ih.Timestamp))
	assert.Equal(t, []uint8(nil), ih.unknownFields)
}

func TestGenerateOnProto(t *testing.T) {
	platform := &SendHeaderMsg{}
	platform.Msg = &Header{}
	platform.Msg.Src = generateInt(32)
	platform.Msg.Dest = generateInt(53)
	platform.Msg.CmdFunc = generateInt(2)
	platform.Msg.CmdId = generateInt(int32(129))
	platform.Msg.DataLen = generateInt(2)
	platform.Msg.NeedAck = generateInt(1)
	platform.Msg.Seq = generateInt(513567)
	serial_number := "HW52ZDH4SF5J6396"
	platform.Msg.DeviceSn = &serial_number
	fmt.Printf("-> %#v\n", platform.Msg)
	pap := &PowerAckPack{}
	pap.SysSeq = generateUInt(1)
	pd, err := proto.Marshal(pap)
	assert.NoError(t, err)
	platform.Msg.Pdata = pd
	data, err := proto.Marshal(platform)
	assert.NoError(t, err)
	baseTest := []byte{0x0a, 0x28, 0x0a, 0x02, 0x08, 0x01, 0x10, 0x20, 0x18, 0x35, 0x40,
		0x02, 0x48, 0x81, 0x01, 0x50, 0x02, 0x58, 0x01, 0x70, 0x9f, 0xac, 0x1f, 0xca,
		0x01, 0x10, 0x48, 0x57, 0x35, 0x32, 0x5a, 0x44, 0x48, 0x34, 0x53, 0x46, 0x35,
		0x4a, 0x36, 0x33, 0x39, 0x36}
	base64Test := base64.RawStdEncoding.EncodeToString(baseTest)
	assert.Equal(t, base64Test, base64.RawStdEncoding.EncodeToString(data))
	assert.Equal(t, baseTest, data)
}

func TestGenerateOffProto(t *testing.T) {
	platform := &SendHeaderMsg{}
	platform.Msg = &Header{}
	platform.Msg.Src = generateInt(32)
	platform.Msg.Dest = generateInt(53)
	platform.Msg.CmdFunc = generateInt(2)
	platform.Msg.CmdId = generateInt(int32(129))
	platform.Msg.NeedAck = generateInt(1)
	platform.Msg.Seq = generateInt(718223)
	serial_number := "HW52ZDH4SF5J6396"
	platform.Msg.DeviceSn = &serial_number
	fmt.Printf("-> %#v\n", platform.Msg)
	data, err := proto.Marshal(platform)
	assert.NoError(t, err)
	baseTest := []byte{0x0a, 0x22, 0x10, 0x20, 0x18, 0x35, 0x40, 0x02, 0x48,
		0x81, 0x01, 0x58, 0x01, 0x70, 0x8f, 0xeb, 0x2b, 0xca, 0x01, 0x10,
		0x48, 0x57, 0x35, 0x32, 0x5a, 0x44, 0x48, 0x34, 0x53, 0x46, 0x35,
		0x4a, 0x36, 0x33, 0x39, 0x36}
	base64Test := base64.RawStdEncoding.EncodeToString(baseTest)
	assert.Equal(t, base64Test, base64.RawStdEncoding.EncodeToString(data))
	assert.Equal(t, baseTest, data)
}

func generateInt(value int32) *int32 {
	return &value
}

func generateUInt(value uint32) *uint32 {
	return &value
}
