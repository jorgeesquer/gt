package lib

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"net/mail"
	"path/filepath"
	"github.com/gtlang/gt/core"
	"strconv"
	"strings"
	"time"
)

func init() {
	core.RegisterLib(STMP, `

declare namespace smtp {
    export function newMessage(): Message

    export function send(
        msg: Message,
        user: string,
        password: string,
        host: string,
        port: number,
        insecureSkipVerify?: boolean): void

    export interface Message {
        from: string
        fromName: string
        to: string[]
        cc: string[]
        bcc: string[]
        replyTo: string
        subject: string
        body: string
        html: boolean
        toString(): string
        attach(fileName: string, data: byte[], inline: boolean): void
    }
}


`)
}

var STMP = []core.NativeFunction{
	core.NativeFunction{
		Name:      "smtp.newMessage",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewObject(&SmtMessage{}), nil
		},
	},
	core.NativeFunction{
		Name:      "smtp.send",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateOptionalArgs(args, core.Object, core.String, core.String, core.String, core.Int, core.Bool); err != nil {
				return core.NullValue, err
			}

			msg, ok := args[0].ToObject().(*SmtMessage)
			if !ok {
				return core.NullValue, fmt.Errorf("expected a mail message, got %s", args[0].TypeName())
			}

			var err error
			var user, smtPasswd, host string
			var port int
			var skipVerify bool

			switch len(args) {
			case 5:
				user = args[1].ToString()
				smtPasswd = args[2].ToString()
				host = args[3].ToString()
				port = int(args[4].ToInt())

			case 6:
				user = args[1].ToString()
				smtPasswd = args[2].ToString()
				host = args[3].ToString()
				port = int(args[4].ToInt())
				skipVerify = args[5].ToBool()
			default:
				return core.NullValue, fmt.Errorf("expected 4 or 5 params, got %d", len(args))
			}

			err = msg.Send(user, smtPasswd, host, port, skipVerify)
			return core.NullValue, err
		},
	},
}

// Message represents a smtp message.
type SmtMessage struct {
	From        mail.Address
	To          core.Value
	Cc          core.Value
	Bcc         core.Value
	ReplyTo     string
	Subject     string
	Body        string
	Html        bool
	Attachments map[string]*attachment
}

func (SmtMessage) Type() string {
	return "smtp.Message"
}

func (m *SmtMessage) GetProperty(key string, vm *core.VM) (core.Value, error) {
	switch key {
	case "from":
		return core.NewString(m.From.Address), nil
	case "fromName":
		return core.NewString(m.From.Name), nil
	case "to":
		if m.To.Type == core.Null {
			m.To = core.NewArray(0)
		}
		return m.To, nil
	case "cc":
		if m.Cc.Type == core.Null {
			m.Cc = core.NewArray(0)
		}
		return m.Cc, nil
	case "bcc":
		if m.Bcc.Type == core.Null {
			m.Bcc = core.NewArray(0)
		}
		return m.Bcc, nil
	case "replyTo":
		return core.NewString(m.ReplyTo), nil
	case "subject":
		return core.NewString(m.Subject), nil
	case "body":
		return core.NewString(m.Body), nil
	case "html":
		return core.NewBool(m.Html), nil
	}
	return core.UndefinedValue, nil
}

func (m *SmtMessage) SetProperty(key string, v core.Value, vm *core.VM) error {
	switch key {
	case "from":
		if v.Type != core.String {
			return ErrInvalidType
		}
		m.From.Address = v.ToString()
		return nil
	case "fromName":
		if v.Type != core.String {
			return ErrInvalidType
		}

		m.From.Name = v.ToString()
		return nil
	case "to":
		if v.Type != core.Array {
			return ErrInvalidType
		}
		m.To = v
		return nil
	case "cc":
		if v.Type != core.Array {
			return ErrInvalidType
		}
		m.Cc = v
		return nil
	case "bcc":
		if v.Type != core.Array {
			return ErrInvalidType
		}
		m.Bcc = v
		return nil
	case "replyTo":
		if v.Type != core.String {
			return ErrInvalidType
		}
		m.ReplyTo = v.ToString()
		return nil
	case "subject":
		if v.Type != core.String {
			return ErrInvalidType
		}
		m.Subject = v.ToString()
		return nil
	case "body":
		if v.Type != core.String {
			return ErrInvalidType
		}
		m.Body = v.ToString()
		return nil
	case "html":
		if v.Type != core.Bool {
			return ErrInvalidType
		}
		m.Html = v.ToBool()
		return nil
	}

	return ErrReadOnlyOrUndefined
}

func (m *SmtMessage) GetMethod(name string) core.NativeMethod {
	switch name {
	case "attach":
		return m.attachData
	case "toString":
		return m.toString
	}
	return nil
}

func (m *SmtMessage) toString(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgRange(args, 0, 0); err != nil {
		return core.NullValue, err
	}
	return core.NewString(string(m.Bytes())), nil
}

func (m *SmtMessage) attachData(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgRange(args, 2, 3); err != nil {
		return core.NullValue, err
	}

	var fileName string
	var data []byte
	var inline bool

	a := args[0]
	switch a.Type {
	case core.String:
		fileName = a.ToString()
	default:
		return core.NullValue, fmt.Errorf("invalid argument 1 type: %v", a)
	}

	b := args[1]
	switch b.Type {
	case core.Bytes, core.String:
		data = b.ToBytes()
	default:
		return core.NullValue, fmt.Errorf("invalid argument 2 type: %v", b)
	}

	if len(args) == 3 {
		c := args[2]
		switch c.Type {
		case core.Bool:
			inline = c.ToBool()
		default:
			return core.NullValue, fmt.Errorf("invalid argument 2 type: %v", c)
		}
	}

	err := m.AttachBuffer(fileName, data, inline)
	return core.NullValue, err
}

