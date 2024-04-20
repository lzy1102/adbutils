package adbutils

import (
	"errors"
	"fmt"
	_ "github.com/lzy1102/adbutils/binaries"
	"github.com/mholt/archiver/v3"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	OKAY            = "OKAY"
	FAIL            = "FAIL"
	DENT            = "DENT"
	DONE            = "DONE"
	DATA            = "DATA"
	TCP             = "tcp"
	UNIX            = "unix"
	DEV             = "dev"
	LOCAL           = "local"
	LOCALRESERVED   = "localreserved"
	LOCALFILESYSTEM = "localfilesystem"
	LOCALABSTRACT   = "localabstract"
	Windows         = "windows"
	Mac             = "darwin"
	Linux           = "linux"
	macAdbURL       = "https://dl.google.com/android/repository/platform-tools-latest-darwin.zip"
	linuxAdbURL     = "https://dl.google.com/android/repository/platform-tools-latest-linux.zip"
	WinAdbURL       = "https://dl.google.com/android/repository/platform-tools-latest-windows.zip"
)

func Uncompression(src, dst string) error {
	err := archiver.Unarchive(src, dst)
	if err != nil {
		// 处理错误
		return err
	}
	return nil
}

func Compression(src, dst string) error {
	var allFile []string
	filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return err
		}
		allFile = append(allFile, path)
		return nil
	})
	err := archiver.Archive(allFile, dst)
	if err != nil {
		// 处理错误
		return err
	}
	return nil
}

func checkServer(host string, port int) bool {
	_, err := net.Dial("tcp", fmt.Sprintf("%v:%v", host, port))
	return err == nil
}

func substr(s string, pos, length int) string {
	runes := []rune(s)
	l := pos + length
	if l > len(runes) {
		l = len(runes)
	}
	return string(runes[pos:l])
}

func getParentDirectory(dirctory string) string {
	return substr(dirctory, 0, strings.LastIndex(dirctory, "/"))
}
func getCurrentFile() string {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		panic(errors.New("Can not get current file info"))
	}
	return getParentDirectory(file)
}

func GetFreePort() int {
	conn, err := net.Listen("tcp", "127.0.0.1:0")
	defer conn.Close()
	if err != nil {
		log.Println("getFreePort error! ", err.Error())
		return 0
	}
	ipPort := strings.Split(conn.Addr().String(), ":")
	port, _ := strconv.Atoi(ipPort[len(ipPort)-1])
	return port
}

// AdbConnection region AdbConnection

type AdbConnection struct {
	Host string
	Port int
	Conn net.Conn
}

func (adbConnection AdbConnection) safeConnect() (*net.Conn, error) {
	conn, err := adbConnection.createSocket()
	if err != nil {
		switch reflect.TypeOf(err) {
		case reflect.TypeOf(&net.OpError{}):
			cmd := exec.Command(AdbPath(), "start-server")
			err = cmd.Start()
			if err != nil {
				log.Println("start adb error: ", err.Error())
				return nil, err
			}
			err = cmd.Wait()
			if err != nil {
				log.Println("start adb error: ", err.Error())
				return nil, err
			}
			conn, err = adbConnection.createSocket()
			if err != nil {
				log.Println("restart adb error! ", err.Error())
				return nil, err
			}
			return conn, nil
		default:
			log.Println("unknown error! ", err.Error())
			return nil, err
		}
	}
	return conn, nil
}

func (adbConnection AdbConnection) SetTimeout(timeOut time.Duration) error {
	if timeOut != 0 {
		var err error
		err = adbConnection.Conn.SetDeadline(time.Now().Add(time.Second * timeOut))
		if err != nil {
			panic(err.Error())
			return err
		}
	}
	return nil
}

func (adbConnection AdbConnection) createSocket() (*net.Conn, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%v:%d", adbConnection.Host, adbConnection.Port))
	if err != nil {
		return nil, err
	}
	return &conn, nil
}

func (adbConnection AdbConnection) Close() {
	err := adbConnection.Conn.Close()
	if err != nil {
		return
	}
}

func (adbConnection AdbConnection) Read(n int) []byte {
	return adbConnection.readFully(n)
}

func (adbConnection AdbConnection) readFully(n int) []byte {
	//t := 0
	//buffer := make([]byte, n)
	//result := bytes.NewBuffer(nil)
	//for t < n {
	//	length, err := adbConnection.Conn.Read(buffer[0:n])
	//	if err != nil {
	//		if err == io.EOF {
	//			break
	//		}
	//		break
	//	}
	//	if length == 0 {
	//		break
	//	}
	//	result.Write(buffer[0:length])
	//	t += length
	//}
	//return result.Bytes()
	buffer := make([]byte, n)

	_, err := io.ReadFull(adbConnection.Conn, buffer)
	if err != nil {
		return nil
	}
	return buffer
}

