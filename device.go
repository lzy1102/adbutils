package adbutils

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type ShellMixin struct {
	Client      *AdbClient
	Serial      string
	TransportID int
	Properties  map[string]string
}

func (mixin ShellMixin) run(cmd string) interface{} {
	//fmt.Println("cmd ", cmd)
	return mixin.Client.Shell(mixin.Serial, cmd, false)
}

func (mixin ShellMixin) SayHello() string {
	content := "hello from " + mixin.Serial
	res := mixin.run("echo " + content)
	return res.(string)
}

func (mixin ShellMixin) SwitchScreen(status bool) {
	KeyMap := map[bool]string{
		true:  "224",
		false: "223",
	}
	mixin.KeyEvent(KeyMap[status])
}

func (mixin ShellMixin) SwitchAirPlane(status bool) {
	base := "settings put global airplane_mode_on"
	am := "am broadcast -a android.intent.action.AIRPLANE_MODE --ez state"
	if status {
		base += "1"
		am += "true"
	} else {
		base += "0"
		am += "false"
	}
	mixin.run(base)
	mixin.run(am)
}

func (mixin ShellMixin) SwitchWifi(status bool) {
	cmdMap := map[bool]string{
		true:  "svc wifi enable",
		false: "svc wifi disable",
	}
	mixin.run(cmdMap[status])
}

func (mixin ShellMixin) KeyEvent(keyCode string) string {
	res := mixin.run("input keyevent " + keyCode)
	return res.(string)
}

func (mixin ShellMixin) CLick(x, y int) {
	mixin.run(fmt.Sprintf("input tap %v %v", x, y))
}

func (mixin ShellMixin) Swipe(x, y, tox, toy int, duration time.Duration) {
	mixin.run(fmt.Sprintf("input swipe %v %v %v %v %v", x, y, tox, toy, duration*1000))
}

func (mixin ShellMixin) SendKeys(text string) {
	// TODO escapeSpecialCharacters
	mixin.run("input text " + text)
}

func (mixin ShellMixin) escapeSpecialCharacters(text string) {}

func (mixin ShellMixin) WlanIp() string {
	//res := mixin.run("ifconfig wlan0")
	res := mixin.run("ip addr show wlan0|grep 'inet '| cut -d' ' -f6 | cut -d/ -f1")
	ipInfo := res.(string)
	fmt.Println(ipInfo)
	return ipInfo
	//return ipInfo
	// TODO regrex
	re := regexp.MustCompile(`(?:\d{1,3}\.){3}\d{1,3}`) // 匹配 IPv4 地址的正则表达式
	match := re.FindString(ipInfo)
	fmt.Println(match) // 输出: 192.168.50.130
	return match
	//res := mixin.run(`ip addr show wlan0 | grep 'inet ' | cut -d' ' -f6 | cut -d/ -f1`)
	//ipInfo := res.(string)
	//return ipInfo
}

func (mixin ShellMixin) install(pathOrUrl string, noLaunch bool, unInstall bool, silent bool, callBack func()) {
}

func (mixin ShellMixin) InstallRemote(remotePath string, clean bool) {
	res := mixin.run("pm install -r -t " + remotePath)
	resInfo := res.(string)
	if !strings.Contains(resInfo, "Success") {
		log.Println(resInfo)
	}
	if clean {
		mixin.run("rm " + remotePath)
	}
}

func (mixin ShellMixin) Uninstall(packageName string) {
	mixin.run("pm uninstall " + packageName)
}

func (mixin ShellMixin) GetProp(prop string) string {
	res := mixin.run("getprop " + prop)
	return strings.TrimSpace(res.(string))
}

func (mixin ShellMixin) ListPackages() []string {
	var result []string
	res := mixin.run("pm list packages")
	output := res.(string)
	for _, packageName := range strings.Split(output, "\n") {
		p := strings.TrimSpace(strings.TrimPrefix(packageName, "package:"))
		if p == "" {
			continue
		}
		result = append(result, p)
	}
	return result
}

func (mixin ShellMixin) ListPackages3() []string {
	var result []string
	res := mixin.run("pm list packages -3")
	output := res.(string)
	for _, packageName := range strings.Split(output, "\n") {
		p := strings.TrimSpace(strings.TrimPrefix(packageName, "package:"))
		if p == "" {
			continue
		}
		result = append(result, p)
	}
	return result
}

