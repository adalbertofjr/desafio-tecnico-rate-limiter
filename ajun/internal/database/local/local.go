package local

import "time"

type DataSource struct {
	RemoteAddrs        map[string]int
	RemoteAddrsDisable map[string]time.Time
}

func InitDataSource() *DataSource {
	return &DataSource{
		RemoteAddrs:        make(map[string]int),
		RemoteAddrsDisable: make(map[string]time.Time),
	}
}

func (ds *DataSource) AddClientIP(clientIP string) int {
	ds.RemoteAddrs[clientIP]++
	return ds.RemoteAddrs[clientIP]
}

func (ds *DataSource) DisableClientIP(clientIP string, duration time.Duration) {
	ds.RemoteAddrsDisable[clientIP] = time.Now().Add(duration)
}

func (ds *DataSource) GetTimeDisabledClientIP(clientIP string) (time.Time, bool) {
	timeDisable, exists := ds.RemoteAddrsDisable[clientIP]
	return timeDisable, exists
}

func (ds *DataSource) GetClientIPCount(clientIP string) int {
	return ds.RemoteAddrs[clientIP]
}

func (ds *DataSource) ListClientIPs() map[string]int {
	return ds.RemoteAddrs
}

func (ds *DataSource) ResetClientIP(clientIP string) {
	delete(ds.RemoteAddrs, clientIP)
	delete(ds.RemoteAddrsDisable, clientIP)
}

func (ds *DataSource) ResetDataClientIPs() {
	ds.RemoteAddrs = make(map[string]int)
	ds.RemoteAddrsDisable = make(map[string]time.Time)
}
