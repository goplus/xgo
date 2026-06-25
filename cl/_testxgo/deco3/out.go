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
func (*Service) _xgodeco_get(req string) (resp string, err error) {
	return req, nil
}
func (_xgo_this *Service) get(req string) (resp string, err error) {
	retry(2, func() error {
		resp, err = _xgo_this._xgodeco_get(req)
		return err
	})
	return
}
