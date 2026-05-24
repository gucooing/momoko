package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"log"
	"mime"
	"momoko/pkg/cache"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"sync"
	texttemplate "text/template"
	"time"
)

const (
	defaultTimeout = 10 * time.Second
	defaultCcsN    = 5
)

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
	UseTLS   bool
	Timeout  time.Duration
	CcsN     int
}

type Message struct {
	Recipient string
	Template  string
	Subject   string
	Data      any
}

type Client struct {
	config   Config
	wg       sync.WaitGroup
	taskChan chan Message
	stopChan chan struct{}
	cooldown *cache.Cache[string, struct{}]
}

func NewClient(config Config) (*Client, error) {
	if config.Host == "" {
		return nil, errors.New("smtp host is required")
	}
	if config.CcsN <= 0 {
		config.CcsN = defaultCcsN
	}
	if config.Port == 0 {
		config.Port = 465
		if !config.UseTLS {
			config.Port = 25
		}
	}
	if config.Timeout <= 0 {
		config.Timeout = defaultTimeout
	}
	if _, err := mail.ParseAddress(config.From); err != nil {
		return nil, fmt.Errorf("invalid from address: %w", err)
	}

	return &Client{
		config:   config,
		wg:       sync.WaitGroup{},
		taskChan: make(chan Message),
		stopChan: make(chan struct{}),
		cooldown: cache.New[string, struct{}](30 * time.Second),
	}, nil
}

func Test(ctx context.Context, config Config, message Message) error {
	c := &Client{config: config}
	return c.send(ctx, message)
}

func (c *Client) Start() {
	for i := 0; i < c.config.CcsN; i++ {
		c.wg.Add(1)
		go c.worker(i)
	}
}

func (c *Client) Close() {
	close(c.stopChan)
}

func (c *Client) worker(id int) {
	defer c.wg.Done()

	for {
		select {
		case task, ok := <-c.taskChan:
			if !ok {
				return
			}
			c.processTask(id, task)
		case <-c.stopChan:
			return
		}
	}
}

func (c *Client) Send(ctx context.Context, message Message) error {
	select {
	case c.taskChan <- message:
		return nil
	case <-c.stopChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) processTask(id int, message Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	message.Subject = hasLineBreak(message.Subject)
	message.Recipient = hasLineBreak(message.Recipient)

	if _, ok := c.cooldown.Get(message.Recipient); ok {
		return
	}
	err := c.send(ctx, message)
	if err != nil {
		log.Printf("sending failed: %v", err)
		return
	}
	c.cooldown.Set(message.Recipient, struct{}{})
}

func (c *Client) send(ctx context.Context, message Message) error {
	// 构建消息
	data, err := c.buildMessage(message)
	if err != nil {
		return err
	}
	dialer := &net.Dialer{Timeout: c.config.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(c.config.Host, strconv.Itoa(c.config.Port)))
	if err != nil {
		return err
	}
	defer conn.Close()

	if c.config.UseTLS {
		conn = tls.Client(conn, &tls.Config{ServerName: c.config.Host, MinVersion: tls.VersionTLS12})
	}
	if err = conn.SetDeadline(time.Now().Add(c.config.Timeout)); err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, c.config.Host)
	if err != nil {
		return err
	}
	defer client.Close()

	// 发送消息
	if c.config.Username != "" || c.config.Password != "" {
		auth := smtp.PlainAuth("", c.config.Username, c.config.Password, c.config.Host)
		if err = client.Auth(auth); err != nil {
			return err
		}
	}
	if err = client.Mail(c.config.From); err != nil {
		return err
	}

	if err = client.Rcpt(message.Recipient); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = writer.Write(data); err != nil {
		_ = writer.Close()
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func (c *Client) buildMessage(message Message) ([]byte, error) {
	subject, err := renderText("subject", message.Subject, message.Data)
	if err != nil {
		return nil, err
	}
	if hasHeaderLineBreak(subject) {
		return nil, errors.New("invalid subject header")
	}
	if hasHeaderLineBreak(message.Recipient) {
		return nil, errors.New("invalid recipient header")
	}

	body, err := renderHTML("body", message.Template, message.Data)
	if err != nil {
		return nil, err
	}

	from := c.config.From
	if c.config.FromName != "" {
		from = (&mail.Address{Name: c.config.FromName, Address: c.config.From}).String()
	}

	var buf bytes.Buffer
	writeHeader(&buf, "From", from)
	writeHeader(&buf, "To", message.Recipient)
	writeHeader(&buf, "Subject", mime.QEncoding.Encode("UTF-8", subject))
	writeHeader(&buf, "MIME-Version", "1.0")
	writeHeader(&buf, "Content-Type", "text/html; charset=UTF-8")
	buf.WriteString("\r\n")
	buf.WriteString(body)
	return buf.Bytes(), nil
}

func renderText(name, value string, data any) (string, error) {
	tpl, err := texttemplate.New(name).Option("missingkey=error").Parse(value)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err = tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

func renderHTML(name, value string, data any) (string, error) {
	tpl, err := htmltemplate.New(name).Option("missingkey=error").Parse(value)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err = tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func writeHeader(buf *bytes.Buffer, key, value string) {
	buf.WriteString(key)
	buf.WriteString(": ")
	buf.WriteString(value)
	buf.WriteString("\r\n")
}

func hasLineBreak(value string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(value)
}

func hasHeaderLineBreak(value string) bool {
	return strings.ContainsAny(value, "\r\n")
}
