package lib

import (
	"bytes"
	"fmt"
)

type StringBuffer struct {
	buffer bytes.Buffer
}

func (b *StringBuffer) Write(format string, elem ...interface{}) {
	if len(elem) == 0 {
		b.buffer.WriteString(format)
		return
	}

	b.buffer.WriteString(fmt.Sprintf(format, elem...))
}

func (b *StringBuffer) String() string {
	return b.buffer.String()
}
