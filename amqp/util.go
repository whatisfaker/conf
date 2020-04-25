package amqp

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
)

//gzipCompress 压缩数据
func gzipCompress(in []byte) []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, _ = w.Write(in)
	w.Flush()
	w.Close()
	return b.Bytes()
}

//gzipUncompress 解压缩
func gzipUncompress(in []byte) ([]byte, error) {
	b := bytes.NewBuffer(in)
	r, err := gzip.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	out, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return out, nil
}
