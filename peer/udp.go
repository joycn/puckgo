package peer

import (
	"net"
)

// UDPParams Peer Params use udp protocl
type UDPParams struct {
	Client  bool   `mapstructure: "is_client"`
	Address string `mapstructure: "address"`
}

// UDP Peer Params use udp protocl
type UDP struct {
	token    []byte
	IsClient bool
	//Local    *net.UDPAddr
	LastSeen *net.UDPAddr
	Conn     *net.UDPConn
}

//CreateUDP return a created udp peer pointer based on params
func CreateUDP(params *UDPParams) (uc *UDP, err error) {
	udpAddr, err := net.ResolveUDPAddr("udp", params.Address)
	uc = new(UDP)
	if params.Client {
		conn, err = net.DialUDP("udp", udpAddr)
		uc.IsClient = true
	} else {
		conn, err := net.ListenUDP("udp", udpAddr)
	}
	if err != nil {
		return nil, err
	}
	uc.Conn = conn
	return uc, nil
}

//Write Implemetation for Peer Interface, send data to u.Remote
func (u *UDP) Write(p []byte) (n int, err error) {

}

//Read Implemetation for Peer Interface, receive data from u.Remote
func (u *UDP) Read(p []byte) (n int, err error) {

}
