package main

import (
	"bytes"
	"fmt"
	"sync"
	"text/template"
	"time"

	"github.com/vanng822/go-premailer/premailer"
	mail "github.com/xhit/go-simple-mail/v2"
)

type Mail struct {
	Domain      string
	Host        string
	Port        int
	Username    string
	Password    string
	Encryption  string
	FromAddress string
	FromName    string
	Wait        *sync.WaitGroup
	MailerChan  chan Message
	ErrorChan   chan error
	DoneChan    chan bool
}

type Message struct {
	From          string
	FromName      string
	To            string
	Subject       string
	Attachment    []string
	AttachmentMap map[string]string
	Data          any
	DataMap       map[string]any
	Template      string
}

func (app *Config) ListenForMail() {
	for {
		select {
		case msg := <-app.Mailer.MailerChan:
			go app.Mailer.SendMail(msg, app.Mailer.ErrorChan)
		case err := <-app.Mailer.ErrorChan:
			app.ErrorLog.Println(err)
		case <-app.Mailer.DoneChan:
			return
		}
	}
}

// function to listen for messages
func (m *Mail) SendMail(msg Message, errorChan chan error) {
	defer m.Wait.Done()
	if msg.Template == "" {
		msg.Template = "mail"
	}

	if msg.From == "" {
		msg.From = m.FromAddress
	}

	if msg.FromName == "" {
		msg.FromName = m.FromName
	}
	if msg.AttachmentMap == nil {
		msg.AttachmentMap = make(map[string]string)
	}

	// data := map[string]any{
	// 	"message": msg.Data,
	// }
	if len(msg.DataMap) == 0 {
		msg.DataMap = make(map[string]any)
	}

	msg.DataMap["message"] = msg.Data

	// build html mail
	formatedMessage, err := m.buildHTMLMessage(msg)
	if err != nil {
		errorChan <- err
	}

	// build plain text mail
	plainMessage, err := m.buildPlainTextMessage(msg)
	if err != nil {
		errorChan <- err
	}

	server := mail.NewSMTPClient()
	server.Host = m.Host
	server.Port = m.Port
	server.Username = m.Username
	server.Password = m.Password
	server.Encryption = m.getEncription(m.Encryption)
	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second

	smtpClient, err := server.Connect()
	if err != nil {
		m.ErrorChan <- err
	}

	email := mail.NewMSG()
	email.SetFrom(msg.From).AddTo(msg.To).SetSubject(msg.Subject)
	email.SetBody(mail.TextPlain, plainMessage)
	email.AddAlternative(mail.TextHTML, formatedMessage)

	if len(msg.Attachment) > 0 {
		for _, x := range msg.Attachment {
			email.AddAttachment(x)
		}
	}
	if len(msg.AttachmentMap) > 0 {
		for key, value := range msg.AttachmentMap {
			email.AddAttachment(value, key)
		}
	}

	err = email.Send(smtpClient)
	if err != nil {
		m.ErrorChan <- err
	}

}

func (m *Mail) buildHTMLMessage(msg Message) (string, error) {
	rendertemplate := fmt.Sprintf("./cmd/web/templates/%s.html.gohtml", msg.Template)

	t, err := template.New("email-html").ParseFiles(rendertemplate)
	if err != nil {
		return "", err
	}
	var tpl bytes.Buffer
	if err = t.ExecuteTemplate(&tpl, "body", msg.DataMap); err != nil {
		return "", err
	}
	formatedmessafe := tpl.String()
	formatedmessafe, err = m.inlineCss(formatedmessafe)
	if err != nil {
		return "", err
	}
	return formatedmessafe, nil

}

func (m *Mail) buildPlainTextMessage(msg Message) (string, error) {
	rendertemplate := fmt.Sprintf("./cmd/web/templates/%s.plain.gohtml", msg.Template)

	t, err := template.New("email-plain").ParseFiles(rendertemplate)
	if err != nil {
		return "", err
	}
	var tpl bytes.Buffer
	if err = t.ExecuteTemplate(&tpl, "body", msg.DataMap); err != nil {
		return "", err
	}
	plainMessage := tpl.String()

	return plainMessage, nil
}

func (m *Mail) inlineCss(s string) (string, error) {
	options := premailer.Options{
		RemoveClasses:     false,
		CssToAttributes:   false,
		KeepBangImportant: true,
	}
	prem, err := premailer.NewPremailerFromString(s, &options)
	if err != nil {
		return "", err
	}
	html, err := prem.Transform()
	if err != nil {
		return "", err
	}
	return html, nil
}

func (m *Mail) getEncription(e string) mail.Encryption {
	switch e {
	case "tls":
		return mail.EncryptionSTARTTLS
	case "ssl":
		return mail.EncryptionSSLTLS
	case "none":
		return mail.EncryptionNone
	default:
		return mail.EncryptionSTARTTLS
	}
}