func (mixin ShellMixin) PackageInfo(packageName string) {
	// TODO
}

func (mixin ShellMixin) Rotation() {}

func (mixin ShellMixin) rawWindowSize() {}

func (mixin ShellMixin) WindowSize() {}

func (mixin ShellMixin) AppStart(packageName, activity string) {
	if activity != "" {
		mixin.run("am start -n " + packageName + "/" + activity)
	} else {
		//mixin.run("android.intent.category.LAUNCHER 1")
		mixin.run("monkey -p " + packageName + " -c " + "android.intent.category.LAUNCHER 1")
	}
}

func (mixin ShellMixin) AppStop(packageName string) {
	mixin.run("am force-stop " + packageName)
}

func (mixin ShellMixin) AppClear(packageName string) {
	mixin.run("pm clear " + packageName)
}

func (mixin ShellMixin) IsScreenOn() bool {
	res := mixin.run("dumpsys power")
	output := res.(string)
	return strings.Contains(output, "mHoldingDisplaySuspendBlocker=true")
}

func (mixin ShellMixin) OpenBrowser(url string) {
	mixin.run("am start -a android.intent.action.VIEW -d " + url)
}

func (mixin ShellMixin) DumpHierarchy() string {
	return ""
}

func (mixin ShellMixin) CurrentApp() string {
	return ""
}

func (mixin ShellMixin) Remove(path string) {
	mixin.run("rm " + path)
}

func (mixin ShellMixin) openTransport(command string, timeOut time.Duration) *AdbConnection {
	c, err := mixin.Client.connect()
	if err != nil {
		return nil
	}
	//fmt.Println("connect device success")
	//fmt.Println(command)
	if timeOut > 0 {
		// 这里修改了一下 使用c设置Conn的timeout
		err := c.SetTimeout(timeOut)
		if err != nil {
			return nil
		}
	}
	if command != "" {
		if mixin.TransportID > 0 {
			//fmt.Println(command)
			c.SendCommand("host-transport-id:" + fmt.Sprintf("%d:%s", mixin.TransportID, command))
			//  send_command(f"host-transport-id:{self._transport_id}:{command}")
			c.CheckOkay()
		} else if mixin.Serial != "" {
			cmd := "host-serial:" + fmt.Sprintf("%s:%s", mixin.Serial, command)
			c.SendCommand(cmd)
			//c.send_command(f"host-serial:{self._serial}:{command}")
			//cmd := "host:tport:serial:" + mixin.Serial
			//c.SendCommand(cmd)
			//c.CheckOkay()
			//c.SendCommand(command)
			c.CheckOkay()
		} else {
			log.Println("RuntimeError")
		}

		//c.CheckOkay()
	} else {
		if mixin.TransportID > 0 {
			c.SendCommand("host:transport-id:" + fmt.Sprintf("%d", mixin.TransportID))
			// c.send_command(f"host:transport-id:{self._transport_id}")
			c.CheckOkay()
		} else if mixin.Serial != "" {
			// # host:tport:serial:xxx is also fine, but receive 12 bytes
			// # recv: 4f 4b 41 59 14 00 00 00 00 00 00 00              OKAY........
			// # so here use host:transport
			//c.SendCommand("host:transport:" + mixin.Serial)

			//# host:tport:serial:xxx is also fine, but receive 12 bytes
			//# recv: 4f 4b 41 59 14 00 00 00 00 00 00 00              OKAY........
			//c.send_command(f"host:tport:serial:{self._serial}")
			//c.check_okay()
			//c.read(8)  # skip 8 bytes
			c.SendCommand(fmt.Sprintf("host:tport:serial:%s", mixin.Serial))
			c.CheckOkay()
			c.Read(8)
		} else {
			log.Println("RuntimeError")
		}

	}
	return c
}

type AdbDevice struct {
	ShellMixin
}

func (adbDevice AdbDevice) getWithCommand(cmd string) string {
	c := adbDevice.openTransport("", adbDevice.Client.SocketTime)
	c.SendCommand(strings.Join([]string{"host-serial", adbDevice.Serial, cmd}, ":"))
	c.CheckOkay()
	return c.ReadStringBlock()
}

func (adbDevice AdbDevice) GetState() string {
	return adbDevice.getWithCommand("get-state")
}

