package spike

func NewTestHttpServer() HttpServer {
	return &testHttpServer{}
}

type testHttpServer struct{}

func (t testHttpServer) Shutdown() error {
	return nil
}

func (t testHttpServer) ListenAndServe() error {
	return nil
}
