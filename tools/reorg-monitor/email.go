package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/smtp"
	"strings"
)

var (
	_ = emailSender(&basicSmtpSender{})
)

type emailSender interface {
	send(m *emailMessage) error
}

type basicSmtpSender struct {
	host       string
	portNumber string
	username   string
	auth       smtp.Auth
}

func newBasicSmtpSender(auth smtp.Auth, host, port string) *basicSmtpSender {
	return &basicSmtpSender{auth: auth, host: host, portNumber: port}
}

func (s *basicSmtpSender) send(m *emailMessage) error {
	return smtp.SendMail(fmt.Sprintf("%s:%s", s.host, s.portNumber), s.auth, s.username, m.to, m.toBytes())
}

type emailMessage struct {
	to          []string
	from        string
	cc          []string
	bcc         []string
	subject     string
	body        string
	attachments map[string][]byte
}

func newEmailMessage(s, b string) *emailMessage {
	return &emailMessage{subject: s, body: b, attachments: make(map[string][]byte)}
}

func (m *emailMessage) toBytes() []byte {
	buf := bytes.NewBuffer(nil)
	withAttachments := len(m.attachments) > 0
	buf.WriteString(fmt.Sprintf("Subject: %s\n", m.subject))
	buf.WriteString(fmt.Sprintf("From: %s\n", m.from))
	buf.WriteString(fmt.Sprintf("To: %s\n", strings.Join(m.to, ",")))
	if len(m.cc) > 0 {
		buf.WriteString(fmt.Sprintf("Cc: %s\n", strings.Join(m.cc, ",")))
	}

	if len(m.bcc) > 0 {
		buf.WriteString(fmt.Sprintf("Bcc: %s\n", strings.Join(m.bcc, ",")))
	}

	buf.WriteString("MIME-Version: 1.0\n")
	writer := multipart.NewWriter(buf)
	boundary := writer.Boundary()
	if withAttachments {
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\n", boundary))
		buf.WriteString(fmt.Sprintf("--%s\n", boundary))
	} else {
		buf.WriteString("Content-Type: text/plain; charset=utf-8\n")
	}

	buf.WriteString(m.body)
	if withAttachments {
		for k, v := range m.attachments {
			buf.WriteString(fmt.Sprintf("\n\n--%s\n", boundary))
			buf.WriteString(fmt.Sprintf("Content-Type: %s\n", http.DetectContentType(v)))
			buf.WriteString("Content-Transfer-Encoding: base64\n")
			buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=%s\n", k))

			b := make([]byte, base64.StdEncoding.EncodedLen(len(v)))
			base64.StdEncoding.Encode(b, v)
			buf.Write(b)
			buf.WriteString(fmt.Sprintf("\n--%s", boundary))
		}

		buf.WriteString("--")
	}

	return buf.Bytes()
}