func (adbDevice AdbDevice) GetSerialNo() string {
	return adbDevice.getWithCommand("get-serialno")
}

func (adbDevice AdbDevice) GetDevPath() string {
	return adbDevice.getWithCommand("get-devpath")
}

func (adbDevice AdbDevice) GetFeatures() string {
	return adbDevice.getWithCommand("features")
}

func (adbDevice AdbDevice) Info() map[string]string {
	res := map[string]string{}
	res["serialno"] = adbDevice.GetSerialNo()
	res["devpath"] = adbDevice.GetDevPath()
	res["state"] = adbDevice.GetState()
	return res
}

func (adbDevice AdbDevice) String() {
	fmt.Printf("AdbDevice(serial=%s)\n", adbDevice.Serial)
}

func (adbDevice AdbDevice) Sync() Sync {
	return Sync{AdbClient: adbDevice.Client, Serial: adbDevice.Serial}
}

func (adbDevice AdbDevice) AdbOut(command string) string {
	ctx, cancelFunc := context.WithCancel(context.Background())
	commandWithPrefix := "-s " + adbDevice.Serial + " " + command
	cmd := exec.CommandContext(ctx, AdbPath(), strings.Split(commandWithPrefix, " ")...)
	stdErr, err := cmd.StderrPipe()
	stdOut, err := cmd.StdoutPipe()

	defer func() {
		cancelFunc()
		_ = stdErr.Close()
		_ = stdOut.Close()
		_ = cmd.Wait()
	}()
	if err != nil {
		log.Println(err.Error())
		return ""
	}
	err = cmd.Start()

	if err != nil {
		log.Println(err.Error())
		return ""
	}
	bytesOut, err := io.ReadAll(stdOut)
	if err != nil {
		log.Println(err.Error())
		return ""
	}
	bytesErr, err := io.ReadAll(stdErr)
	if err != nil {
		log.Println(err.Error())
		return ""
	}
	if len(bytesErr) != 0 {
		log.Println(string(bytesErr))
	}
	return strings.TrimSpace(string(bytesOut))
}

func (adbDevice AdbDevice) Shell(cmdargs string, stream bool, timeOut time.Duration) interface{} {
	c := adbDevice.openTransport("", timeOut)
	c.SendCommand("shell:" + cmdargs)
	c.CheckOkay()
	if stream {
		return c
	}
	output := c.ReadUntilClose()
	// 简单返回
	return output
}

func (adbDevice AdbDevice) StartTCPIP(port string) string {
	ctx, cancelFunc := context.WithCancel(context.Background())
	commandWithPrefix := "-s " + adbDevice.Serial + " " + "tcpip" + " " + port
	cmd := exec.CommandContext(ctx, AdbPath(), strings.Split(commandWithPrefix, " ")...)
	stdErr, err := cmd.StderrPipe()
	stdOut, err := cmd.StdoutPipe()

	defer func() {
		cancelFunc()
		_ = stdErr.Close()
		_ = stdOut.Close()
		_ = cmd.Wait()
	}()
	if err != nil {
		log.Println(err.Error())
		return ""
	}
	err = cmd.Start()

	if err != nil {
		log.Println(err.Error())
		return ""
	}
	bytesOut, err := io.ReadAll(stdOut)
	if err != nil {
		log.Println(err.Error())
		return ""
	}
	bytesErr, err := io.ReadAll(stdErr)
	if err != nil {
		log.Println(err.Error())
		return ""
	}
	if len(bytesErr) != 0 {
		log.Println(string(bytesErr))
	}
	return strings.TrimSpace(string(bytesOut))
}

func (adbDevice AdbDevice) ShellOutPut(cmd string) string {
	res := adbDevice.Client.Shell(adbDevice.Serial, cmd, false)
	return res.(string)
}

func (adbDevice AdbDevice) ForWard(local, remote string, noRebind bool) *AdbConnection {
	args := []string{"forward:"}
	if noRebind {
		args = append(args, "norebind:")
	}
	args = append(args, []string{local, ";", remote}...)
	//c := adbDevice.openTransport("", adbDevice.Client.SocketTime)
	//c.SendCommand(strings.Join(args, ""))
	//c.CheckOkay()
	//c.CheckOkay()
	//return c
	return adbDevice.openTransport(strings.Join(args, ""), adbDevice.Client.SocketTime)
}

