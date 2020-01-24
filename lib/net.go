package lib

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(Net, `

declare namespace net {
    export function ipAddress(): string

    export function macAddress(): string

    export type dialNetwork = "tcp" | "tcp4" | "tcp6" | "udp" | "udp4" | "udp6" | "ip" | "ip4" | "ip6" | "unix" | "unixgram" | "unixpacket"

    export type listenNetwork = "tcp" | "tcp4" | "tcp6" | "unix" | "unixpacket"

    export interface Connection {
        read(b: byte[]): number
        write(b: byte[]): number
        setDeadline(t: time.Time): void
        setWriteDeadline(t: time.Time): void
        setReadDeadline(t: time.Time): void
        close(): void
    }

    export interface Listener {
        accept(): Connection
        close(): void
    }

    export function dial(network: dialNetwork, address: string): Connection
    export function dialTimeout(network: dialNetwork, address: string, d: time.Duration | number): Connection
    export function listen(network: listenNetwork, address: string): Listener
}

`)
}

var Net = []core.NativeFunction{
	core.NativeFunction{
		Name:      "net.listen",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("netListen") {
				return core.NullValue, ErrUnauthorized
			}

			if err := ValidateArgs(args, core.String, core.String); err != nil {
				return core.NullValue, err
			}

			listener, err := newNetListener(args[0].ToString(), args[1].ToString(), vm)
			if err != nil {
				return core.NullValue, err
			}

			return core.NewObject(listener), nil
		},
	},
	core.NativeFunction{
		Name:      "net.dial",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String, core.String); err != nil {
				return core.NullValue, err
			}
			conn, err := net.Dial(args[0].ToString(), args[1].ToString())
			if err != nil {
				return core.NullValue, err
			}

			return core.NewObject(newNetConn(conn, vm)), nil
		},
	},
	core.NativeFunction{
		Name:      "net.dialTimeout",
		Arguments: 3,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected param 1 to be string, got %s", args[0].TypeName())
			}
			if args[1].Type != core.String {
				return core.NullValue, fmt.Errorf("expected param 2 to be string, got %s", args[1].TypeName())
			}

			d, err := ToDuration(args[2])
			if err != nil {
				return core.NullValue, err
			}

			conn, err := net.DialTimeout(args[0].ToString(), args[1].ToString(), d)
			if err != nil {
				return core.NullValue, err
			}

			return core.NewObject(newNetConn(conn, vm)), nil
		},
	},
	core.NativeFunction{
		Name:      "net.ipAddress",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			addrs, err := net.InterfaceAddrs()
			if err != nil {
				return core.NullValue, err
			}

			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						return core.NewString(ipnet.IP.String()), nil
					}
				}
			}

			return core.NullValue, fmt.Errorf("no IP address found")
		},
	},
	core.NativeFunction{
		Name:      "net.macAddress",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			addrs, err := net.InterfaceAddrs()
			if err != nil {
				return core.NullValue, err
			}

			var ip string

			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						ip = ipnet.IP.String()
						break
					}
				}
			}

			if ip == "" {
				return core.NullValue, fmt.Errorf("no IP address found")
			}

			interfaces, err := net.Interfaces()
			if err != nil {
				return core.NullValue, err
			}

			var hardwareName string

			for _, interf := range interfaces {
				if addrs, err := interf.Addrs(); err == nil {
					for _, addr := range addrs {
						// only interested in the name with current IP address
						if strings.Contains(addr.String(), ip) {
							hardwareName = interf.Name
							break
						}
					}
				}
			}

			if hardwareName == "" {
				return core.NullValue, fmt.Errorf("no network hardware found")
			}

			netInterface, err := net.InterfaceByName(hardwareName)
			if err != nil {
				return core.NullValue, err
			}

			macAddress := netInterface.HardwareAddr

			// verify if the MAC address can be parsed properly
			hwAddr, err := net.ParseMAC(macAddress.String())
			if err != nil {
				return core.NullValue, err
			}

			return core.NewString(hwAddr.String()), nil
		},
	},
}

func newNetConn(conn net.Conn, vm *core.VM) netConn {
	f := netConn{conn: conn}
	vm.SetGlobalFinalizer(f)
	return f
}

type netConn struct {
	conn net.Conn
}

func (netConn) Type() string {
	return "net.Connection"
}

func (c netConn) Close() error {
	return c.conn.Close()
}

func (c netConn) GetMethod(name string) core.NativeMethod {
	switch name {
	case "read":
		return c.read
	case "write":
		return c.write
	case "setDeadline":
		return c.setDeadline
	case "setWriteDeadline":
		return c.setWriteDeadline
	case "setReadDeadline":
		return c.setReadDeadline
	case "close":
		return c.close
	}
	return nil
}

func (c netConn) Write(b []byte) (n int, err error) {
	return c.conn.Write(b)
}

func (c netConn) Read(b []byte) (n int, err error) {
	return c.conn.Read(b)
}

func (c netConn) read(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Bytes); err != nil {
		return core.NullValue, err
	}
	b := args[0].ToBytes()
	n, err := c.conn.Read(b)
	if err != nil {
		return core.NullValue, err
	}
	return core.NewInt(n), nil
}

func (c netConn) write(args []core.Value, vm *core.VM) (core.Value, error) {
	var b []byte
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}
	a := args[0]
	switch a.Type {
	case core.Array, core.Bytes:
		b = a.ToBytes()
	case core.Int:
		b = []byte{byte(a.ToInt())}
	default:
		return core.NullValue, ErrInvalidType
	}

	n, err := c.conn.Write(b)
	if err != nil {
		return core.NullValue, err
	}
	return core.NewInt(n), nil
}

func (c netConn) setDeadline(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	t, ok := args[0].ToObjectOrNil().(TimeObj)
	if !ok {
		return core.NullValue, ErrInvalidType
	}

	c.conn.SetDeadline(time.Time(t))
	return core.NullValue, nil
}

func (c netConn) setWriteDeadline(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	t, ok := args[0].ToObjectOrNil().(TimeObj)
	if !ok {
		return core.NullValue, ErrInvalidType
	}

	c.conn.SetWriteDeadline(time.Time(t))
	return core.NullValue, nil
}

func (c netConn) setReadDeadline(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	t, ok := args[0].ToObjectOrNil().(TimeObj)
	if !ok {
		return core.NullValue, ErrInvalidType
	}

	c.conn.SetReadDeadline(time.Time(t))
	return core.NullValue, nil
}

func (c netConn) close(args []core.Value, vm *core.VM) (core.Value, error) {
	c.conn.Close()
	return core.NullValue, nil
}

func newNetListener(network, port string, vm *core.VM) (*netListener, error) {
	ls, err := net.Listen(network, port)
	if err != nil {
		return nil, err
	}
	listener := &netListener{ls: ls}
	vm.SetGlobalFinalizer(listener)
	return listener, nil
}

type netListener struct {
	ls net.Listener
}

func (netListener) Type() string {
	return "net.Listener"
}

func (c *netListener) Close() error {
	return c.ls.Close()
}

func (c *netListener) GetMethod(name string) core.NativeMethod {
	switch name {
	case "accept":
		return c.accept
	case "close":
		return c.close
	}
	return nil
}

func (c *netListener) accept(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	conn, err := c.ls.Accept()
	if err != nil {
		return core.NullValue, err
	}

	return core.NewObject(newNetConn(conn, vm)), nil
}

func (c *netListener) close(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	err := c.ls.Close()
	if err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}
