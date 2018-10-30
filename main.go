// datToExcel project main.go
package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mahonia"
	"os"
	//	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/larspensjo/config"
	"github.com/robfig/cron"
	"github.com/smartwalle/going/email"
)

func datToExcel(ExportFile, f *os.File, titleBars []string, ti string) string {
	defer ExportFile.Close()
	WriteFile := csv.NewWriter(ExportFile)
	WriteFile.Write(titleBars)
	defer f.Close()
	//打开文件出错处理
	buff := bufio.NewReader(f) //读入缓存
	for {
		line, err := buff.ReadString('\n') //以'\n'为结束符读入一行
		if err != nil || io.EOF == err {
			break
		}
		str := strings.Trim(line, "\r\n")
		conv := mahonia.NewEncoder("gbk")
		str = conv.ConvertString(str)
		if str == "--------------------" {
			break
		} else {
			slice := strings.Split(str, "^?")
			if len(slice) < 50 {
				slice = []string{slice[0], slice[1], slice[6], slice[8]}
			} else {
				slice = []string{slice[2], slice[3], slice[51]}
			}
			fmt.Println(slice)
			WriteFile.Write(slice)
		}
	}
	WriteFile.Flush()
	excelFileName := ExportFile.Name()
	return excelFileName
}

func prerr(err error) {
	if err != nil {
		panic(err)
	}
}

var (
	configFile = flag.String("configfile", "emailAlertOPS_config.ini", "General configuration file")
	Version    = "emailAlertOPS V1.1.20161027 "
	Auther     = "Frayn Fu"
)

func getArgs() {
	version := flag.Bool("v", false, "version")
	flag.Parse()
	if *version {
		fmt.Println("Version：", Version)
		fmt.Println("Auther:", Auther)
		return
	}
}

func main() {
	if len(os.Args) > 1 {
		getArgs()
	} else {
		datToExcelScheduleJob()
		//		sendingEmail()
		//		TOPIC := readconfigfile()
		//		excelFileName1 := generateCSV(TOPIC)
		//		fmt.Println(excelFileName1)
		//		ti := time.Now().Format("20060102")
		//		file1, _ := ListDir(TOPIC["pathres"], TOPIC["suffix"], ti)
		//		file2, _ := ListDir(TOPIC["pathgen"], TOPIC["suffix"], ti)
		//		fmt.Println(file1, file2)
	}
}

