package lib

import (
	"encoding/json"
	"fmt"

	"github.com/gtlang/gt/core"

	"github.com/gorilla/websocket"
)

func init() {
	core.RegisterLib(WebSocket, `

declare namespace websocket {
    export function upgrade(r: http.Request): WebsocketConnection

    export interface WebsocketConnection {
        guid: string
        write(v: any): number | void
        writeJSON(v: any): void
        writeText(text: string | byte[]): void
        readMessage(): WebSocketMessage
        close(): void
    }

    export interface WebSocketMessage {
        type: WebsocketType
        message: string
    }

    export enum WebsocketType {
        text = 1,
        binary = 2,
        close = 8,
        ping = 9,
        pong = 10
    }
}

`)
}

var upgrader = websocket.Upgrader{} // websockets: use default options

var WebSocket = []core.NativeFunction{
	core.NativeFunction{
		Name:      "websocket.upgrade",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("networking") {
				return core.NullValue, ErrUnauthorized
			}
			if err := ValidateArgs(args, core.Object); err != nil {
				return core.NullValue, err
			}
			r, ok := args[0].ToObject().(*request)
			if !ok {
				return core.NullValue, fmt.Errorf("invalid Request, got %s", args[1].TypeName())
			}

			c, err := upgrader.Upgrade(r.writer, r.request, nil)
			if err != nil {
				return core.NullValue, err
			}

			// Maximum message size allowed from peer.
			c.SetReadLimit(8192)

			return core.NewObject(newWebsocketConn(c, vm)), nil
		},
	},
}

func newWebsocketConn(con *websocket.Conn, vm *core.VM) *websocketConn {
	f := &websocketConn{con: con}
	vm.SetGlobalFinalizer(f)
	return f
}

type websocketConn struct {
	guid string
	con  *websocket.Conn
}

func (c *websocketConn) Type() string {
	return "http.WebsocketConnection"
}

func (c *websocketConn) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "guid":
		return core.NewString(c.guid), nil
	}
	return core.UndefinedValue, nil
}

func (c *websocketConn) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "guid":
		if v.Type != core.String {
			return ErrInvalidType
		}
		c.guid = v.ToString()
		return nil
	}
	return ErrReadOnlyOrUndefined
}

func (c *websocketConn) GetMethod(name string) core.NativeMethod {
	switch name {
	case "write":
		return c.write
	case "writeText":
		return c.writeTextMessage
	case "writeJSON":
		return c.writeJSON
	case "readMessage":
		return c.readMessage
	case "close":
		return c.close
	}
	return nil
}

func (c *websocketConn) Close() error {
	ws := c.con

	// // Time allowed to write a message to the peer.
	// writeWait := 5 * time.Second
	// ws.SetWriteDeadline(time.Now().Add(writeWait))
	// ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	// time.Sleep(writeWait)

	return ws.Close()
}

func (c *websocketConn) Write(b []byte) (n int, err error) {
	err = c.con.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *websocketConn) write(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expecting 1 parameter, got %d", len(args))
	}

	v := args[0]
	var b []byte

	switch v.Type {
	case core.String, core.Bytes:
		b = v.ToBytes()
	default:
		return core.NullValue, ErrInvalidType
	}

	n, err := c.Write(b)

	return core.NewInt(n), err
}

func (c *websocketConn) close(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expecting no parameters, got %d", len(args))
	}

	err := c.Close()
	return core.NullValue, err
}

func (c *websocketConn) readMessage(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expecting no parameters, got %d", len(args))
	}

	mType, msg, err := c.con.ReadMessage()
	if err != nil {
		return core.NullValue, err
	}

	result := make(map[string]core.Value)
	result["message"] = core.NewBytes(msg)
	result["type"] = core.NewInt(mType)
	return core.NewMapValues(result), nil
}

func (c *websocketConn) writeJSON(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expecting 1 parameter, got %d", len(args))
	}

	v := args[0].Export(0)

	b, err := json.Marshal(v)

	if err != nil {
		return core.NullValue, err
	}

	if err := vm.AddAllocations(len(b)); err != nil {
		return core.NullValue, err
	}

	err = c.con.WriteMessage(websocket.TextMessage, b)
	return core.NullValue, err
}

func (c *websocketConn) writeTextMessage(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expecting 1 parameter, got %d", len(args))
	}

	var b []byte
	a := args[0]

	switch a.Type {
	case core.String, core.Bytes:
		b = a.ToBytes()
	default:
		return core.NullValue, fmt.Errorf("invalid parameter type: %s", a.TypeName())
	}

	err := c.con.WriteMessage(websocket.TextMessage, b)
	return core.NullValue, err
}
