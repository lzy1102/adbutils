package test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/lzy1102/adbutils"
)

var adb = adbutils.AdbClient{Host: "localhost", Port: 5037, SocketTime: 10}

func TestServerVersion(t *testing.T) {
	version := adb.ServerVersion()
	t.Logf("version: %d", version)
}

func Test_startApp(t *testing.T) {
	for _, device := range adb.DeviceList() {
		fmt.Println(device.Serial)
		device.AppStart("com.amazon.mShop.android.shopping", "")
		time.Sleep(10 * time.Second)
	}
}

func Test_forward(t *testing.T) {
	//adb.Shell("38ebb830", "forward tcp:12345 tcp:8080", false)
	for _, device := range adb.DeviceList() {
		fmt.Println(device.ForWard("tcp:12345", "tcp:8080", false))
		fmt.Println(device.Serial, device.ForwardList())
		fmt.Println(device.SayHello())
		//fmt.Println(device.Client.Shell(device.Serial, "forward tcp:12345 tcp:8080", false))
	}

	//device := adb.Device(adbutils.SerialNTransportID{Serial: "38ebb830"})
	//list := device.ForwardList()
	//fmt.Println(list)
	//fmt.Println(device.ForWardPort("8080"))
}

func TestConnect(t *testing.T) {
	// adb := adbutils.NewAdb("localhost", 5037, 10)
	for _, i := range adb.DeviceList() {
		adb.Connect(i.Serial)
		snNtid := adbutils.SerialNTransportID{
			Serial: i.Serial,
		}
		fmt.Println(adb.Device(snNtid).SayHello())
		// fmt.Println(adb.Device(snNtid).Push("/Users/sato/Desktop/go-scrcpy-client/scrcpy/scrcpy-server.jar", "/data/local/tmp/scrcpy-server.jar"))
	}

}

func Test_downloadADB(t *testing.T) {
	fmt.Println(adbutils.AdbPath())
	adb.Connect("192.168.50.142:5555")
	for _, i := range adb.DeviceList() {
		fmt.Println(i.Serial)
		//fmt.Println(i.StartTCPIP("5555"))
		// fmt.Println(adb.Device(snNtid).Push("/Users/sato/Desktop/go-scrcpy-client/scrcpy/scrcpy-server.jar", "/data/local/tmp/scrcpy-server.jar"))
	}
}

func Test_path(t *testing.T) {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	fmt.Println(exPath)
}

func Test_rutier(t *testing.T) {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		panic(errors.New("Can not get current file info"))
	}
	fmt.Println(file)
}

func Test_IP(t *testing.T) {
	fmt.Println(adbutils.AdbPath())
	//adb.Connect("192.168.50.142:5555")
	devices := adb.DeviceList()
	for _, device := range devices {
		fmt.Println("ip", device.Serial, device.WlanIp())
	}
}
