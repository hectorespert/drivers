package tasmota

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/reef-pi/hal"
)

type tasmotaPin struct {
	address string
	number  int
}

func (p *tasmotaPin) Close() error {
	return nil
}

func (p *tasmotaPin) Name() string {
	return fmt.Sprintf("Tasmota Pin %d", p.number)
}

func (p *tasmotaPin) Number() int {
	return p.number
}

func (p *tasmotaPin) doRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	c := http.Client{
		Timeout: 5 * time.Second,
	}
	return c.Do(req)
}

func (p *tasmotaPin) readBody(body io.ReadCloser) ([]byte, error) {
	defer body.Close()
	msg, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (p *tasmotaPin) LastState() bool {
	const urlBase = "http://%s/cm?cmnd=Power%d"
	uri := fmt.Sprintf(urlBase, p.address, p.number)
	resp, err := p.doRequest(uri)
	if err != nil {
		return false
	}
	if resp.StatusCode != 200 {
		return false
	}
	body, err := p.readBody(resp.Body)
	if err != nil {
		return false
	}
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return false
	}

	if result[fmt.Sprintf("POWER%d", p.number)] == "ON" {
		return true
	}

	if result["POWER"] == "ON" {
		return true
	}

	return false
}

func (p *tasmotaPin) Set(value float64) error {
	const urlBase = "http://%s/cm?cmnd=Dimmer%d%%20%.0f"
	uri := fmt.Sprintf(urlBase, p.address, p.number, value)
	resp, err := p.doRequest(uri)
	if err != nil {
		return err
	}
	if resp.StatusCode == 200 {
		return nil
	}
	body, err := p.readBody(resp.Body)
	if err != nil {
		return err
	}
	return fmt.Errorf("HTTP Code:%d. Body:%v", resp.StatusCode, string(body))
}

func (p *tasmotaPin) Write(b bool) error {
	const baseUri = "http://%s/cm?cmnd=Power%d%%20%t"
	uri := fmt.Sprintf(baseUri, p.address, p.number, b)
	resp, err := p.doRequest(uri)
	if err != nil {
		return err
	}
	if resp.StatusCode == 200 {
		return nil
	}
	body, err := p.readBody(resp.Body)
	if err != nil {
		return err
	}
	return fmt.Errorf("HTTP Code:%d. Body:%v", resp.StatusCode, string(body))
}

type httpDriver struct {
	meta hal.Metadata
	pins []*tasmotaPin
}

func (m *httpDriver) Close() error {
	return nil
}

func (m *httpDriver) Metadata() hal.Metadata {
	return m.meta
}

func (m *httpDriver) Pins(capability hal.Capability) ([]hal.Pin, error) {
	switch capability {
	case hal.DigitalOutput, hal.PWM:
		pins := make([]hal.Pin, len(m.pins))
		for i, p := range m.pins {
			pins[i] = p
		}
		return pins, nil
	default:
		return nil, fmt.Errorf("unsupported capability:%s", capability.String())
	}
}

func (m *httpDriver) PWMChannels() []hal.PWMChannel {
	channels := make([]hal.PWMChannel, len(m.pins))
	for i, p := range m.pins {
		channels[i] = p
	}
	return channels
}

func (m *httpDriver) PWMChannel(pin int) (hal.PWMChannel, error) {
	if pin < 0 || pin >= len(m.pins) {
		return nil, fmt.Errorf("unknown pin: %d", pin)
	}
	return m.pins[pin], nil
}

func (m *httpDriver) DigitalOutputPins() []hal.DigitalOutputPin {
	pins := make([]hal.DigitalOutputPin, len(m.pins))
	for i, p := range m.pins {
		pins[i] = p
	}
	return pins
}

func (m *httpDriver) DigitalOutputPin(pin int) (hal.DigitalOutputPin, error) {
	if pin < 0 || pin >= len(m.pins) {
		return nil, fmt.Errorf("unknown pin: %d", pin)
	}
	return m.pins[pin], nil
}

type factory struct {
	meta       hal.Metadata
	parameters []hal.ConfigParameter
}

var pwmDriverFactory *factory
var once sync.Once

const address = "Address"
const outputs = "Outputs"

func HttpDriverFactory() hal.DriverFactory {

	once.Do(func() {
		pwmDriverFactory = &factory{
			meta: hal.Metadata{
				Name:         "Tasmota Http",
				Description:  "Tasmota Http Driver",
				Capabilities: []hal.Capability{hal.PWM, hal.DigitalOutput},
			},
			parameters: []hal.ConfigParameter{
				{
					Name:    address,
					Type:    hal.String,
					Order:   0,
					Default: "192.168.1.4",
				},
				{
					Name:    outputs,
					Type:    hal.Integer,
					Order:   1,
					Default: 1,
				},
			},
		}
	})

	return pwmDriverFactory
}

func (f *factory) GetParameters() []hal.ConfigParameter {
	return f.parameters
}

func (f *factory) ValidateParameters(parameters map[string]interface{}) (bool, map[string][]string) {
	var failures = make(map[string][]string)

	if v, ok := parameters[address]; ok {
		val, ok := v.(string)
		if !ok {
			failure := fmt.Sprint(address, " is not a string. ", v, " was received.")
			failures[address] = append(failures[address], failure)
		} else if len(val) <= 0 {
			failure := fmt.Sprint(address, " empty values are not allowed.")
			failures[address] = append(failures[address], failure)
		} else if len(val) >= 256 {
			failure := fmt.Sprint(address, " size should be lower than 255 characters. ", val, " was received.")
			failures[address] = append(failures[address], failure)
		}
	} else {
		failure := fmt.Sprint(address, " is a required parameter, but was not received.")
		failures[address] = append(failures[address], failure)
	}

	if v, ok := parameters[outputs]; ok {
		val, ok := v.(int)
		if !ok {
			failure := fmt.Sprint(outputs, " is not an integer. ", v, " was received.")
			failures[outputs] = append(failures[outputs], failure)
		} else if val < 1 {
			failure := fmt.Sprint(outputs, " value should be greater than 0. ", val, " was received.")
			failures[outputs] = append(failures[outputs], failure)
		}
	} else {
		failure := fmt.Sprint(outputs, " is a required parameter, but was not received.")
		failures[outputs] = append(failures[outputs], failure)
	}

	return len(failures) == 0, failures
}

func (f *factory) Metadata() hal.Metadata {
	return f.meta
}

func (f *factory) NewDriver(parameters map[string]interface{}, hardwareResources interface{}) (hal.Driver, error) {
	if parameters[outputs] == nil {
		parameters[outputs] = "1"
	}

	if outputStr, ok := parameters[outputs].(string); ok {
		if outputInt, err := strconv.Atoi(outputStr); err == nil {
			parameters[outputs] = outputInt
		}
	}

	if valid, failures := f.ValidateParameters(parameters); !valid {
		return nil, errors.New(hal.ToErrorString(failures))
	}

	addr := parameters[address].(string)
	numOutputs := parameters[outputs].(int)

	pins := make([]*tasmotaPin, numOutputs)
	for i := 0; i < numOutputs; i++ {
		pins[i] = &tasmotaPin{
			address: addr,
			number:  i + 1,
		}
	}

	driver := &httpDriver{
		meta: f.meta,
		pins: pins,
	}
	return driver, nil
}
