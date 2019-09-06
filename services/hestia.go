package services

//HestiaService stores the variables that should be used when running the service
type HestiaService struct {
}

//InitHestiaService initializes the service for Hestia
func InitHestiaService() *HestiaService {

	rs := &HestiaService{}
	return rs
}
