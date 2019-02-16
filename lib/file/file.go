package file

import (
	"encoding/csv"
	"errors"
	"github.com/cnlh/nps/lib/common"
	"github.com/cnlh/nps/lib/lg"
	"github.com/cnlh/nps/lib/rate"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

func NewCsv(runPath string) *Csv {
	return &Csv{
		RunPath: runPath,
	}
}

type Csv struct {
	Tasks            []*Tunnel
	Path             string
	Hosts            []*Host   //域名列表
	Clients          []*Client //客户端
	RunPath          string    //存储根目录
	ClientIncreaseId int       //客户端id
	TaskIncreaseId   int       //任务自增ID
	HostIncreaseId   int
	sync.Mutex
}

func (s *Csv) Init() {
	s.LoadClientFromCsv()
	s.LoadTaskFromCsv()
	s.LoadHostFromCsv()
}

func (s *Csv) StoreTasksToCsv() {
	// 创建文件
	csvFile, err := os.Create(filepath.Join(s.RunPath, "conf", "tasks.csv"))
	if err != nil {
		lg.Fatalf(err.Error())
	}
	defer csvFile.Close()
	writer := csv.NewWriter(csvFile)
	for _, task := range s.Tasks {
		if task.NoStore {
			continue
		}
		record := []string{
			strconv.Itoa(task.Port),
			task.Mode,
			task.Target,
			common.GetStrByBool(task.Status),
			strconv.Itoa(task.Id),
			strconv.Itoa(task.Client.Id),
			task.Remark,
		}
		err := writer.Write(record)
		if err != nil {
			lg.Fatalf(err.Error())
		}
	}
	writer.Flush()
}

func (s *Csv) openFile(path string) ([][]string, error) {
	// 打开文件
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// 获取csv的reader
	reader := csv.NewReader(file)

	// 设置FieldsPerRecord为-1
	reader.FieldsPerRecord = -1

	// 读取文件中所有行保存到slice中
	return reader.ReadAll()
}

func (s *Csv) LoadTaskFromCsv() {
	path := filepath.Join(s.RunPath, "conf", "tasks.csv")
	records, err := s.openFile(path)
	if err != nil {
		lg.Fatalln("配置文件打开错误:", path)
	}
	var tasks []*Tunnel
	// 将每一行数据保存到内存slice中
	for _, item := range records {
		post := &Tunnel{
			Port:   common.GetIntNoErrByStr(item[0]),
			Mode:   item[1],
			Target: item[2],
			Status: common.GetBoolByStr(item[3]),
			Id:     common.GetIntNoErrByStr(item[4]),
			Remark: item[6],
		}
		post.Flow = new(Flow)
		if post.Client, err = s.GetClient(common.GetIntNoErrByStr(item[5])); err != nil {
			continue
		}
		tasks = append(tasks, post)
		if post.Id > s.TaskIncreaseId {
			s.TaskIncreaseId = post.Id
		}
	}
	s.Tasks = tasks
}

func (s *Csv) GetTaskId() int {
	s.Lock()
	defer s.Unlock()
	s.TaskIncreaseId++
	return s.TaskIncreaseId
}
func (s *Csv) GetHostId() int {
	s.Lock()
	defer s.Unlock()
	s.HostIncreaseId++
	return s.HostIncreaseId
}

func (s *Csv) GetIdByVerifyKey(vKey string, addr string) (int, error) {
	s.Lock()
	defer s.Unlock()
	for _, v := range s.Clients {
		if common.Getverifyval(v.VerifyKey) == vKey && v.Status {
			if arr := strings.Split(addr, ":"); len(arr) > 0 {
				v.Addr = arr[0]
			}
			return v.Id, nil
		}
	}
	return 0, errors.New("not found")
}

func (s *Csv) NewTask(t *Tunnel) {
	t.Flow = new(Flow)
	s.Tasks = append(s.Tasks, t)
	s.StoreTasksToCsv()
}

func (s *Csv) UpdateTask(t *Tunnel) error {
	for k, v := range s.Tasks {
		if v.Id == t.Id {
			s.Tasks = append(s.Tasks[:k], s.Tasks[k+1:]...)
			s.Tasks = append(s.Tasks, t)
			s.StoreTasksToCsv()
			return nil
		}
	}
	return errors.New("不存在")
}

func (s *Csv) DelTask(id int) error {
	for k, v := range s.Tasks {
		if v.Id == id {
			s.Tasks = append(s.Tasks[:k], s.Tasks[k+1:]...)
			s.StoreTasksToCsv()
			return nil
		}
	}
	return errors.New("不存在")
}

func (s *Csv) GetTask(id int) (v *Tunnel, err error) {
	for _, v = range s.Tasks {
		if v.Id == id {
			return
		}
	}
	err = errors.New("未找到")
	return
}

func (s *Csv) StoreHostToCsv() {
	// 创建文件
	csvFile, err := os.Create(filepath.Join(s.RunPath, "conf", "hosts.csv"))
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()
	// 获取csv的Writer
	writer := csv.NewWriter(csvFile)
	// 将map中的Post转换成slice，因为csv的Write需要slice参数
	// 并写入csv文件
	for _, host := range s.Hosts {
		if host.NoStore {
			continue
		}
		record := []string{
			host.Host,
			host.Target,
			strconv.Itoa(host.Client.Id),
			host.HeaderChange,
			host.HostChange,
			host.Remark,
			host.Location,
			strconv.Itoa(host.Id),
		}
		err1 := writer.Write(record)
		if err1 != nil {
			panic(err1)
		}
	}
	// 确保所有内存数据刷到csv文件
	writer.Flush()
}

func (s *Csv) LoadClientFromCsv() {
	path := filepath.Join(s.RunPath, "conf", "clients.csv")
	records, err := s.openFile(path)
	if err != nil {
		lg.Fatalln("配置文件打开错误:", path)
	}
	var clients []*Client
	// 将每一行数据保存到内存slice中
	for _, item := range records {
		post := &Client{
			Id:        common.GetIntNoErrByStr(item[0]),
			VerifyKey: item[1],
			Remark:    item[2],
			Status:    common.GetBoolByStr(item[3]),
			RateLimit: common.GetIntNoErrByStr(item[8]),
			Cnf: &Config{
				U:        item[4],
				P:        item[5],
				Crypt:    common.GetBoolByStr(item[6]),
				Compress: item[7],
			},
		}
		if post.Id > s.ClientIncreaseId {
			s.ClientIncreaseId = post.Id
		}
		if post.RateLimit > 0 {
			post.Rate = rate.NewRate(int64(post.RateLimit * 1024))
			post.Rate.Start()
		}
		post.Flow = new(Flow)
		post.Flow.FlowLimit = int64(common.GetIntNoErrByStr(item[9]))
		clients = append(clients, post)
	}
	s.Clients = clients
}

func (s *Csv) LoadHostFromCsv() {
	path := filepath.Join(s.RunPath, "conf", "hosts.csv")
	records, err := s.openFile(path)
	if err != nil {
		lg.Fatalln("配置文件打开错误:", path)
	}
	var hosts []*Host
	// 将每一行数据保存到内存slice中
	for _, item := range records {
		post := &Host{
			Host:         item[0],
			Target:       item[1],
			HeaderChange: item[3],
			HostChange:   item[4],
			Remark:       item[5],
			Location:     item[6],
			Id:           common.GetIntNoErrByStr(item[7]),
		}
		if post.Client, err = s.GetClient(common.GetIntNoErrByStr(item[2])); err != nil {
			continue
		}
		post.Flow = new(Flow)
		hosts = append(hosts, post)
		if post.Id > s.HostIncreaseId {
			s.HostIncreaseId = post.Id
		}
	}
	s.Hosts = hosts
}

func (s *Csv) DelHost(id int) error {
	for k, v := range s.Hosts {
		if v.Id == id {
			s.Hosts = append(s.Hosts[:k], s.Hosts[k+1:]...)
			s.StoreHostToCsv()
			return nil
		}
	}
	return errors.New("不存在")
}

func (s *Csv) IsHostExist(h *Host) bool {
	for _, v := range s.Hosts {
		if v.Host == h.Host && h.Location == v.Location {
			return true
		}
	}
	return false
}

func (s *Csv) NewHost(t *Host) {
	t.Flow = new(Flow)
	s.Hosts = append(s.Hosts, t)
	s.StoreHostToCsv()
}

func (s *Csv) UpdateHost(t *Host) error {
	for k, v := range s.Hosts {
		if v.Host == t.Host {
			s.Hosts = append(s.Hosts[:k], s.Hosts[k+1:]...)
			s.Hosts = append(s.Hosts, t)
			s.StoreHostToCsv()
			return nil
		}
	}
	return errors.New("不存在")
}

func (s *Csv) GetHost(start, length int, id int) ([]*Host, int) {
	list := make([]*Host, 0)
	var cnt int
	for _, v := range s.Hosts {
		if id == 0 || v.Client.Id == id {
			cnt++
			if start--; start < 0 {
				if length--; length > 0 {
					list = append(list, v)
				}
			}
		}
	}
	return list, cnt
}

func (s *Csv) DelClient(id int) error {
	for k, v := range s.Clients {
		if v.Id == id {
			s.Clients = append(s.Clients[:k], s.Clients[k+1:]...)
			s.StoreClientsToCsv()
			return nil
		}
	}
	return errors.New("不存在")
}

func (s *Csv) NewClient(c *Client) {
	if c.Id == 0 {
		c.Id = s.GetClientId()
	}
	c.Flow = new(Flow)
	s.Lock()
	defer s.Unlock()
	s.Clients = append(s.Clients, c)
	s.StoreClientsToCsv()
}

func (s *Csv) GetClientId() int {
	s.Lock()
	defer s.Unlock()
	s.ClientIncreaseId++
	return s.ClientIncreaseId
}

func (s *Csv) UpdateClient(t *Client) error {
	s.Lock()
	defer s.Unlock()
	for _, v := range s.Clients {
		if v.Id == t.Id {
			v.Cnf = t.Cnf
			v.VerifyKey = t.VerifyKey
			v.Remark = t.Remark
			v.RateLimit = t.RateLimit
			v.Flow = t.Flow
			v.Rate = t.Rate
			s.StoreClientsToCsv()
			return nil
		}
	}
	return errors.New("该客户端不存在")
}

func (s *Csv) GetClientList(start, length int) ([]*Client, int) {
	list := make([]*Client, 0)
	var cnt int
	for _, v := range s.Clients {
		if v.NoDisplay {
			continue
		}
		cnt++
		if start--; start < 0 {
			if length--; length > 0 {
				list = append(list, v)
			}
		}
	}
	return list, cnt
}

func (s *Csv) GetClient(id int) (v *Client, err error) {
	for _, v = range s.Clients {
		if v.Id == id {
			return
		}
	}
	err = errors.New("未找到客户端")
	return
}
func (s *Csv) GetClientIdByVkey(vkey string) (id int, err error) {
	for _, v := range s.Clients {
		if v.VerifyKey == vkey {
			id = v.Id
			return
		}
	}
	err = errors.New("未找到客户端")
	return
}

func (s *Csv) GetHostById(id int) (h *Host, err error) {
	for _, v := range s.Hosts {
		if v.Id == id {
			h = v
			return
		}
	}
	err = errors.New("The host could not be parsed")
	return
}

//get key by host from x
func (s *Csv) GetInfoByHost(host string, r *http.Request) (h *Host, err error) {
	var hosts []*Host
	for _, v := range s.Hosts {
		//Remove http(s) http(s)://a.proxy.com
		//*.proxy.com *.a.proxy.com  Do some pan-parsing
		tmp := strings.Replace(v.Host, "*", `\w+?`, -1)
		var re *regexp.Regexp
		if re, err = regexp.Compile(tmp); err != nil {
			return
		}
		if len(re.FindAllString(host, -1)) > 0 {
			//URL routing
			hosts = append(hosts, v)
		}
	}
	for _, v := range hosts {
		//If not set, default matches all
		if v.Location == "" {
			v.Location = "/"
		}
		if strings.Index(r.RequestURI, v.Location) == 0 {
			if h == nil || (len(v.Location) > len(h.Location)) {
				h = v
			}
		}
	}
	if h != nil {
		return
	}
	err = errors.New("The host could not be parsed")
	return
}
func (s *Csv) StoreClientsToCsv() {
	// 创建文件
	csvFile, err := os.Create(filepath.Join(s.RunPath, "conf", "clients.csv"))
	if err != nil {
		lg.Fatalln(err.Error())
	}
	defer csvFile.Close()
	writer := csv.NewWriter(csvFile)
	for _, client := range s.Clients {
		if client.NoStore {
			continue
		}
		record := []string{
			strconv.Itoa(client.Id),
			client.VerifyKey,
			client.Remark,
			strconv.FormatBool(client.Status),
			client.Cnf.U,
			client.Cnf.P,
			common.GetStrByBool(client.Cnf.Crypt),
			client.Cnf.Compress,
			strconv.Itoa(client.RateLimit),
			strconv.Itoa(int(client.Flow.FlowLimit)),
		}
		err := writer.Write(record)
		if err != nil {
			lg.Fatalln(err.Error())
		}
	}
	writer.Flush()
}
