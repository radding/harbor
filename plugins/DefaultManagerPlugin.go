package plugins

type DefaulManagerServer struct {
}

func (d *DefaulManagerServer) CanHandle(req CanHandleRequest) (bool, error) {
	return false, nil
}

func (d *DefaulManagerServer) Clone(req CloneRequest) (string, error) {
	return "", nil
}
