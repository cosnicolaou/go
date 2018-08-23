pulsemon
========

pulsemon is a Raspberry Pi based monitor designed to count and log pulses generated via a reed switch such as commonly used on water flow meters.
It uses a [PiFace 2](http://www.piface.org.uk/products/piface_digital_2/) board for it's I/O. 

The reed switch input is connected to one of the 5V GPIO pins on the piface 2 (though the pins seem to measure 3.3V on my board so maybe the documentation is off) and that input should be normally open. When the reed
switch is closed, the pulse can be measured. The switching time is slow and hence the input must be debounced.

pulsemon counts pulses in two ways:

- it shows the lowest 4 binary bits of the count on the piface leds.
- it writes a log file with the timestamp of each pulse logged. This log file is a binary file with time stamps encoded as go time.Time values. Simple
utilities are provided to read this file.

The log file is intended for post-hoc analysis such as comparing to a
monthly utility bill or other historical analysis.

pulsemon can send alert and status emails as follows:

- an alert email is sent when the flow is too high
- an alert email is sent if no flow is detected for some periond of time
- a status email is sent once per day showing the usage for the past 24 hours

In addition, pulsemon can be configured to forward any pulses received by
either triggering a relay or a cmos switch. The former is used to integrate
with an irrigation controller for instance, but the latter has not been
fully tested but could be similarly used.

A json config file is used to set the various parameters that control the alerts
and I/O configuration.
