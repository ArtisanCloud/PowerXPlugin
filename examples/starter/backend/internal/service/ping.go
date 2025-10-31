package service

// PingService 提供最小示例业务逻辑。
type PingService struct{}

// NewPingService 构造 PingService。
func NewPingService() *PingService {
	return &PingService{}
}

// Ping 返回示例响应。
func (s *PingService) Ping() map[string]string {
	return map[string]string{"status": "ok"}
}
