package haproxy

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var defaultHTTPClient = &http.Client{
	Timeout: 5 * time.Second,
}

type Stats struct {
	Pxname string `json:"pxname"`
	Svname string `json:"svname"`
	Stot   int    `json:"stot"`   // [LFBS]: cumulative number of sessions
	Type   string `json:"type"`   // [LFBS]: type of proxy
	Rate   int    `json:"rate"`   // [.FBS]: number of sessions per second over last elapsed second
	Bck    int    `json:"bck"`    // [..BS]: number of backup servers (backend), server is backup (server)
	Status string `json:"status"` // [LFBS]: status (UP/DOWN/NOLB/MAINT/MAINT(via)/MAINT(resolution)...)
	Scur   int    `json:"scur"`   // [LFBS]: current sessions
}

type haproxyClient struct {
	host       string
	port       string
	path       string
	httpClient *http.Client
}

type ConfigOption func(*haproxyClient)

func Host(host string) ConfigOption {
	return func(r *haproxyClient) {
		r.host = host
	}
}

func Port(port string) ConfigOption {
	return func(r *haproxyClient) {
		r.port = port
	}
}

func Path(path string) ConfigOption {
	return func(r *haproxyClient) {
		r.path = path
	}
}

func HTTPClient(client *http.Client) ConfigOption {
	return func(r *haproxyClient) {
		r.httpClient = client
	}
}