func (adbDevice AdbDevice) Reverse(local, remote string, noRebind bool) *AdbConnection {
	args := []string{"reverse:forward:"}
	if noRebind {
		args = append(args, "norebind:")
	}
	args = append(args, []string{local, ";", remote}...)

	c := adbDevice.openTransport("", adbDevice.Client.SocketTime)
	c.SendCommand(strings.Join(args, ""))
	c.CheckOkay()
	c.CheckOkay()
	return c
	//adbDevice.openTransport("", adbDevice.Client.SocketTime)
	//return adbDevice.openTransport(strings.Join(args, ""), adbDevice.Client.SocketTime)
}

func (adbDevice AdbDevice) ForWardPort(remote interface{}) int {
	tmpRemote := ""
	switch remote.(type) {
	case int:
		tmpRemote = "tcp:" + remote.(string)
	default:
		for _, f := range adbDevice.ForwardList() {
			if f.Serial == adbDevice.Serial && f.Remote == tmpRemote && strings.HasPrefix(f.Local, "tcp") {
				port, err := strconv.Atoi(f.Local[:2])
				if err != nil {
					return 0
				}
				return port
			}
		}
	}
	localPort := GetFreePort()
	adbDevice.ForWard(fmt.Sprintf("tcp:%d", localPort), tmpRemote, false)
	return localPort
}

func (adbDevice AdbDevice) ForwardList() []ForwardItem {
	c := adbDevice.openTransport("list-forward", adbDevice.Client.SocketTime)
	content := c.ReadStringBlock()
	//fmt.Println(content)
	var forwardItems []ForwardItem
	for _, line := range strings.Split(content, "\n") {
		parts := strings.Split(strings.TrimSpace(line), " ")
		//fmt.Println(parts)
		if len(parts) != 3 {
			continue
		} else {
			forwardItems = append(forwardItems, ForwardItem{
				Serial: string(parts[0]),
				Local:  string(parts[1]),
				Remote: string(parts[2]),
			})
		}
	}
	return forwardItems
}

func (adbDevice AdbDevice) Push(local, remote string) string {
	return adbDevice.AdbOut(fmt.Sprintf("push %v %v", local, remote))
}

func (adbDevice AdbDevice) CreateConnection(netWork, address string) net.Conn {
	c := adbDevice.openTransport("", 0)
	c.SendCommand("host:transport:" + adbDevice.Serial)
	c.CheckOkay()
	switch netWork {
	case TCP:
		c.SendCommand("tcp:" + address)
		c.CheckOkay()
	case UNIX, LOCALABSTRACT:
		c.SendCommand("localabstract:" + address)
		c.CheckOkay()
	case LOCALFILESYSTEM, LOCAL, DEV, LOCALRESERVED:
		c.SendCommand(netWork + ":" + address)
		c.CheckOkay()
	default:
		panic("not support net work: " + netWork)
	}
	return c.Conn
}

// Sync region ync
type Sync struct {
	*AdbClient
	Serial string
}

func (sync Sync) prepareSync(path, cmd string) (*AdbConnection, error) {
	c := sync.AdbClient.Device(SerialNTransportID{Serial: sync.Serial}).openTransport("", 10)
	c.SendCommand("sync:")
	c.CheckOkay()
	//pathLength := len([]byte(path))
	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, uint32(5)) // python struct.pack("<I",  小端序 int 4byte
	cmdByte := []byte(cmd)
	msg := append(cmdByte, bs...)
	msg = append(msg, []byte(path)...)
	_, err := c.Conn.Write(msg)
	if err != nil {
		log.Println("prepareSync write error: ", err.Error())
		return nil, err
	}
	return c, nil
}

func (sync Sync) Exist(path string) bool {
	return sync.Stat(path).Mtime != nil
}

func (sync Sync) Stat(path string) FileInfo {
	c, err := sync.prepareSync(path, "STAT")
	if c.ReadString(4) != "STAT" || err != nil {
		log.Println("Stat sync error!")
	}
	fileInfo := FileInfo{Path: path}
	res := []uint32{}
	for i := 0; i < 3; i++ {
		res = append(res, binary.LittleEndian.Uint32(c.Read(4)))
	}
	fileInfo.Mode = int(res[0])
	fileInfo.Size = int(res[1])
	if res[2] != 0 {
		mtime := time.Unix(int64(res[2]), 0)
		fileInfo.Mtime = &mtime
	}
	return fileInfo
}

