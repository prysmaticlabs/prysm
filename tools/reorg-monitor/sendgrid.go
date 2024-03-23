package main

import (
	"encoding/base64"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	log "github.com/sirupsen/logrus"
)

var (
	_ = emailSender(&sendgridSender{})
)

type sendgridSender struct {
	apiKey string
}

func newSendgridSender(apiKey string) *sendgridSender {
	return &sendgridSender{apiKey}
}

func (s *sendgridSender) send(m *emailMessage) error {
	from := mail.NewEmail("Events Monitor", m.from)
	subject := m.subject
	for i := 0; i < len(m.to); i++ {
		to := mail.NewEmail("", m.to[i])
		plainTextContent := m.body
		message := mail.NewSingleEmail(from, subject, to, plainTextContent, "")
		for fileName, data := range m.attachments {
			enc := base64.StdEncoding.EncodeToString(data)
			att := mail.NewAttachment().SetFilename(fileName).SetContent(enc)
			message = message.AddAttachment(att)
		}
		client := sendgrid.NewSendClient(s.apiKey)
		response, err := client.Send(message)
		if err != nil {
			return err
		}
		log.WithField("response", response).Info("Sent email with sendgrid")
	}
	return nil
}