func Status(opts ...ConfigOption) ([]*Stats, error) {
	cfg := &haproxyClient{
		host:       "127.0.0.1",
		port:       "9999",
		path:       "/haproxy?stats",
		httpClient: defaultHTTPClient,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	data, err := cfg.fetch()
	if err != nil {
		return nil, err
	}

	// Parse the CSV data into the Stats struct
	status, err := cfg.parseCSV(data)
	if err != nil {
		return nil, err
	}

	return status, nil
}

// fetchStats fetches the raw statistics data from HAProxy.
/*
## Example haproxy configuration for stats endpoint:
frontend stats
	bind *:9999
	mode http
	stats enable
	stats uri /haproxy?stats
*/
func (c *haproxyClient) fetch() ([]byte, error) {
	url := fmt.Sprintf("http://%s:%s%s;csv", c.host, c.port, c.path)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func parseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func mapToStats(s map[string]string) *Stats {
	n := &Stats{
		Pxname: s["pxname"],
		Svname: s["svname"],
		Status: s["status"],
		Stot:   parseInt(s["stot"]),
		Rate:   parseInt(s["rate"]),
		Bck:    parseInt(s["bck"]),
		Scur:   parseInt(s["scur"]),
	}
	// (0=frontend, 1=backend, 2=server, 3=listener)
	switch s["type"] {
	case "0":
		n.Type = "FRONTEND"
	case "1":
		n.Type = "BACKEND"
	case "2":
		n.Type = "SERVER"
	case "3":
		n.Type = "LISTENER"
	default:
		n.Type = "UNKNOWN"
	}
	return n
}

/*
## Example CSV output:

# pxname,svname,qcur,qmax,scur,smax,slim,stot,bin,bout,dreq,dresp,ereq,econ,eresp,wretr,wredis,status,weight,act,bck,chkfail,chkdown,lastchg,downtime,qlimit,pid,iid,sid,throttle,lbtot,tracked,type,rate,rate_lim,rate_max,check_status,check_code,check_duration,hrsp_1xx,hrsp_2xx,hrsp_3xx,hrsp_4xx,hrsp_5xx,hrsp_other,hanafail,req_rate,req_rate_max,req_tot,cli_abrt,srv_abrt,comp_in,comp_out,comp_byp,comp_rsp,lastsess,last_chk,last_agt,qtime,ctime,rtime,ttime,agent_status,agent_code,agent_duration,check_desc,agent_desc,check_rise,check_fall,check_health,agent_rise,agent_fall,agent_health,addr,cookie,mode,algo,conn_rate,conn_rate_max,conn_tot,intercepted,dcon,dses,wrew,connect,reuse,cache_lookups,cache_hits,srv_icur,src_ilim,qtime_max,ctime_max,rtime_max,ttime_max,eint,idle_conn_cur,safe_conn_cur,used_conn_cur,need_conn_est,uweight,agg_server_status,agg_server_check_status,agg_check_status,-,ssl_sess,ssl_reused_sess,ssl_failed_handshake,h2_headers_rcvd,h2_data_rcvd,h2_settings_rcvd,h2_rst_stream_rcvd,h2_goaway_rcvd,h2_detected_conn_protocol_errors,h2_detected_strm_protocol_errors,h2_rst_stream_resp,h2_goaway_resp,h2_open_connections,h2_backend_open_streams,h2_total_connections,h2_backend_total_streams,
example-stats,FRONTEND,,,1,6,50000,1836,179923,8029010,5,0,4,,,,,OPEN,,,,,,,,,1,1,0,,,,0,1,0,2,,,,0,1826,0,9,0,0,,1,2,1836,,,0,0,0,0,,,,,,,,,,,,,,,,,,,,,http,,1,2,1836,1827,0,0,0,,,0,0,,,,,,,0,,,,,,,,,-,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,
example-198.51.100.74:80,FRONTEND,,,   0,4,50000,27,7729,96470,0,0,0,,,,,OPEN,,,,,,,,,1,2,0,,,,0,0,0,4,,,,0,14,0,13,0,0,,0,4,27,,,0,0,0,0,,,,,,,,,,,,,,,,,,,,,http,,0,4,27,0,0,0,0,,,0,0,,,,,,,0,,,,,,,,,-,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,
example-198.51.100.74:443,FRONTEND,,,0,3,50000,79,20238,35724,0,0,0,,,,,OPEN,,,,,,,,,1,3,0,,,,0,0,0,3,,,,0,78,0,2,0,0,,0,4,80,,,4908,2851,0,26,,,,,,,,,,,,,,,,,,,,,http,,0,3,83,0,0,0,0,,,0,0,,,,,,,0,,,,,,,,,-,79,0,4,74,0,222,0,0,0,0,0,0,0,0,74,74,
example-backend-default,198.51.100.196:443,0,0,0,1,,53,12278,59308,,0,,0,0,0,0,UP,1,1,0,0,0,60269,0,,1,4,1,,53,,2,0,,3,L6OK,,2,0,53,0,0,0,0,,,,53,0,0,,,,,1902,,,0,4,4153,4216,,,,Layer6 check passed,,2,3,4,,,,,,http,,,,,,,,0,53,0,,,0,,0,6,10003,10018,0,0,0,0,2,1,,,,-,3549,2528,0,,,,,,,,,,,,,,
example-backend-default,198.51.100.197:80,0,0,0,1,,54,15689,72886,,0,,0,0,0,0,UP,1,1,0,0,0,60269,0,,1,4,2,,54,,2,0,,3,L4OK,,0,0,39,0,15,0,0,,,,54,0,0,,,,,1901,,,0,1,5001,5048,,,,Layer4 check passed,,2,3,4,,,,,,http,,,,,,,,0,53,1,,,0,,0,2,10007,10017,0,0,0,0,1,1,,,,-,0,0,0,,,,,,,,,,,,,,
example-backend-default,198.51.100.151:8080,0,0,0,0,,0,0,0,,0,,0,0,0,0,no check,1,0,1,,,60269,,,1,4,3,,0,,2,0,,0,,,,0,0,0,0,0,0,,,,0,0,0,,,,,-1,,,0,0,0,0,,,,,,,,,,,,,,http,,,,,,,,0,0,0,,,0,,0,0,0,0,0,0,0,0,0,1,,,,-,0,0,0,,,,,,,,,,,,,,
example-backend-default,BACKEND,0,0,0,2,10000,107,27967,132194,0,0,,0,0,0,0,UP,2,2,1,,0,60269,0,,1,4,0,,107,,1,0,,6,,,,0,92,0,15,0,0,,,,107,0,0,4908,2851,0,26,1901,,,0,2,4581,4636,,,,,,,,,,,,,,http,,,,,,,,0,106,1,0,0,,,0,6,10007,10018,0,,,,,2,0,0,0,-,3549,2528,0,0,0,0,0,0,0,0,0,0,0,0,0,0,
example-backend-devdump,198.51.100.218:3000,0,0,0,0,,0,0,0,,0,,0,0,0,0,UP,1,1,0,0,0,60269,0,,1,5,1,,0,,2,0,,0,L4OK,,0,0,0,0,0,0,0,,,,0,0,0,,,,,-1,,,0,0,0,0,,,,Layer4 check passed,,2,3,4,,,,,,http,,,,,,,,0,0,0,,,0,,0,0,0,0,0,0,0,0,1,1,,,,-,0,0,0,,,,,,,,,,,,,,
example-backend-devdump,198.51.100.151:8080,0,0,0,0,,0,0,0,,0,,0,0,0,0,no check,1,0,1,,,60269,,,1,5,2,,0,,2,0,,0,,,,0,0,0,0,0,0,,,,0,0,0,,,,,-1,,,0,0,0,0,,,,,,,,,,,,,,http,,,,,,,,0,0,0,,,0,,0,0,0,0,0,0,0,0,0,1,,,,-,0,0,0,,,,,,,,,,,,,,
example-backend-devdump,BACKEND,0,0,0,0,10000,0,0,0,0,0,,0,0,0,0,UP,1,1,1,,0,60269,0,,1,5,0,,0,,1,0,,0,,,,0,0,0,0,0,0,,,,0,0,0,0,0,0,0,-1,,,0,0,0,0,,,,,,,,,,,,,,http,,,,,,,,0,0,0,0,0,,,0,0,0,0,0,,,,,1,0,0,0,-,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,
example-fixed-response-4,BACKEND,0,0,0,0,10000,0,0,0,0,0,,0,0,0,0,UP,0,0,0,,0,60269,,,1,6,0,,0,,1,0,,0,,,,0,0,0,0,0,0,,,,0,0,0,0,0,0,0,-1,,,0,0,0,0,,,,,,,,,,,,,,http,,,,,,,,0,0,0,0,0,,,0,0,0,0,0,,,,,0,0,0,0,-,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,
example-fixed-response-5,BACKEND,0,0,0,0,10000,0,0,0,0,0,,0,0,0,0,UP,0,0,0,,0,60269,,,1,7,0,,0,,1,0,,0,,,,0,0,0,0,0,0,,,,0,0,0,0,0,0,0,-1,,,0,0,0,0,,,,,,,,,,,,,,http,,,,,,,,0,0,0,0,0,,,0,0,0,0,0,,,,,0,0,0,0,-,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,
*/

func (c *haproxyClient) parseCSV(data []byte) ([]*Stats, error) {
	r := csv.NewReader(bytes.NewReader(data))
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no records found in CSV data")
	}
	if len(records) < 2 {
		status := []*Stats{}
		return status, nil
	}

	header := []string{}
	status := make([]*Stats, 0, len(records))

	for i, record := range records {
		if i == 0 && strings.HasPrefix(record[0], "#") {
			for k, col := range record {
				if k == 0 {
					col = strings.TrimPrefix(col, "#")
					col = strings.TrimSpace(col)
				}
				header = append(header, col)
			}
			continue
		}
		if len(header) == 0 {
			return nil, fmt.Errorf("header not found before data rows")
		}
		if len(header) != len(record) {
			return nil, fmt.Errorf("mismatched number of columns in record %d", i)
		}
		statsRow := map[string]string{}
		for j, col := range header {
			statsRow[col] = record[j]
		}
		status = append(status, mapToStats(statsRow))
	}

	return status, nil
}
