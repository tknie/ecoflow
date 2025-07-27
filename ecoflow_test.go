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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tknie/log"
	"github.com/tknie/services"
)

const (
	accessToken = "${ECOFLOW_ACCESS_TOKEN}"
	secretKey   = "${ECOFLOW_SECRET_KEY}"
	converter   = "${ECOFLOW_CONVERTER}"
	battery     = "${ECOFLOW_BATTERY}"
)

func TestClientAllDevices(t *testing.T) {
	accessKey := os.ExpandEnv(accessToken)
	secretKey := os.ExpandEnv(secretKey)
	sn := os.ExpandEnv(converter)

	log.Log.Debugf("AccessKey: %v", accessKey)
	log.Log.Debugf("SecretKey: %v", secretKey)
	log.Log.Debugf("Serial Number: %v", sn)
	client := NewClient(accessKey, secretKey)
	m, err := client.GetDeviceList(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, "0", m.Code)
	assert.Equal(t, "Success", m.Message)
	assert.Len(t, m.Devices, 3)
	assert.Equal(t, sn, m.Devices[0].SN)
	sn = os.ExpandEnv(battery)
	assert.Equal(t, sn, m.Devices[1].SN)
}

func TestClientMicroConverter(t *testing.T) {
	accessKey := os.ExpandEnv(accessToken)
	secretKey := os.ExpandEnv(secretKey)
	value := 150
	if value > 600 || value < 0 {
		services.ServerMessage("Value %f out of range in 0:1000", value)
		return
	}

	sn := os.ExpandEnv(converter)

	log.Log.Debugf("AccessKey: %v", accessKey)
	log.Log.Debugf("SecretKey: %v", secretKey)
	log.Log.Debugf("Serial Number: %v", sn)
	client := NewClient(accessKey, secretKey)
	m, err := client.GetDeviceAllParameters(context.Background(), sn)
	assert.NoError(t, err)
	x, ok := m["20_1.batErrorInvLoadLimit"]
	fmt.Println(m)
	assert.True(t, ok)
	assert.NotNil(t, x)

	sn = os.ExpandEnv(battery)
	m, err = client.GetDeviceAllParameters(context.Background(), sn)
	assert.NoError(t, err)
	x, ok = m["pd.standbyMin"]
	fmt.Println(m)
	assert.True(t, ok)
	assert.NotNil(t, x)
}
