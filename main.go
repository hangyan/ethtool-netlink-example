// copy some code from: https://github.com/meilihao/demo/blob/8779a9541430420bc31337dee6b28983a8ef4439/netlink/client2.go
package main

import (
	"github.com/mdlayher/ethtool"
	"github.com/mdlayher/netlink"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"os"
)

var NETLINK_PORT uint32 = 100

const nlmsgAlignTo = 4

func socket() int {
	fd, err := unix.Socket(
		// Always used when opening netlink sockets.
		unix.AF_NETLINK,
		// Seemingly used interchangeably with SOCK_DGRAM,
		// but it appears not to matter which is used.
		unix.SOCK_RAW,
		// The netlink family that the socket will communicate
		// with, such as NETLINK_ROUTE or NETLINK_GENERIC.
		unix.NETLINK_GENERIC,
	)

	if err != nil {
		log.Fatal(err)
	}

	err = unix.Bind(fd, &unix.SockaddrNetlink{
		Family: unix.AF_NETLINK,
		Groups: 0,
		Pid:    0,
	})

	if err != nil {
		log.Fatal(err)
	}

	log.Println("create and build socket success")
	return fd

}

func buildMessage(content []byte) []byte {
	data := make([]byte, nlmsgAlign(len(content)+1))
	copy(data, content)
	msg := netlink.Message{
		Header: netlink.Header{
			Length:   16 + uint32(len(content)+1),
			Type:     0,
			Flags:    netlink.Request | netlink.Acknowledge,
			Sequence: 1,
			PID:      0,
		},
	}
	buf, err := msg.MarshalBinary()
	if err != nil {
		log.Fatal(err)
	}
	return buf
}

// from https://github.com/mdlayher/netlink/blob/master/align.go
func nlmsgAlign(n int) int {
	return (n + nlmsgAlignTo - 1) & ^(nlmsgAlignTo - 1) // (nlmsgAlignTo-1)取反再与(n + nlmsgAlignTo -1)即可减去多余的个数
}

func ethtoolOps() {
	c, err := ethtool.New()
	if err != nil {
		log.Fatal(err)
	}

	ifi := ethtool.Interface{
		Name: "ens192",
	}

	infos, err := c.LinkMode(ifi)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("%+v", infos)
}

func main() {
	fd := socket()
	defer func() {
		unix.Close(fd)
	}()

	ethtoolOps()

	log.Info("start send msg")
	content := "hello, musk"
	err := unix.Sendto(fd, buildMessage([]byte(content)), 0, &unix.SockaddrNetlink{
		// Always used when sending on netlink sockets.
		Family: unix.AF_NETLINK,
		Pid:    0, // to kernel
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Info("send msg done")

	log.Info("start receive msg")
	b := make([]byte, os.Getpagesize())
	for {
		// Peek at the buffer to see how many bytes are available.
		n, _, err := unix.Recvfrom(fd, b, unix.MSG_PEEK)
		if err != nil {
			log.Error(err)

			continue
		}
		// Break when we can read all messages.
		if n < len(b) {
			break
		}
		// Double in size if not enough bytes.
		b = make([]byte, len(b)*2)
	}
	// Read out all available messages.
	n, _, _ := unix.Recvfrom(fd, b, 0)
	log.Infof("get msg len: %d\n", n)

	log.Infof("data: %+v", b[:n])

	m := &netlink.Message{}
	if err = m.UnmarshalBinary(b[:n]); err != nil { // mdlayher/netlink要求int(m.Header.Length) == n, 与实际情况不符所以报错. 因此自行按照m.UnmarshalBinary源码重新实现
		log.Fatal(err)
	}

	log.Info("receive msg done")
}
