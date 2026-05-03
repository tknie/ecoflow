package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tknie/ecoflow"
	"github.com/tknie/log"
)

var serialNumberConverter = ""
var client *ecoflow.Client

func prepareEcoflow() {

	accessKey := os.ExpandEnv("${ECOFLOW_ACCESS_KEY}")
	secretKey := os.ExpandEnv("${ECOFLOW_SECRET_KEY}")

	log.Log.Debugf("AccessKey: %v", accessKey)
	log.Log.Debugf("SecretKey: %v", secretKey)
	client = ecoflow.NewClient(accessKey, secretKey)
	client.RefreshDeviceList()
	serialNumberConverter = os.Getenv("ECOFLOW_DEVICE_SN")

}

func ListEcoflowDevices() {
	prepareEcoflow()
	devs, err := client.GetDeviceList(context.Background())
	if err != nil {
		fmt.Println("List device error:", err)
		return
	}
	for i, d := range devs.Devices {
		fmt.Println(i, d.Online, d.SN)
		m, err := client.GetDeviceAllParameters(context.Background(), d.SN)
		if err != nil {
			fmt.Println("Get info of device error:", err)
			return
		}
		dumpMap(0, m)
	}
}

func dumpMap(level int, m map[string]interface{}) {
	prefix := " " + strings.Repeat("  ", level*2)
	for k, v := range m {
		switch v.(type) {
		case map[string]interface{}:
			fmt.Printf(prefix+"%s:\n", k)
			dumpMap(level+1, v.(map[string]interface{}))
		default:
			if s, ok := v.(string); ok {
				fmt.Printf(prefix+"%s=%s\n", k, s)
			} else {
				fmt.Printf(prefix+"%s=%v\n", k, v)
			}
		}
	}
}

func SetCarACOn(sn string, turnOn bool) {
	prepareEcoflow()

	resp, err := client.SetCarACOn(sn, turnOn)

	fmt.Println(err, resp)
}

func main() {
	list := false
	flag.BoolVar(&list, "l", false, "List all devices")
	flag.Parse()

	if list {
		ListEcoflowDevices()
	}
}
