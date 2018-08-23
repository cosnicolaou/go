package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"
)

var hostname string

func init() {
	hostname, _ = os.Hostname()
}

// Configuration represents the configuration for the pulsemon family of tools.
type Configuration struct {
	// SMTP configuration for alert emails.
	Server  string   `json:"smtp_server"`
	Port    string   `json:"smtp_port"`
	User    string   `json:"smtp_user"`
	Passwd  string   `json:"smtp_password"`
	To      []string `json:"smtp_to"`
	From    string   `json:"smtp_from"`
	Subject string   `json:"smtp_subject"`

	// Set the time of day to send a status email at in HH:MM format.
	StatusEmailTime    string `json:"status_email_time"`
	StatisEmailSubject string `json:"status_email_subject"`

	// Alert configuation, if more than AlertPulses are counted
	// over AlertInterval then an email is sent.
	AlertInterval     string `json:"alert_interval"`
	IdleAlertInterval string `json:"idle_alert_interval"`
	AlertPulses       int64  `json:"alert_pulses"`

	// Number of gallons per pulse.
	GallonsPerPulse int `json:"gallons_per_pulse"`

	// Record the time of each pulse in binary, little endian, 64 bit unix
	// nanoseconds.
	PulseTimestampFile string `json:"pulse_timestamps_file"`

	// Parsed and processed configuration information.

	// AlertInterval as a time.Duration.
	AlertDuration time.Duration

	// IdleAlertInterval as a time.Duration
	IdleAlertDuration time.Duration

	// StatusEmailTime as a time.Time.
	StatusTime time.Time

	PollingInterval int `json:"polling_interval_ms"`

	// Hardware specific configuration, doesn't really belong here. Set to
	// -1 to disable.
	InputPin          int `json:"input_pin"`
	InputDebounceMS   int `json:"input_debounce_ms"`
	OutputRelayPin    int `json:"relay_pin"`
	OutputRelayHoldMS int `json:"relay_hold_ms"`
	OutputPin         int `json:"output_pin"`
	OutputPinHoldMS   int `json:"output_hold_ms"`
}

// ReadConfig reads the configuration from the specified file.
func ReadConfig(filename string, config *Configuration) error {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read: %v", filename)
	}
	if err := json.Unmarshal(buf, config); err != nil {
		return fmt.Errorf("failed to unmarshal %v: %v", filename, err)
	}
	interval, err := time.ParseDuration(config.AlertInterval)
	if err != nil {
		return fmt.Errorf("failed to parse alert_interval %v as time.Duration: %v", config.AlertInterval, err)
	}

	idle, err := time.ParseDuration(config.IdleAlertInterval)
	if err != nil {
		return fmt.Errorf("failed to parse idle_alert_interval %v as time.Duration: %v", config.IdleAlertInterval, err)
	}

	emailAt, err := time.Parse("15:04", config.StatusEmailTime)
	if err != nil {
		return fmt.Errorf("failed to parse %v in 15:04 format", config.StatusEmailTime)
	}

	config.StatusTime = emailAt
	config.AlertDuration = interval
	config.IdleAlertDuration = idle
	return nil
}

type SMTPClient struct {
	auth                        smtp.Auth
	to                          []string
	host, from                  string
	alertSubject, statusSubject string
}

func (sc *SMTPClient) String() string {
	return fmt.Sprintf("%v: from %v, to %v, alert subject: %v, status subject %v", sc.host, sc.from, strings.Join(sc.to, ","), sc.alertSubject, sc.statusSubject)
}

// ConfigureEmail configures and optional tests an smtp email client by sending
// a 'hello' message.
func (config *Configuration) ConfigureEmail(sendHello bool) (*SMTPClient, error) {
	client := &SMTPClient{
		host:          net.JoinHostPort(config.Server, config.Port),
		to:            config.To,
		from:          config.From,
		alertSubject:  config.Subject,
		statusSubject: config.StatisEmailSubject,
	}
	if len(client.host) == 0 {
		return nil, nil
	}
	client.auth = smtp.PlainAuth("", config.User, config.Passwd, config.Server)
	err := client.Status(fmt.Sprintf("%v started on %v @ %v\n", os.Args[0], hostname, time.Now()))
	if err != nil {
		return nil, err
	}
	fmt.Printf("sent hello email to %v\n", strings.Join(client.to, ","))
	return client, nil
}

// Alert sends an alert email.
func (sc *SMTPClient) Alert(body string) error {
	return sc.Send(sc.alertSubject, body)
}

// Status sends a status email.
func (sc *SMTPClient) Status(body string) error {
	return sc.Send(sc.statusSubject, body)
}

// Send sends a generic email.
func (sc *SMTPClient) Send(subject, body string) error {
	if sc == nil || sc.auth == nil {
		return nil
	}
	msg := fmt.Sprintf("To: %v\r\nSubject: %v\r\n\r\n%v\r\nHost: %v\r\n",
		sc.to, subject, body, hostname)
	err := smtp.SendMail(sc.host, sc.auth, sc.from, sc.to, []byte(msg))
	if err != nil {
		err = fmt.Errorf("smtp.SendMail failed: %v, from: %v, to: %v", sc.host, sc.from, sc.to, err)
	}
	return err
}
