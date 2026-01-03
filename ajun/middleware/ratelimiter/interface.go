package ratelimiter

type Backend interface {
	Get(clientIP string) (*ClientIPData, error)
	Set(clientIP string, data *ClientIPData) error
	Delete(clientIP string) error
	List() (map[string]*ClientIPData, error)
}
