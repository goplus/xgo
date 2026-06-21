package main

type CompletionParams interface {
	MaxOutputTokens(int64) CompletionParams
	Temperature(float64) CompletionParams
}
type completionParamsImpl struct {
	maxTokens   int64
	temperature float64
}
type Client struct {
}

func (p *completionParamsImpl) MaxOutputTokens(n int64) CompletionParams {
	p.maxTokens = n
	return p
}
func (p *completionParamsImpl) Temperature(t float64) CompletionParams {
	p.temperature = t
	return p
}
func (c *Client) CompletionParams() CompletionParams {
	return &completionParamsImpl{}
}
func (c *Client) Complete(prompt string, params CompletionParams) {
}

var c Client

func main() {
	c.Complete("hello", c.CompletionParams().MaxOutputTokens(1024).Temperature(0.7))
}
