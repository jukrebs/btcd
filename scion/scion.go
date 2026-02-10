package scion

import (
	"context"
	"crypto/tls"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/netsec-ethz/scion-apps/pkg/pan"
	"github.com/netsec-ethz/scion-apps/pkg/quicutil"
)

const (
	// Network is the net.Addr network name used internally by btcd to signal
	// that an address should be dialed/listened to using SCION/PAN.
	Network = "scion"
)

// IsAddress reports whether addr is a SCION/PAN UDP address string of the form
// "ISD-AS,[IP]:port" (IPv6 may be bracketed as in the SCION format).
func IsAddress(addr string) bool {
	_, err := pan.ParseUDPAddr(addr)
	return err == nil
}

// SplitHostPort is like net.SplitHostPort, but also accepts SCION addresses.
//
// net.SplitHostPort refuses to parse SCION addresses because they aren't valid
// RFC 3986 host:port pairs. PAN provides a compatible splitter for SCION.
func SplitHostPort(hostport string) (host, port string, err error) {
	if h, p, err := net.SplitHostPort(hostport); err == nil {
		return h, p, nil
	}
	return pan.SplitHostPort(hostport)
}

// JoinHostPort is like net.JoinHostPort, but avoids adding extra brackets for
// SCION hosts ("ISD-AS,[IP]").
func JoinHostPort(host, port string) string {
	if isHost(host) {
		return host + ":" + port
	}
	return net.JoinHostPort(host, port)
}

// Dial dials a SCION/PAN endpoint and returns a net.Conn compatible stream.
//
// The returned connection is QUIC-based (quicutil.SingleStream) and is
// encrypted but unauthenticated (InsecureSkipVerify), matching btcd's TCP
// trust model.
func Dial(ctx context.Context, address string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	remote, err := pan.ResolveUDPAddr(ctx, address)
	if err != nil {
		return nil, err
	}

	tlsCfg := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{quicutil.SingleStreamProto},
	}

	session, err := pan.DialQUIC(ctx, netip.AddrPort{}, remote, nil, nil, "", tlsCfg, nil)
	if err != nil {
		return nil, err
	}

	ss, err := quicutil.NewSingleStream(session)
	if err != nil {
		_ = session.CloseWithError(0x1, "singlestream init failed")
		return nil, err
	}
	return ss, nil
}

// Listen listens on a SCION/PAN endpoint and returns a net.Listener whose
// Accept returns net.Conn compatible streams.
func Listen(ctx context.Context, address string) (net.Listener, error) {
	local, err := pan.ResolveUDPAddr(ctx, address)
	if err != nil {
		return nil, err
	}

	tlsCfg := &tls.Config{
		Certificates: quicutil.MustGenerateSelfSignedCert(),
		NextProtos:   []string{quicutil.SingleStreamProto},
	}

	quicListener, err := pan.ListenQUIC(ctx, netip.AddrPortFrom(local.IP, local.Port), nil, tlsCfg, nil)
	if err != nil {
		return nil, err
	}

	return quicutil.SingleStreamListener{Listener: quicListener}, nil
}

// ExtractIPPort attempts to parse a SCION address and returns its IP and port.
// This is useful when btcd needs to keep using legacy address structures.
func ExtractIPPort(address string) (net.IP, uint16, error) {
	udp, err := pan.ParseUDPAddr(address)
	if err != nil {
		return nil, 0, err
	}

	// netip.Addr -> net.IP
	ip := net.IP(udp.IP.AsSlice())
	return ip, udp.Port, nil
}

func isHost(host string) bool {
	// host is expected to be the host portion without a port.
	if strings.Contains(host, ":") {
		// Fast path for SCION host candidates and IPv6; verify via PAN parser.
		_, err := pan.ParseUDPAddr(host + ":0")
		return err == nil
	}
	if strings.Contains(host, ",") {
		_, err := pan.ParseUDPAddr(host + ":0")
		return err == nil
	}
	return false
}