func (adbConnection AdbConnection) SendCommand(cmd string) {
	msg := fmt.Sprintf("%04x%s", len(cmd), cmd)
	_, err := adbConnection.Conn.Write([]byte(msg))
	if err != nil {
		log.Println("write error!", err.Error())
		return
	}
}

func (adbConnection AdbConnection) ReadString(n int) string {
	res := adbConnection.Read(n)
	return string(res)
}

func (adbConnection AdbConnection) ReadStringBlock() string {
	str := adbConnection.ReadString(4)
	if len(str) == 0 {
		log.Println("receive data error connection closed")
	}
	size, _ := strconv.ParseUint(str, 16, 32)
	return adbConnection.ReadString(int(size))
}

func (adbConnection AdbConnection) ReadUntilClose() string {
	buf := []byte{}
	for {
		chunk := adbConnection.Read(4096)
		if len(chunk) == 0 {
			break
		}

		buf = append(buf, chunk...)
	}
	return string(buf)
}

func (adbConnection AdbConnection) CheckOkay() {
	data := adbConnection.ReadString(4)
	if data == FAIL {
		log.Println(fmt.Sprintf("receive data: %v connection closed", data))
	} else if data == OKAY {
		return
	}
	log.Println(fmt.Sprintf("Unknown data: %v", data))
}

// end region AdbConnection

// AdbClient region AdbClient
type AdbClient struct {
	Host       string
	Port       int
	SocketTime time.Duration
}

func downloadFile(url string, localPath string, wg *sync.WaitGroup) error {
	defer func() {
		if wg != nil {
			wg.Done()
		}
	}()
	var (
		buf     = make([]byte, 32*1024)
		written int64
	)
	tmpFilePath := localPath
	client := new(http.Client)
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	file, err := os.Create(tmpFilePath)
	defer file.Close()
	if err != nil {
		return err
	}
	fileSize, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 32)
	if err != nil {
		log.Println(err.Error())
	}
	defer resp.Body.Close()
	if resp.Body == nil {
		return errors.New("body is null")
	}
	for {
		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			nw, ew := file.Write(buf[0:nr])
			log.Println(fmt.Sprintf("Download %v Done:%dKb, Total:%dKb, Process:%.2f", url, written/1024, fileSize, float32(written)/float32(fileSize)))
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	//if err == nil {
	//	err = os.Rename(tmpFilePath, localPath)
	//}
	return err
}

func createDir(path string) bool {
	_exist, _err := pathExists(path)
	if _err != nil {
		return true
	}
	if _exist {
		return true
	} else {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return true
		}
	}
	return false
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
func copyFile(srcPath, dstPath string) error {
	// 打开源文件
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	// 创建目标文件
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	// 将源文件内容复制到目标文件
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	// 确保所有内容都已在磁盘上
	err = dstFile.Sync()
	if err != nil {
		return err
	}
	return nil
}

func AdbPath() string {
	// so ugly
	//currentPath := getCurrentFile()
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	currentPath := filepath.Dir(ex)
	fmt.Println(currentPath)
	platform := runtime.GOOS
	adbPath := ""
	subPath := "mac"
	url := macAdbURL
	if platform == Linux {
		subPath = Linux
		url = linuxAdbURL
	} else if platform == Windows {
		subPath = "win"
		url = WinAdbURL
	}
	dir, _ := filepath.Abs(path.Join(currentPath, "binaries", subPath))
	os.MkdirAll(dir, 777)
	exist, err := pathExists(dir)
	if err != nil {
		log.Println(err.Error())
		return ""
	}
	if !exist {
		createDir(dir)
	}
	adbPath, _ = filepath.Abs(path.Join(dir, "adb"))
	if platform == Windows {
		adbPath, _ = filepath.Abs(path.Join(dir, "adb.exe"))
	}
	exist, _ = pathExists(adbPath)
	if !exist {
		os.TempDir()
		tmp := strings.Split(url, "/")
		localPath := tmp[len(tmp)-1]
		fmt.Println(localPath)
		downloadFile(url, localPath, nil)
		Uncompression(localPath, "./tmp")
		filepath.Walk("./tmp", func(path1 string, info fs.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			abs, err := filepath.Abs(path.Join(dir, info.Name()))
			if err != nil {
				return err
			}
			copyFile(path1, abs)
			_ = os.Chmod(abs, 0777)
			return nil
		})
		os.RemoveAll("./tmp")
		os.RemoveAll(localPath)
		//if platform == Windows {
		//	AdbWinApiPath, _ := filepath.Abs(path.Join(dir, "AdbWinApi.dll"))
		//	AdbWinUsbApiPath, _ := filepath.Abs(path.Join(dir, "AdbWinUsbApi.dll"))
		//	wg := &sync.WaitGroup{}
		//	wg.Add(3)
		//	err = downloadFile(url+"/adb.exe", adbPath, wg)
		//	err = downloadFile(url+"/AdbWinApi.dll", AdbWinApiPath, wg)
		//	err = downloadFile(url+"/AdbWinUsbApi.dll", AdbWinUsbApiPath, wg)
		//	wg.Wait()
		//	if err != nil {
		//		log.Println("get adb error!", err.Error())
		//	}
		//} else {
		//	err = downloadFile(url, adbPath, nil)
		//	if err != nil {
		//		log.Println("get adb error!", err.Error())
		//	}
		//	_ = os.Chmod(adbPath, 0777)
		//}
	}
	return adbPath
}

