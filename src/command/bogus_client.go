package command

type bogusClient struct {
	outputChannel chan string
}

func (c bogusClient) ConfigPath() string {
	return ""
}
func (c bogusClient) Status() int {
	return CONF
}
func (c bogusClient) StatusConf()                                         {}
func (c bogusClient) StatusEnable()                                       {}
func (c bogusClient) StatusExit()                                         {}
func (c bogusClient) ConfigPathSet(path string)                           {}
func (c bogusClient) Newline()                                            {}
func (c bogusClient) Send(msg string)                                     {}
func (c bogusClient) SendNow(msg string)                                  {}
func (c bogusClient) Sendln(msg string)                                   {}
func (c bogusClient) SendlnNow(msg string)                                {}
func (c bogusClient) InputQuit()                                          {}
func (c bogusClient) HistoryAdd(cmd string)                               {}
func (c bogusClient) HistoryShow()                                        {}
func (c bogusClient) LineBufferComplete(autoComplete string, attach bool) {}
func (c bogusClient) Output() chan<- string {
	return c.outputChannel
}

func NewBogusClient() *bogusClient {
	c := &bogusClient{outputChannel: make(chan string)}
	close(c.outputChannel) // closed channel will break writers
	return c
}
