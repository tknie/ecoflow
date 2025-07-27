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
	"log/slog"
	"time"

	"github.com/tknie/services"
)

var quit = make(chan struct{})

// RefreshDeviceList refresh device list using HTTP device list request
func (client *Client) RefreshDeviceList() {
	//get all linked ecoflow devices. Returns SN and online status
	list, err := client.GetDeviceList(context.Background())
	if err != nil {
		services.ServerMessage("Error getting device list: %v", err)
	} else {
		devices = list
	}
}

// SetEnvironmentPowerConsumption set new environment consumption value
func (client *Client) SetEnvironmentPowerConsumption(converter string, value float64) {

	params := make(map[string]interface{})
	// Ecoflow need to set a value times by 10
	// e.g. 200 watt needs value 2000
	params["permanentWatts"] = value * 10
	cmdReq := CmdSetRequest{
		Id:      fmt.Sprint(time.Now().UnixMilli()),
		CmdCode: "WN511_SET_PERMANENT_WATTS_PACK",
		Sn:      converter,
		Params:  params,
	}

	jsonData, err := json.Marshal(cmdReq)
	if err != nil {
		services.ServerMessage("Error marshal data: %v", err)
		return
	}

	var req map[string]interface{}

	err = json.Unmarshal(jsonData, &req)
	if err != nil {
		services.ServerMessage("Error unmarshal data: %v", err)
		return
	}
	cmd, err := client.SetDeviceParameter(context.Background(), req)

	if err != nil {
		services.ServerMessage("Error set device parameter: %v", err)
	} else {
		services.ServerMessage("Set device parameter to %f: %s", value, cmd.Message)
	}
}

func (c *Client) SetDeviceParameter(ctx context.Context, request map[string]interface{}) (*CmdSetResponse, error) {
	slog.Debug("SetDeviceParameter", "request", request)

	r := NewHttpRequest(c.httpClient, "PUT", ecoflowAPI+setDeviceFunctionPath, request, c.accessToken, c.secretToken)

	response, err := r.Execute(ctx)
	if err != nil {
		return nil, err
	}

	slog.Debug("SetDeviceParameter", "response", string(response))

	var cmdResponse *CmdSetResponse
	err = json.Unmarshal(response, &cmdResponse)
	if err != nil {
		return nil, err
	}

	return cmdResponse, nil
}
