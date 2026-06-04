package freebuf

import "io"

var (
	_ Buffer         = (*CloseBufferWrapper)(nil)
	_ io.ReadCloser  = (*CloseBufferWrapper)(nil)
	_ io.WriteCloser = (*CloseBufferWrapper)(nil)
)

type CloseBufferWrapper struct {
	Buffer
}

func (c *CloseBufferWrapper) Close() error {
	c.Buffer.FreeMe()
	return nil
}
