package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// TLS record / handshake 最小常量（仅 SNI 解析需要）
type recordType uint8

const (
	recordTypeHandshake recordType = 22
	typeClientHello     uint8      = 1
	extensionServerName uint16     = 0
	scsvRenegotiation   uint16     = 0x00ff
)

const maxClientHeaderSize = 16*1024 + 5

// isCompleteTLSRecord 判断是否已读满首个 TLS record。
func isCompleteTLSRecord(header []byte) bool {
	if len(header) < 5 {
		return false
	}
	length := binary.BigEndian.Uint16(header[3:5])
	return len(header) >= int(length)+5
}

// extractHTTPHost 从原始 HTTP 请求字节中解析 Host（不含端口）。
func extractHTTPHost(data []byte) (string, error) {
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(data)))
	if err != nil {
		return "", err
	}
	if req.Body != nil {
		_ = req.Body.Close()
	}
	return hostFromHTTPHost(req.Host), nil
}

// hostFromHTTPHost 从 Host 头去掉端口。
func hostFromHTTPHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	// host:port 或 [ipv6]:port
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

// getSNIServerName 从 TLS ClientHello 中提取 SNI。
func getSNIServerName(buf []byte) (string, error) {
	n := len(buf)
	if n < 5 {
		return "", fmt.Errorf("数据过短，不是 TLS 握手")
	}
	if recordType(buf[0]) != recordTypeHandshake {
		return "", fmt.Errorf("不是 TLS Handshake")
	}
	if buf[1] != 3 {
		return "", fmt.Errorf("不支持 TLS 版本 < 3")
	}
	if buf[5] != typeClientHello {
		return "", fmt.Errorf("不是 ClientHello")
	}

	msg := &clientHelloMsg{}
	if !msg.unmarshal(buf[5:n]) {
		return "", fmt.Errorf("解析 ClientHello 失败")
	}
	if msg.serverName == "" {
		return "", fmt.Errorf("ClientHello 中无 SNI")
	}
	return msg.serverName, nil
}

type clientHelloMsg struct {
	vers                         uint16
	random                       []byte
	sessionID                    []byte
	cipherSuites                 []uint16
	compressionMethods           []uint8
	serverName                   string
	secureRenegotiationSupported bool
}

// unmarshal 解析 ClientHello（不含 5 字节 TLS record 头）。
// 仅提取 SNI；其它扩展跳过。
func (m *clientHelloMsg) unmarshal(data []byte) bool {
	if len(data) < 42 {
		return false
	}
	// HandshakeType(1) + length(3) + version(2) + random(32) = 38, then sessionID
	m.vers = uint16(data[4])<<8 | uint16(data[5])
	m.random = data[6:38]
	sessionIDLen := int(data[38])
	if sessionIDLen > 32 || len(data) < 39+sessionIDLen {
		return false
	}
	m.sessionID = data[39 : 39+sessionIDLen]
	data = data[39+sessionIDLen:]
	if len(data) < 2 {
		return false
	}
	cipherSuiteLen := int(data[0])<<8 | int(data[1])
	if cipherSuiteLen%2 == 1 || len(data) < 2+cipherSuiteLen {
		return false
	}
	numCipherSuites := cipherSuiteLen / 2
	m.cipherSuites = make([]uint16, numCipherSuites)
	for i := 0; i < numCipherSuites; i++ {
		m.cipherSuites[i] = uint16(data[2+2*i])<<8 | uint16(data[3+2*i])
		if m.cipherSuites[i] == scsvRenegotiation {
			m.secureRenegotiationSupported = true
		}
	}
	data = data[2+cipherSuiteLen:]
	if len(data) < 1 {
		return false
	}
	compressionMethodsLen := int(data[0])
	if len(data) < 1+compressionMethodsLen {
		return false
	}
	m.compressionMethods = data[1 : 1+compressionMethodsLen]
	data = data[1+compressionMethodsLen:]

	m.serverName = ""
	if len(data) == 0 {
		return true
	}
	if len(data) < 2 {
		return false
	}
	extensionsLength := int(data[0])<<8 | int(data[1])
	data = data[2:]
	if extensionsLength != len(data) {
		return false
	}

	for len(data) != 0 {
		if len(data) < 4 {
			return false
		}
		extension := uint16(data[0])<<8 | uint16(data[1])
		length := int(data[2])<<8 | int(data[3])
		data = data[4:]
		if len(data) < length {
			return false
		}
		if extension == extensionServerName {
			d := data[:length]
			if len(d) < 2 {
				return false
			}
			namesLen := int(d[0])<<8 | int(d[1])
			d = d[2:]
			if len(d) != namesLen {
				return false
			}
			for len(d) > 0 {
				if len(d) < 3 {
					return false
				}
				nameType := d[0]
				nameLen := int(d[1])<<8 | int(d[2])
				d = d[3:]
				if len(d) < nameLen {
					return false
				}
				if nameType == 0 {
					m.serverName = string(d[:nameLen])
					if strings.HasSuffix(m.serverName, ".") {
						return false
					}
					// 找到 host_name 即可
					return true
				}
				d = d[nameLen:]
			}
		}
		data = data[length:]
	}
	return true
}
