package tasmota

import (
	"os"
	"testing"

	"github.com/reef-pi/hal"
)

func TestHttpDriver_AsDigitalOut(t *testing.T) {

	address := os.Getenv("TASMOTA_TEST_ADDRESS")

	if len(address) == 0 {
		address = "192.168.1.46"
	}

	f := HttpDriverFactory()

	params := map[string]interface{}{
		"Address": address,
		"Outputs": "2",
	}

	d, err := f.NewDriver(params, nil)
	if err != nil {
		t.Fatal(err)
	}

	meta := d.Metadata()
	if len(meta.Capabilities) != 2 {
		t.Error("Expected 2 capabilities, found:", len(meta.Capabilities))
	}

	o, ok := d.(hal.DigitalOutputDriver)
	if !ok {
		t.Error("Failed to type driver to Digital output driver")
	}

	if len(o.DigitalOutputPins()) != 2 {
		t.Error("Expected 2 digital output pins, found:", len(o.DigitalOutputPins()))
	}

	p, err := o.DigitalOutputPin(0)
	if err != nil || p == nil {
		t.Error("Expected a digital output pin")
	}

	if p.Name() != "Tasmota Pin 1" {
		t.Error("Expected 'Tasmota Pin 1' name, found: ", p.Name())
	}

	if p.Number() != 1 {
		t.Error("Expected number 1, found: ", p.Number())
	}

	p2, err := o.DigitalOutputPin(1)
	if err != nil || p2 == nil {
		t.Error("Expected a second digital output pin")
	}

	if p2.Name() != "Tasmota Pin 2" {
		t.Error("Expected 'Tasmota Pin 2' name, found: ", p2.Name())
	}

	if p2.Number() != 2 {
		t.Error("Expected number 2, found: ", p2.Number())
	}

	_, err = o.DigitalOutputPin(2)
	if err == nil {
		t.Error("Expected error for out of range pin")
	}

	testRealDevice := os.Getenv("TASMOTA_TEST_REAL_DEVICE")

	if testRealDevice == "True" {

		err = p.Write(true)
		if err != nil {
			t.Error("Expected write true in the digital output, error: ", err.Error())
		}

		if !p.LastState() {
			t.Error("Expected last state is true")
		}

		err = p.Write(false)
		if err != nil {
			t.Error("Expected write false in the digital output, error: ", err.Error())
		}

		if p.LastState() {
			t.Error("Expected last state is false")
		}
	}

}

func TestHttpDriver_AsPWMDriver(t *testing.T) {

	address := os.Getenv("TASMOTA_TEST_ADDRESS")

	if len(address) == 0 {
		address = "192.168.1.46"
	}

	f := HttpDriverFactory()

	params := map[string]interface{}{
		"Address": address,
		"Outputs": "1",
	}

	d, err := f.NewDriver(params, nil)
	if err != nil {
		t.Fatal(err)
	}

	meta := d.Metadata()
	if len(meta.Capabilities) != 2 {
		t.Error("Expected 2 capabilities, found:", len(meta.Capabilities))
	}

	pwm, ok := d.(hal.PWMDriver)
	if !ok {
		t.Error("Failed to type driver to PWM driver")
	}

	if len(pwm.PWMChannels()) != 1 {
		t.Error("Expected a single pwm channel, found:", len(pwm.PWMChannels()))
	}

	p, err := pwm.PWMChannel(0)
	if err != nil {
		t.Error("Expected a pwm pin")
	}

	if p.Name() != "Tasmota Pin 1" {
		t.Error("Expected 'Tasmota Pin 1' name, found: ", p.Name())
	}

	if p.Number() != 1 {
		t.Error("Expected number 1, found: ", p.Number())
	}

	_, err = pwm.PWMChannel(1)
	if err == nil {
		t.Error("Expected error for out of range PWM channel")
	}

	testRealDevice := os.Getenv("TASMOTA_TEST_REAL_DEVICE")

	if testRealDevice == "True" {

		err = p.Set(100)
		if err != nil {
			t.Error("Expected to set 100 in the pwm output, error: ", err.Error())
		}

		if !p.LastState() {
			t.Error("Expected last state is true")
		}

		err = p.Set(0)
		if err != nil {
			t.Error("Expected to set 0 in the pwm output, error: ", err.Error())
		}

		if p.LastState() {
			t.Error("Expected last state is false")
		}
	}

}

func TestHttpDriver_FactoryValidateParameters(t *testing.T) {

	f := HttpDriverFactory()

	params := map[string]interface{}{
		"Address": "192.168.1.46",
		"Outputs": "1",
	}

	_, err := f.NewDriver(params, nil)
	if err != nil {
		t.Fatal(err)
	}

	params = map[string]interface{}{
		"Address": "",
		"Outputs": "1",
	}

	_, err = f.NewDriver(params, nil)
	if err == nil {
		t.Fatal("Expected error")
	}

	params = map[string]interface{}{
		"Address": 1,
		"Outputs": "1",
	}

	_, err = f.NewDriver(params, nil)
	if err == nil {
		t.Fatal("Expected error")
	}

	params = map[string]interface{}{
		"Address": nil,
		"Outputs": "1",
	}

	_, err = f.NewDriver(params, nil)
	if err == nil {
		t.Fatal("Expected error")
	}

}

func TestHttpDriver_DefaultOutputs(t *testing.T) {

	f := HttpDriverFactory()

	params := map[string]interface{}{
		"Address": "192.168.1.46",
	}

	d, err := f.NewDriver(params, nil)
	if err != nil {
		t.Fatal(err)
	}

	o, ok := d.(hal.DigitalOutputDriver)
	if !ok {
		t.Fatal("Failed to type driver to Digital output driver")
	}

	if len(o.DigitalOutputPins()) != 1 {
		t.Error("Expected 1 digital output pin by default, found:", len(o.DigitalOutputPins()))
	}
}

func TestHttpDriver_MultipleOutputs(t *testing.T) {

	f := HttpDriverFactory()

	params := map[string]interface{}{
		"Address": "192.168.1.46",
		"Outputs": "4",
	}

	d, err := f.NewDriver(params, nil)
	if err != nil {
		t.Fatal(err)
	}

	o, ok := d.(hal.DigitalOutputDriver)
	if !ok {
		t.Fatal("Failed to type driver to Digital output driver")
	}

	if len(o.DigitalOutputPins()) != 4 {
		t.Error("Expected 4 digital output pins, found:", len(o.DigitalOutputPins()))
	}

	for i := 0; i < 4; i++ {
		pin, err := o.DigitalOutputPin(i)
		if err != nil {
			t.Errorf("Expected pin %d, got error: %v", i, err)
		}
		expectedName := "Tasmota Pin " + string(rune('1'+i))
		if pin.Number() != i+1 {
			t.Errorf("Expected pin number %d, found: %d", i+1, pin.Number())
		}
		_ = expectedName
	}

	pins, err := d.Pins(hal.DigitalOutput)
	if err != nil {
		t.Fatal(err)
	}
	if len(pins) != 4 {
		t.Error("Expected 4 pins from Pins(), found:", len(pins))
	}

	pins, err = d.Pins(hal.PWM)
	if err != nil {
		t.Fatal(err)
	}
	if len(pins) != 4 {
		t.Error("Expected 4 PWM pins from Pins(), found:", len(pins))
	}

	_, err = d.Pins(hal.DigitalInput)
	if err == nil {
		t.Error("Expected error for unsupported capability")
	}
}