func (sync Sync) IterDirectory(path string) []FileInfo {
	c, err := sync.prepareSync(path, "LIST")
	if err != nil {
		log.Println("get file list error ", err.Error())
	}
	var fileInfos []FileInfo
	for {
		response := c.ReadString(4)
		if response == DONE {
			break
		}
		fileInfo := FileInfo{}
		var res []uint32
		for i := 0; i < 4; i++ {
			res = append(res, binary.LittleEndian.Uint32(c.Read(4)))
		}
		name := c.ReadString(int(res[3]))
		fileInfo.Mode = int(res[0])
		fileInfo.Size = int(res[1])
		fileInfo.Path = name
		if res[2] != 0 {
			mtime := time.Unix(int64(res[2]), 0)
			fileInfo.Mtime = &mtime
		}
		fileInfos = append(fileInfos, fileInfo)
	}
	return fileInfos
}

func (sync Sync) List(path string) []FileInfo {
	return sync.IterDirectory(path)
}

func (sync Sync) Push(src, dst string, mode int, check bool) int {
	//path := dst + "," + ""
	path := dst + "," + strconv.Itoa(syscall.S_IFREG|mode)
	c, err := sync.prepareSync(path, "SEND")
	if err != nil {
		log.Println("Sync Push err ! ", err.Error())
	}
	file, err := os.OpenFile(src, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Println("when Push, read local file error! ", err.Error())
	}
	totalSize := 0
	for {
		chunk := make([]byte, 0)
		_, err = file.Read(chunk)
		if err != nil {
			log.Println("when Push, read local file error! ", err.Error())
		}
		if len(chunk) == 0 {
			msg := []byte("DONE")
			bs := make([]byte, 4)
			binary.LittleEndian.PutUint32(bs, uint32(time.Now().Unix()))
			msg = append(msg, bs...)
			_, err = c.Conn.Write(msg)
			if err != nil {
				log.Println("Sync Push send done error! ", err.Error())
			}
			break
		}
		msg := []byte("DONE")
		bs := make([]byte, 4)
		binary.LittleEndian.PutUint32(bs, uint32(len(chunk)))
		msg = append(msg, bs...)
		_, err = c.Conn.Write(msg)
		if err != nil {
			log.Println("when push write content error! ", err.Error())
		}
		_, err = c.Conn.Write(chunk)
		if err != nil {
			log.Println("when push write content error! ", err.Error())
		}
	}
	if check {
		fileSize := sync.Stat(dst).Size
		if fileSize != totalSize {
			log.Println(fmt.Sprintf("Push not complete, expect pushed %d, actually pushed %d", totalSize, fileSize))
		}
	}
	return totalSize
}

func (sync Sync) IterContent(path string) []byte {
	c, err := sync.prepareSync(path, "RECV")
	if err != nil {
		log.Println("IterContent error ", err.Error())
	}
	var chunks []byte
	for {
		cmd := c.ReadString(4)
		switch cmd {
		case FAIL:
			strSize := binary.LittleEndian.Uint32(c.Read(4))
			errMsg := c.ReadString(int(strSize))
			log.Println(fmt.Sprintf("Get %s Error %s", errMsg, path))
		case DATA:
			chunkSize := binary.LittleEndian.Uint32(c.Read(4))
			chunk := c.Read(int(chunkSize))
			if len(chunk) != int(chunkSize) {
				log.Println("read chunk missing")
			}
			chunks = append(chunks, chunk...)
		case DONE:
			break
		default:
			log.Println("Invalid sync cmd: ", cmd)
		}
	}
	return chunks
}

func (sync Sync) ReadBytes(path string) []byte {
	return sync.IterContent(path)
}

func (sync Sync) ReadText(path string) string {
	return string(sync.ReadBytes(path))
}

func (sync Sync) Pull(src, dst string) int {
	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Println("Sync pull file error! ", err.Error())
	}
	bytes := sync.IterContent(src)
	size, err := f.Write(bytes)
	if err != nil {
		log.Println("Sync pull file error, when write! ", err.Error())
		return 0
	}
	return size
}
