package main

import (
	"errors"
	"flag"
	"fmt"
	"strconv"

	"github.com/golang/glog"
	"github.com/sodabrew/chilctl/cx34"
	"github.com/sodabrew/chilctl/units"
)

var (
	ttyDevice      = flag.String("tty", "/dev/ttyUSB0", "Path to RS-4845 serial port.")
	unitId         = flag.Int("unit", 1, "Device unit id number.")
	rawFlag        = flag.Bool("raw", false, "Print the raw register values.")
	setMode        = flag.String("set-mode", "", "Set heating/cooling mode: H, C, HW, CW")
	setHeatingTemp = flag.String("set-heating-temp", "", "Set heating target temperature (must add C or F suffix, e.g. 35C)")
	setCoolingTemp = flag.String("set-cooling-temp", "", "Set cooling target temperature (must add C or F suffix, e.g. 50F)")
	setDHWTemp     = flag.String("set-dhw-temp", "", "Set the domestic hot water target temperature (must add C or F suffix, e.g. 130F)")
	versionFlag    = flag.Bool("version", false, "Return the version of the program.")
)

const (
	version = "v0.1"
)

func main() {
	flag.Parse()
	if *versionFlag {
		fmt.Printf("%s\n", version)
		return
	}

	cxClient, err := cx34.Connect(&cx34.Params{
		TTYDevice: *ttyDevice,
		Mode:      cx34.Modbus,
		UnitId:    *unitId,
	})
	if err != nil {
		glog.Errorf("error connecting to CX34: %v", err)
		return
	}

	state, err := cxClient.ReadState()
	if err != nil {
		glog.Errorf("error getting CX34 state: %v", err)
	}

	if *rawFlag {
		fmt.Printf("%+v\n", state)
	} else {
		printState(state)
	}

	if *setCoolingTemp != "" {
		temp, err := parseTemperatureFlag(*setCoolingTemp)
		if err != nil{
			glog.Errorf("error parsing temperature: %v", err)
			return
		}
		fmt.Printf("Setting cooling target temp to %.2f°F\n", temp.Fahrenheit())
		cxClient.SetCoolingTemp(temp)
	}

	if *setHeatingTemp != "" {
		temp, err := parseTemperatureFlag(*setHeatingTemp)
		if err != nil{
			glog.Errorf("error parsing temperature: %v", err)
			return
		}
		fmt.Printf("Setting heating target temp to %.2f°F\n", temp.Fahrenheit())
		cxClient.SetHeatingTemp(temp)
	}

	if *setDHWTemp != "" {
		temp, err := parseTemperatureFlag(*setHeatingTemp)
		if err != nil{
			glog.Errorf("error parsing temperature: %v", err)
			return
		}
		fmt.Printf("Setting DHW target temp to %.2f°F\n", temp.Fahrenheit())
		cxClient.SetDomesticHotWaterTemp(temp)
	}

	if *setMode != "" {
		var mode cx34.AirConditioningMode = 0
		switch *setMode {
		case "C":
			mode = cx34.AirConditioningModeCooling
		case "H":
			mode = cx34.AirConditioningModeHeating
		case "CW":
			mode = cx34.AirConditioningModeCoolDHW
		case "HW":
			mode = cx34.AirConditioningModeHeatDHW
		case "W":
			mode = cx34.AirConditioningModeOnlyDHW
		default:
			glog.Errorf("invalid mode: %v ", *setMode)
			return
		}

		fmt.Printf("Setting mode to %s\n", mode)
		cxClient.SetACMode(mode)
	}

	return
}

func parseTemperatureFlag(string) (units.Temperature, error) {
	suffix := (*setCoolingTemp)[len(*setCoolingTemp)-1:]
	floatVal, err := strconv.ParseFloat((*setCoolingTemp)[0:len(*setCoolingTemp)-1], 64)
	if err != nil{
		return units.Temperature(0), err
	}

	var temp units.Temperature
	switch suffix {
	case "c", "C":
		temp = units.FromCelsius(floatVal)
	case "f", "F":
		temp = units.FromFahrenheit(floatVal)
	default:
		return units.Temperature(0), errors.New("temperature suffix not recognized, use C or F")
	}

	return temp, nil
}

func printState(state *cx34.State) {
	cop, running := state.COP()
	runningStr := ""
	if running {
		runningStr = "running"
	} else {
		runningStr = "stopped"
	}

	fmt.Printf(
`Summary for CX34 unit %d:
  Mode: %s
  COP: %.2f (%s)
  Power: %.2f Watts
  Outdoor Temp: %.2f °F
  Hot Water Tank Temp: %.2f °F
  Cooling Target Temp: %.2f °F
  Heating Target Temp: %.2f °F
  Hot Water Target Temp: %.2f °F
  Inlet Temp: %.2f °F
  Outlet Temp: %.2f °F
  Pump Speed: %.2f l/s
  Useful Heat Rate: %s
`,
		*unitId,
		state.ACMode(),
		cop, runningStr,
		state.ApparentPower(),
		state.AmbientTemp().Fahrenheit(),
		state.DomesticHotWaterTankTemp().Fahrenheit(),
		state.ACCoolingTargetTemp().Fahrenheit(),
		state.ACHeatingTargetTemp().Fahrenheit(),
		state.DomesticHotWaterTargetTemp().Fahrenheit(),
		state.ACInletWaterTemp().Fahrenheit(),
		state.ACOutletWaterTemp().Fahrenheit(),
		state.FlowRate(),
		state.UsefulHeatRateExplained(),
	)
}
