package tasmota

import (
	"encoding/json"
	"fmt"
	"github.com/reef-pi/hal"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockTasmotaServer creates a mock Tasmota device server for testing
func mockTasmotaServer(t *testing.T) *httptest.Server {
	// In-memory store for device state
	powerStates := make(map[int]bool)
	dimmerStates := make(map[int]float64)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cmnd := r.URL.Query().Get("cmnd")
		if cmnd == "" {
			http.Error(w, "Missing cmnd parameter", http.StatusBadRequest)
			return
		}

		// Parse command - format: "Command<n> value" or "Command value"
		response := make(map[string]interface{})

		// Handle Power commands
		if len(cmnd) >= 5 && cmnd[:5] == "Power" {
			// Extract output number and value
			var outputNum int
			var value string

			// Check if it's Power<n> or just Power
			if len(cmnd) > 5 && cmnd[5] >= '0' && cmnd[5] <= '9' {
				// Power<n> format
				fmt.Sscanf(cmnd[5:], "%d", &outputNum)
				// Find where the number ends
				i := 5
				for i < len(cmnd) && cmnd[i] >= '0' && cmnd[i] <= '9' {
					i++
				}
				if i < len(cmnd) && cmnd[i] == ' ' {
					value = cmnd[i+1:]
				}
			} else if len(cmnd) > 6 && cmnd[5] == ' ' {
				// Power value format
				outputNum = 1
				value = cmnd[6:]
			} else {
				// Query format
				outputNum = 1
				value = ""
			}

			// Handle Power command
			if value == "" {
				// Query
				state, ok := powerStates[outputNum]
				if !ok {
					state = false
				}
				stateStr := "OFF"
				if state {
					stateStr = "ON"
				}
				if outputNum == 0 || outputNum == 1 {
					response["POWER"] = stateStr
				}
				if outputNum != 1 {
					response[fmt.Sprintf("POWER%d", outputNum)] = stateStr
				}
			} else if value == "1" || value == "ON" || value == "on" || value == "true" || value == "True" {
				powerStates[outputNum] = true
				stateStr := "ON"
				if outputNum == 0 || outputNum == 1 {
					response["POWER"] = stateStr
				}
				if outputNum != 1 {
					response[fmt.Sprintf("POWER%d", outputNum)] = stateStr
				}
			} else if value == "0" || value == "OFF" || value == "off" || value == "false" || value == "False" {
				powerStates[outputNum] = false
				stateStr := "OFF"
				if outputNum == 0 || outputNum == 1 {
					response["POWER"] = stateStr
				}
				if outputNum != 1 {
					response[fmt.Sprintf("POWER%d", outputNum)] = stateStr
				}
			}
		} else if len(cmnd) >= 6 && cmnd[:6] == "Dimmer" {
			// Handle Dimmer command
			var value float64
			if len(cmnd) > 7 && cmnd[6] == ' ' {
				fmt.Sscanf(cmnd[7:], "%f", &value)
				dimmerStates[0] = value
				if value > 0 {
					powerStates[0] = true
					response["POWER"] = "ON"
				} else {
					powerStates[0] = false
					response["POWER"] = "OFF"
				}
				response["Dimmer"] = int(value)
			} else {
				// Query
				value, ok := dimmerStates[0]
				if !ok {
					value = 0
				}
				response["Dimmer"] = int(value)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))

	return server
}

func TestHttpDriver_AsDigitalOut(t *testing.T) {

	server := mockTasmotaServer(t)
	defer server.Close()

	// Extract host:port from server URL
	address := server.URL[7:] // Remove "http://"

	f := HttpDriverFactory()

	params := map[string]interface{}{
		"Address": address,
		"Output":  "2",
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

	if len(o.DigitalOutputPins()) != 1 {
		t.Error("Expected a single digital output pin, found:", len(o.DigitalOutputPins()))
	}

	p, err := o.DigitalOutputPin(0)
	if err != nil || p == nil {
		t.Error("Expected a digital output pin")
	}

	if p.Name() != "Tasmota" {
		t.Error("Expected Tasmota name, found: ", p.Name())
	}

	if p.Number() != 0 {
		t.Error("Expected number 0, found: ", p.Number())
	}

	// Test with mock server
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

func TestHttpDriver_AsPWMDriver(t *testing.T) {

	server := mockTasmotaServer(t)
	defer server.Close()

	// Extract host:port from server URL
	address := server.URL[7:] // Remove "http://"

	f := HttpDriverFactory()

	params := map[string]interface{}{
		"Address": address,
		"Output":  "0",
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

	if p.Name() != "Tasmota" {
		t.Error("Expected Tasmota name, found: ", p.Name())
	}

	if p.Number() != 0 {
		t.Error("Expected number 0, found: ", p.Number())
	}

	// Test with mock server
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

func TestParseOutputs_SingleOutput(t *testing.T) {
	outputs, err := parseOutputs("1")
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if len(outputs) != 1 || outputs[0] != 1 {
		t.Errorf("Expected [1], got %v", outputs)
	}
}

func TestParseOutputs_DiscreteOutputs(t *testing.T) {
	outputs, err := parseOutputs("1,2,3")
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if len(outputs) != 3 || outputs[0] != 1 || outputs[1] != 2 || outputs[2] != 3 {
		t.Errorf("Expected [1, 2, 3], got %v", outputs)
	}
}

func TestParseOutputs_Range(t *testing.T) {
	outputs, err := parseOutputs("1-3")
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if len(outputs) != 3 || outputs[0] != 1 || outputs[1] != 2 || outputs[2] != 3 {
		t.Errorf("Expected [1, 2, 3], got %v", outputs)
	}
}

func TestParseOutputs_Mixed(t *testing.T) {
	outputs, err := parseOutputs("1-3,5,7-9")
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	expected := []int{1, 2, 3, 5, 7, 8, 9}
	if len(outputs) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(outputs))
	}
	for i, v := range expected {
		if outputs[i] != v {
			t.Errorf("Expected outputs[%d]=%d, got %d", i, v, outputs[i])
		}
	}
}

func TestParseOutputs_EmptyString(t *testing.T) {
	_, err := parseOutputs("")
	if err == nil {
		t.Error("Expected error for empty string")
	}
}

func TestParseOutputs_Duplicates(t *testing.T) {
	_, err := parseOutputs("1,1")
	if err == nil {
		t.Error("Expected error for duplicate output")
	}
}

func TestParseOutputs_DuplicatesInRange(t *testing.T) {
	_, err := parseOutputs("1-3,2")
	if err == nil {
		t.Error("Expected error for duplicate output in range")
	}
}

func TestParseOutputs_ReversedRange(t *testing.T) {
	_, err := parseOutputs("3-1")
	if err == nil {
		t.Error("Expected error for reversed range")
	}
}

func TestParseOutputs_NegativeNumbers(t *testing.T) {
	_, err := parseOutputs("-1")
	if err == nil {
		t.Error("Expected error for negative number")
	}
}

func TestParseOutputs_InvalidFormat(t *testing.T) {
	_, err := parseOutputs("abc")
	if err == nil {
		t.Error("Expected error for invalid format")
	}
}

func TestParseOutputs_Sorted(t *testing.T) {
	outputs, err := parseOutputs("3,1,2")
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if outputs[0] != 1 || outputs[1] != 2 || outputs[2] != 3 {
		t.Errorf("Expected sorted [1, 2, 3], got %v", outputs)
	}
}

func TestHttpDriver_FactoryValidateParameters(t *testing.T) {

	f := HttpDriverFactory()

	params := map[string]interface{}{
		"Address": "192.168.1.46",
	}

	_, err := f.NewDriver(params, nil)
	if err != nil {
		t.Fatal(err)
	}

	params = map[string]interface{}{
		"Address": "",
	}

	_, err = f.NewDriver(params, nil)
	if err == nil {
		t.Fatal("Expected error")
	}

	params = map[string]interface{}{
		"Address": 1,
	}

	_, err = f.NewDriver(params, nil)
	if err == nil {
		t.Fatal("Expected error")
	}

	params = map[string]interface{}{
		"Address": nil,
	}

	_, err = f.NewDriver(params, nil)
	if err == nil {
		t.Fatal("Expected error")
	}

}