func datToExcelScheduleJob() {
	c := cron.New()
	spec := "0 1/1 * * * ?"
	c.AddFunc(spec, sendingEmail)
	c.Start()
	select {}
}
func generateCSV(TOPIC map[string]string) (excelFileName string) {
	//	tiy := time.Now().AddDate(0, 0, -1).Format("20060102")
	ti := time.Now().Format("20060102")
	tii := time.Now().Format("20060102150405")

	logFile, _ := os.OpenFile(ti+"/"+ti+".txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer logFile.Close()

	os.IsExist(os.Mkdir(ti, os.ModePerm))
	os.IsExist(os.Mkdir("backup", os.ModePerm))
	conv := mahonia.NewEncoder("gbk")
	//找当天的文件去和backup比对是否已处理，如果文件名是包含fksq,process fksq,else including fkhf, process fkhf
	filegen, _ := ListDir(TOPIC["pathgen"], TOPIC["suffix"], ti)
	fmt.Println(filegen)
	_, err := os.Stat("backup/" + filegen)
	if err == nil {
		fmt.Println(tii + ": " + filegen + " exist in backup!")
		logFile.WriteString(tii + ": " + filegen + " exist in backup!" + "\r\n")
		filegen = "filegen exist"
		fmt.Println(filegen)
	} else if strings.Contains(filegen, "bos_fksq_0011") {
		fmt.Println(filegen)
		logFile.WriteString(tii + ": " + filegen + " copied into backup!" + "\r\n")
		CopyFile("backup/"+filegen, TOPIC["pathgen"]+filegen)
	}
	fileres, _ := ListDir(TOPIC["pathres"], TOPIC["suffix"], ti)
	fmt.Println(fileres)
	_, err = os.Stat("backup/" + fileres)
	if err == nil {
		fmt.Println(tii + ": " + fileres + " exist in backup!")
		logFile.WriteString(tii + ": " + fileres + " exist in backup!" + "\r\n")
		fileres = "fileres exist"

	} else {
		logFile.WriteString(tii + ": " + fileres + " copied into backup!" + "\r\n")
		//		fmt.Println(fileres)
		CopyFile("backup/"+fileres, TOPIC["pathres"]+fileres)
	}
	if strings.Contains(filegen, "bos_fksq_0011") {
		fGen, _ := os.Open(TOPIC["pathgen"] + filegen) //打开文件
		ExportFileGen, _ := os.OpenFile(ti+"/"+"放款申请"+tii+".csv", os.O_WRONLY|os.O_CREATE, os.ModePerm)
		titleBarGen := []string{"合同号", "姓名", "放款金额"}
		for i, v := range titleBarGen {
			titleBarGen[i] = conv.ConvertString(v)
		}
		excelFileName = datToExcel(ExportFileGen, fGen, titleBarGen, ti)
	} else if strings.Contains(fileres, "bos_fkhf_0011") {
		fRes, _ := os.Open(TOPIC["pathres"] + fileres) //打开文件
		ExportFileRes, _ := os.OpenFile(ti+"/"+"放款回复"+tii+".csv", os.O_WRONLY|os.O_CREATE, os.ModePerm)
		titleBarRes := []string{"合同号", "姓名", "放款金额", "未通过原因"}
		for i, v := range titleBarRes {
			titleBarRes[i] = conv.ConvertString(v)
		}
		excelFileName = datToExcel(ExportFileRes, fRes, titleBarRes, ti)
	}
	return excelFileName
}
func sendingEmail() {
	TOPIC := readconfigfile()
	excelFileName := generateCSV(TOPIC)
	fmt.Println(excelFileName + "testing")
	if excelFileName != "" {
		SendEmail(TOPIC, excelFileName)
	}

}

func SendEmail(TOPIC map[string]string, excelFileName string) {
	ti := time.Now().Format("20060102")
	var config = &email.MailConfig{}
	config.Username = TOPIC["username"]
	config.Host = TOPIC["host"]
	config.Password = TOPIC["password"]
	config.Port = TOPIC["port"]
	config.Secure = false

	var e = email.NewTextMessage("上海联贷文件"+ti, "")
	//	var e = email.NewTextMessage("testing", "")
	e.From = TOPIC["from"]
	e.To = strings.Split(TOPIC["to"], ",")
	e.Cc = strings.Split(TOPIC["cc"], ",")
	//		e.Content = "上海联贷文件： " + strings.TrimLeft(excelFileName1, ti+"/") + ", " + strings.TrimLeft(excelFileName2, ti+"/") + ", " + strings.TrimLeft(excelFileName3, ti+"/")
	e.Content = "上海联贷文件： " + strings.TrimLeft(excelFileName, ti+"/")
	e.AttachFile(excelFileName)
	err := email.SendMail(config, e)
	prerr(err)
}

func readconfigfile() (TOPIC map[string]string) {
	TOPIC = make(map[string]string)
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()

	//set config file std
	cfg, err := config.ReadDefault(*configFile)
	if err != nil {
		log.Fatalf("Fail to find", *configFile, err)
	}
	//set config file std End

	//Initialized topic from the configuration
	if cfg.HasSection("topicArr") {
		section, err := cfg.SectionOptions("topicArr")
		if err == nil {
			for _, v := range section {
				options, err := cfg.String("topicArr", v)
				if err == nil {
					TOPIC[v] = options
				}
			}
		}
	}

	//Initialized topic from the configuration END
	return TOPIC
}

func ListDir(dirPth, suffix, ti string) (file string, err error) {
	//	files = make([]string, 0, 10)

	dir, _ := ioutil.ReadDir(dirPth)

	suffix = strings.ToUpper(suffix) //忽略后缀匹配的大小写

	for _, fi := range dir {
		if fi.IsDir() { // 忽略目录
			continue
		}
		if strings.HasSuffix(strings.ToUpper(fi.Name()), suffix) { //匹配文件
			if ti == fi.ModTime().Format("20060102") {
				fmt.Println(fi.ModTime().Format("20060102150405"))
				file = fi.Name()
			}
		}
	}
	return file, nil
}
func byteString(p []byte) string {
	for i := 0; i < len(p); i++ {
		if p[i] == 0 {
			return string(p[0:i])
		}
	}
	return string(p)
}

func CopyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}