func (m *SmtMessage) ContentType() string {
	if m.Html {
		return "text/html"
	}
	return "text/plain"
}

// Send sends the message.
func (m *SmtMessage) Send(user, password, host string, port int, insecureSkipVerify bool) error {
	auth := PlainAuth("", user, password, host)
	address := host + ":" + strconv.Itoa(port)

	return SendMail(address, auth, m.From.Address, m.AllRecipients(), m.Bytes(), insecureSkipVerify)
}

// AttachBuffer attaches a binary attachment.
func (m *SmtMessage) AttachBuffer(filename string, buf []byte, inline bool) error {
	if m.Attachments == nil {
		m.Attachments = make(map[string]*attachment)
	}

	m.Attachments[filename] = &attachment{
		Filename: filename,
		Data:     buf,
		Inline:   inline,
	}
	return nil
}

// Attachment represents an email attachment.
type attachment struct {
	Filename string
	Data     []byte
	Inline   bool
}

func (m *SmtMessage) ToList() []string {
	if m.To.Type == core.Null {
		return []string{}
	}

	a := m.To.ToArray()
	dirs := make([]string, len(a))
	for i, v := range a {
		dirs[i] = v.ToString()
	}
	return dirs
}

func (m *SmtMessage) CcList() []string {
	if m.Cc.Type == core.Null {
		return []string{}
	}

	a := m.Cc.ToArray()
	dirs := make([]string, len(a))
	for i, v := range a {
		dirs[i] = v.ToString()
	}
	return dirs
}

func (m *SmtMessage) BccList() []string {
	if m.Bcc.Type == core.Null {
		return []string{}
	}

	a := m.Bcc.ToArray()
	dirs := make([]string, len(a))
	for i, v := range a {
		dirs[i] = v.ToString()
	}
	return dirs
}

// Tolist returns all the recipients of the email
func (m *SmtMessage) AllRecipients() []string {
	dirs := m.ToList()
	dirs = append(dirs, m.CcList()...)
	dirs = append(dirs, m.BccList()...)
	return dirs
}

// Bytes returns the mail data
func (m *SmtMessage) Bytes() []byte {
	buf := bytes.NewBuffer(nil)

	buf.WriteString("From: " + m.From.String() + "\r\n")

	buf.WriteString("Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\n")

	buf.WriteString("To: " + strings.Join(m.ToList(), ",") + "\r\n")

	cc := m.CcList()
	if len(cc) > 0 {
		buf.WriteString("Cc: " + strings.Join(cc, ",") + "\r\n")
	}

	bcc := m.BccList()
	if len(bcc) > 0 {
		buf.WriteString("Bcc: " + strings.Join(bcc, ",") + "\r\n")
	}

	//fix  Encode
	var coder = base64.StdEncoding
	var subject = "=?UTF-8?B?" + coder.EncodeToString([]byte(m.Subject)) + "?="
	buf.WriteString("Subject: " + subject + "\r\n")

	if len(m.ReplyTo) > 0 {
		buf.WriteString("Reply-To: " + m.ReplyTo + "\r\n")
	}

	buf.WriteString("MIME-Version: 1.0\r\n")

	boundary := "f46d043c813270fc6b04c2d223da"

	if len(m.Attachments) > 0 {
		buf.WriteString("Content-Type: multipart/mixed; boundary=" + boundary + "\r\n")
		buf.WriteString("\r\n--" + boundary + "\r\n")
	}

	buf.WriteString(fmt.Sprintf("Content-Type: %s; charset=utf-8\r\n\r\n", m.ContentType()))
	buf.WriteString(m.Body)
	buf.WriteString("\r\n")

	if len(m.Attachments) > 0 {
		for _, attachment := range m.Attachments {
			buf.WriteString("\r\n\r\n--" + boundary + "\r\n")

			if attachment.Inline {
				buf.WriteString("Content-Type: message/rfc822\r\n")
				buf.WriteString("Content-Disposition: inline; filename=\"" + attachment.Filename + "\"\r\n\r\n")

				buf.Write(attachment.Data)
			} else {
				ext := filepath.Ext(attachment.Filename)
				mimetype := mime.TypeByExtension(ext)
				if mimetype != "" {
					mime := fmt.Sprintf("Content-Type: %s\r\n", mimetype)
					buf.WriteString(mime)
				} else {
					buf.WriteString("Content-Type: application/octet-stream\r\n")
				}
				buf.WriteString("Content-Transfer-Encoding: base64\r\n")

				buf.WriteString("Content-Disposition: attachment; filename=\"=?UTF-8?B?")
				buf.WriteString(coder.EncodeToString([]byte(attachment.Filename)))
				buf.WriteString("?=\"\r\n\r\n")

				b := make([]byte, base64.StdEncoding.EncodedLen(len(attachment.Data)))
				base64.StdEncoding.Encode(b, attachment.Data)

				// write base64 content in lines of up to 76 chars
				for i, l := 0, len(b); i < l; i++ {
					buf.WriteByte(b[i])
					if (i+1)%76 == 0 {
						buf.WriteString("\r\n")
					}
				}
			}

			buf.WriteString("\r\n--" + boundary)
		}

		buf.WriteString("--")
	}

	return buf.Bytes()
}
