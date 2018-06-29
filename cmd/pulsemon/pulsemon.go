package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cosnicolaou/go/cmd/pulsemon/internal"
	"github.com/luismesas/goPi/piface"
	"github.com/luismesas/goPi/spi"
)

const (
	pollingInterval  = 10 * time.Millisecond
	pulseMeterPin    = 0
	pulseLEDPin      = 7
	debounceDuration = 100 * time.Millisecond
	debounceCount    = int(debounceDuration / pollingInterval)
)

var (
	// number of pulses since start.
	pulseCounter int64
	// channel used to send pulse timestamps from the polling loop
	// to any other interested process
	pulseTimes chan time.Time

	hostname string

	configFileFlag    string
	verboseFlag       bool
	timestampFileFlag string
	globalConfig      internal.Configuration
)

func init() {
	flag.StringVar(&configFileFlag, "config", "", "configuration file in JSON format")
	flag.BoolVar(&verboseFlag, "verbose", false, "output debug/trace information to the console")
	flag.StringVar(&timestampFileFlag, "read-timestamp-file", "", "if set, read and print the contents of the specified timestamps file and exit")
	hostname, _ = os.Hostname()
}

func main() {
	flag.Parse()

	if len(timestampFileFlag) > 0 {
		if err := internal.ReadTimestamps(timestampFileFlag); err != nil {
			fmt.Fprintf(os.Stderr, "failed to read or parse: %v: %v", timestampFileFlag, err)
		}
		return
	}

	if err := internal.ReadConfig(configFileFlag, &globalConfig); err != nil {
		panic(err)
	}

	smtpClient, err := globalConfig.ConfigureEmail(true)
	if err != nil {
		panic(err)
	}
	if smtpClient == nil {
		fmt.Printf("email alerts are not configured")
	}

	timestampWriter, err := internal.NewTimestampFileWriter(
		globalConfig.PulseTimestampFile)
	if err != nil {
		panic(err)
	}

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt, os.Kill, syscall.SIGTERM)

	pulseTimes = make(chan time.Time, 1024)

	// creates a new pifacedigital instance
	pfd := piface.NewPiFaceDigital(spi.DEFAULT_HARDWARE_ADDR, spi.DEFAULT_BUS, spi.DEFAULT_CHIP)

	// Initializes pifacedigital board
	if err := pfd.InitBoard(); err != nil {
		fmt.Printf("Error on init board: %s", err)
		return
	}

	// Log to console and append to the timestamp file.
	go console(pfd, timestampWriter, smtpClient, pulseTimes)

	// Generate an alert if a certain number of pulses per time period
	// are counted.
	go alert(globalConfig.AlertDuration,
		globalConfig.AlertPulses,
		int64(globalConfig.GallonsPerPulse),
		smtpClient)

	// Poll for pulses.
	go poll(pfd, pulseMeterPin, pulseTimes)

	go forward(pfd, 100*time.Millisecond, 0, 400*time.Millisecond)

	// Send a daily email.
	go daily(globalConfig.StatusTime, int64(globalConfig.GallonsPerPulse), smtpClient)

	<-sigch
	fmt.Printf("closing %v\n", globalConfig.PulseTimestampFile)
	timestampWriter.Close()
}

func console(pfd *piface.PiFaceDigital,
	timestampFile *internal.TimestampFileWriter,
	smtp *internal.SMTPClient,
	pulseTimes <-chan time.Time) {
	var prev, cur int64
	storage := make([]byte, 0, 128)
	buf := storage[:0]
	pfd.Leds[4].SetValue(0)
	pfd.Leds[5].SetValue(0)
	pfd.Leds[6].SetValue(0)
	pfd.Leds[7].SetValue(0)

	for {
		time.Sleep(500 * time.Millisecond)
		cur = atomic.LoadInt64(&pulseCounter)
		if cur != prev {
			prev = cur
			val := byte(cur & 0xff)
			pfd.Leds[4].SetValue(val & 0x01)
			pfd.Leds[5].SetValue((val & 0x02) >> 1)
			pfd.Leds[6].SetValue((val & 0x04) >> 2)
			pfd.Leds[7].SetValue((val & 0x08) >> 3)
			buf = strconv.AppendInt(storage, cur, 10)
			now := time.Now().String()
			buf = append(buf, ' ', '-', ' ')
			buf = append(buf, []byte(now)...)
			buf = append(buf, '\n')
			os.Stderr.Write(buf)
			n := 0
			for {
				// drain all event times.
				select {
				case event := <-pulseTimes:
					if err := timestampFile.Append(event); err != nil {
						msg := fmt.Sprintf("ERROR appending to timestamp file: %v", err)
						fmt.Fprintf(os.Stderr, "%s\n", msg)
						smtp.Send(msg)
					}
					n++
				default:
					goto drained
				}
			}
		drained:
			if verboseFlag {
				fmt.Fprintf(os.Stderr, "drained %v pulse timestamps", n)
			}
		}
	}
}

func alert(interval time.Duration, pulses int64, gallonsPerPulse int64, smtp *internal.SMTPClient) {
	last := atomic.LoadInt64(&pulseCounter)
	for {
		time.Sleep(interval)
		cur := atomic.LoadInt64(&pulseCounter)
		if seen := cur - last; seen > pulses {
			msg := fmt.Sprintf("ALERT: %v gallons over %v: %v\n", seen*gallonsPerPulse, interval, time.Now())
			os.Stdout.WriteString(msg)
			smtp.Send(msg)
		}
		last = cur
	}
}

func poll(pfd *piface.PiFaceDigital, pin int, pulseTimes chan<- time.Time) {
	fmt.Printf("polling pin %v\n", pin)
	count := debounceCount
	for {
		time.Sleep(pollingInterval)
		val := pfd.InputPins[pin].Value()
		if val == 0 {
			// Circuit is open.
			if count < 0 {
				count = debounceCount
			}
			continue
		}
		// Circuit is closed.

		// Debounce by waiting for debounceCount iterations before
		// counting a pulse. Once a pulse is counted, let the counter
		// run negative until the pin reads 0 again; ie. a rising
		// edge trigger for a pulse longer than debouceCount is counted.
		count--
		if count == 0 {
			atomic.AddInt64(&pulseCounter, 1)
			pulseTimes <- time.Now()
		}
	}
}

func forward(pfd *piface.PiFaceDigital, interval time.Duration, relayPin int, relayHold time.Duration) {
	pfd.Relays[relayPin].AllOff()
	last := atomic.LoadInt64(&pulseCounter)
	for {
		time.Sleep(interval)
		cur := atomic.LoadInt64(&pulseCounter)
		if seen := cur - last; seen > 0 {
			fmt.Fprintf(os.Stderr, "Forwarding %v pulses\n", seen)
			for i := int64(0); i < seen; i++ {
				pfd.Relays[relayPin].AllOn()
				time.Sleep(relayHold)
				pfd.Relays[relayPin].AllOff()
			}
		}
		last = cur
	}
}

func daily(hhmm time.Time, gallonsPerPulse int64, smtp *internal.SMTPClient) {
	for {
		duration := internal.UntilHHMM(hhmm)
		prev := atomic.LoadInt64(&pulseCounter)
		<-time.After(duration)
		// send email
		cur := atomic.LoadInt64(&pulseCounter)
		seen := cur - prev
		msg := fmt.Sprintf("ALERT: %v gallons over %v: %v\n", seen*gallonsPerPulse, duration, time.Now())
		smtp.Send(msg)
		prev = cur
	}
}
