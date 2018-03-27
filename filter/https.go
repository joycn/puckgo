package filter

import (
	"bufio"
	"fmt"
)

// TLS record types.
type recordType uint8

const (
	recordHeaderLen                = 5
	recordTypeHandshake recordType = 22
	typeClientHello     uint8      = 1
	extensionServerName uint16     = 0
)

func ServerName(data []byte) (string, bool) {
	if len(data) < 42 {
		return "", false
	}
	sessionIdLen := int(data[38])
	if sessionIdLen > 32 || len(data) < 39+sessionIdLen {
		return "", false
	}
	//m.sessionId = data[39 : 39+sessionIdLen]
	data = data[39+sessionIdLen:]
	if len(data) < 2 {
		return "", false
	}
	// cipherSuiteLen is the number of bytes of cipher suite numbers. Since
	// they are uint16s, the number must be even.
	cipherSuiteLen := int(data[0])<<8 | int(data[1])
	if cipherSuiteLen%2 == 1 || len(data) < 2+cipherSuiteLen {
		return "", false
	}
	//numCipherSuites := cipherSuiteLen / 2
	//m.cipherSuites = make([]uint16, numCipherSuites)
	//for i := 0; i < numCipherSuites; i++ {
	//m.cipherSuites[i] = uint16(data[2+2*i])<<8 | uint16(data[3+2*i])
	//if m.cipherSuites[i] == scsvRenegotiation {
	//m.secureRenegotiationSupported = true
	//}
	//}
	data = data[2+cipherSuiteLen:]
	if len(data) < 1 {
		return "", false
	}
	compressionMethodsLen := int(data[0])
	if len(data) < 1+compressionMethodsLen {
		return "", false
	}
	//m.compressionMethods = data[1 : 1+compressionMethodsLen]

	data = data[1+compressionMethodsLen:]

	//m.nextProtoNeg = false
	//m.ocspStapling = false
	//m.ticketSupported = false
	//m.sessionTicket = nil
	//m.signatureAndHashes = nil
	//m.alpnProtocols = nil
	//m.scts = false

	if len(data) == 0 {
		// ClientHello is optionally followed by extension data
		return "", false
	}
	if len(data) < 2 {
		return "", false
	}

	extensionsLength := int(data[0])<<8 | int(data[1])
	data = data[2:]
	if extensionsLength != len(data) {
		return "", false
	}

	for len(data) != 0 {
		if len(data) < 4 {
			return "", false
		}
		extension := uint16(data[0])<<8 | uint16(data[1])
		length := int(data[2])<<8 | int(data[3])
		data = data[4:]
		if len(data) < length {
			return "", false
		}

		switch extension {
		case extensionServerName:
			d := data[:length]
			if len(d) < 2 {
				return "", false
			}
			namesLen := int(d[0])<<8 | int(d[1])
			d = d[2:]
			if len(d) != namesLen {
				return "", false
			}
			for len(d) > 0 {
				if len(d) < 3 {
					return "", false
				}
				nameType := d[0]
				nameLen := int(d[1])<<8 | int(d[2])
				d = d[3:]
				if len(d) < nameLen {
					return "", false
				}
				if nameType == 0 {
					return string(d[:nameLen]), true
				}
				d = d[nameLen:]
			}
			//case extensionNextProtoNeg:
			//if length > 0 {
			//return "", false
			//}
			////m.nextProtoNeg = true
			//case extensionStatusRequest:
			////m.ocspStapling = length > 0 && data[0] == statusTypeOCSP
			//case extensionSupportedCurves:
			//// http://tools.ietf.org/html/rfc4492#section-5.5.1
			//if length < 2 {
			//return "", false
			//}
			//l := int(data[0])<<8 | int(data[1])
			//if l%2 == 1 || length != l+2 {
			//return "", false
			//}
			//numCurves := l / 2
			////m.supportedCurves = make([]CurveID, numCurves)
			//d := data[2:]
			//for i := 0; i < numCurves; i++ {
			////m.supportedCurves[i] = CurveID(d[0])<<8 | CurveID(d[1])
			//d = d[2:]
			//}
			////case extensionSupportedPoints:
			//// http://tools.ietf.org/html/rfc4492#section-5.5.2
			////if length < 1 {
			////return "", false
			////}
			////l := int(data[0])
			////if length != l+1 {
			////return "", false
			////}
			////m.supportedPoints = make([]uint8, l)
			////copy(m.supportedPoints, data[1:])
			////case extensionSessionTicket:
			//// http://tools.ietf.org/html/rfc5077#section-3.2
			////m.ticketSupported = true
			////m.sessionTicket = data[:length]
			//case extensionSignatureAlgorithms:
			//// https://tools.ietf.org/html/rfc5246#section-7.4.1.4.1
			//if length < 2 || length&1 != 0 {
			//return "", false
			//}
			//l := int(data[0])<<8 | int(data[1])
			//if l != length-2 {
			//return "", false
			//}
			//n := l / 2
			//d := data[2:]
			//m.signatureAndHashes = make([]signatureAndHash, n)
			//for i := range m.signatureAndHashes {
			//m.signatureAndHashes[i].hash = d[0]
			//m.signatureAndHashes[i].signature = d[1]
			//d = d[2:]
			//}
			//case extensionRenegotiationInfo:
			//if length == 0 {
			//return "", false
			//}
			//d := data[:length]
			//l := int(d[0])
			//d = d[1:]
			//if l != len(d) {
			//return "", false
			//}

			//m.secureRenegotiation = d
			//m.secureRenegotiationSupported = true
			//case extensionALPN:
			//if length < 2 {
			//return "", false
			//}
			//l := int(data[0])<<8 | int(data[1])
			//if l != length-2 {
			//return "", false
			//}
			//d := data[2:length]
			//for len(d) != 0 {
			//stringLen := int(d[0])
			//d = d[1:]
			//if stringLen == 0 || stringLen > len(d) {
			//return "", false
			//}
			//m.alpnProtocols = append(m.alpnProtocols, string(d[:stringLen]))
			//d = d[stringLen:]
			//}
			//case extensionSCT:
			//m.scts = true
			//if length != 0 {
			//return "", false
			//}
		}
		data = data[length:]
	}

	return "", false
}

func filterByTLSServerName(r *bufio.Reader) (string, FilterAction, Buffer, error) {

	header, err := r.Peek(recordHeaderLen)

	if err != nil {
		return "", Again, nil, err
	}
	typ := recordType(header[0])
	switch typ {

	case recordTypeHandshake:
		major := header[1]
		minor := header[2]

		if major != 3 || minor > 3 {
			return "", Continue, nil, fmt.Errorf("inval format")
		}

		msgLen := int(header[3])<<8 | int(header[4])

		if msgLen < 4 {
			return "", Continue, nil, fmt.Errorf("msg length invalid")
		}

		data, err := r.Peek(recordHeaderLen + msgLen)
		if err != nil {
			return "", Again, nil, err
		}

		data = data[recordHeaderLen:]

		handshakeLen := int(data[1])<<16 | int(data[2])<<8 | int(data[3])

		if len(data) < handshakeLen+4 {
			return "", Stop, nil, fmt.Errorf("handshakeLen invalid")
		}

		switch data[0] {
		case typeClientHello:
			if serverName, found := ServerName(data); found {
				return serverName, Stop, nil, nil
			}
		}

		return "", Stop, nil, fmt.Errorf("SNI not found")
	}
	return "", Continue, nil, fmt.Errorf("not https")
}

func NewHTTPSFilter() *Filter {
	return &Filter{Name: "tls", Func: filterByTLSServerName}
}
