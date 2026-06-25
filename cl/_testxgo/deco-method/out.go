package main

type Service struct {
	name string
}

func (s *Service) _xgodeco_call(req string) (resp string, err error) {
	resp = s.name + ":" + req
	return
}
func (s *Service) call(req string) (resp string, err error) {
	retry(2, func() error {
		resp, err = s._xgodeco_call(req)
		return err
	})
	return
}
func retry(times int, fn func() error) {
	for i := 0; i < times; i++ {
		if fn() == nil {
			return
		}
	}
}

var svc = &Service{name: "test"}

func main() {
	svc.call("ping")
}