func (adb *AdbClient) connect() *AdbConnection {
	adbConnection := &AdbConnection{
		Host: adb.Host,
		Port: adb.Port,
	}
	conn, err := adbConnection.safeConnect()
	if err != nil {
		log.Println("get connect error: ", err.Error())
	}
	adbConnection.Conn = *conn
	return adbConnection

}

func (adb *AdbClient) ServerVersion() int {
	c := adb.connect()
	c.SendCommand("host:version")
	c.CheckOkay()
	res := c.ReadStringBlock()
	l, _ := strconv.Atoi(res)
	return l + 16
}

func (adb *AdbClient) ServerKill() {
	if checkServer(adb.Host, adb.Port) {
		c := adb.connect()
		c.SendCommand("host:kill")
		c.CheckOkay()
	}
}

func (adb *AdbClient) WaitFor() {
	// pass
}

func (adb *AdbClient) Connect(addr string) string {
	//addr (str): adb remote address [eg: 191.168.0.1:5555]
	c := adb.connect()
	c.SendCommand("host:connect:" + addr)
	return c.ReadStringBlock()
}

func (adb *AdbClient) Disconnect(addr string, raiseErr bool) string {
	//addr (str): adb remote address [eg: 191.168.0.1:5555]
	c := adb.connect()
	c.SendCommand("host:disconnect:" + addr)
	return c.ReadStringBlock()
}

type SerialNTransportID struct {
	Serial      string
	TransportID int
}

func (adb *AdbClient) Shell(serial string, command string, stream bool) interface{} {
	snNtid := SerialNTransportID{Serial: serial}
	return adb.Device(snNtid).Shell(command, stream, adb.SocketTime)
}

func (adb *AdbClient) DeviceList() []AdbDevice {
	var res []AdbDevice
	c := adb.connect()
	c.SendCommand("host:devices")
	c.CheckOkay()
	outPut := c.ReadStringBlock()
	outPuts := strings.Split(outPut, "\n")
	for _, line := range outPuts {
		parts := strings.Split(strings.TrimSpace(line), "\t")
		if len(parts) != 2 {
			continue
		}
		if parts[1] == "device" {
			res = append(res, AdbDevice{ShellMixin{Client: adb, Serial: parts[0]}})
		}
	}
	return res
}

func (adb *AdbClient) Device(snNtid SerialNTransportID) AdbDevice {
	if snNtid.Serial != "" || snNtid.TransportID != 0 {
		return AdbDevice{ShellMixin{Client: adb, Serial: snNtid.Serial, TransportID: snNtid.TransportID}}
	}
	serial := os.Getenv("ANDROID_SERIAL")
	if serial != "" {
		ds := adb.DeviceList()
		if len(ds) == 0 {
			log.Println("Error: Can't find any android device/emulator")
		} else if len(ds) > 1 {
			log.Println("more than one device/emulator, please specify the serial number")
		} else {
			return ds[0]
		}
	}
	return AdbDevice{ShellMixin{Client: adb, Serial: snNtid.Serial, TransportID: snNtid.TransportID}}
}

func NewAdb(host string, port int, timeOut time.Duration) *AdbClient {
	adb := &AdbClient{Host: host, Port: port, SocketTime: time.Second * timeOut}
	return adb
}

// end region AdbClient
