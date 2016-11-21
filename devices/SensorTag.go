package devices

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"

	"github.com/godbus/dbus"
	"github.com/muka/go-bluetooth/api"
	"github.com/muka/go-bluetooth/bluez"
	"github.com/muka/go-bluetooth/bluez/profile"
	"github.com/op/go-logging"
	"github.com/tj/go-debug"
)

var logger = logging.MustGetLogger("main")
var dbgtag = debug.Debug("bluez:sensortag")

var notifications chan dbus.Signal

var sensorTagUUIDs = map[string]string{

	"TemperatureData":   "AA01",
	"TemperatureConfig": "AA02",
	"TemperaturePeriod": "AA03",

	"AccelerometerData":   "AA11",
	"AccelerometerConfig": "AA12",
	"AccelerometerPeriod": "AA13",

	"HumidityData":   "AA21",
	"HumidityConfig": "AA22",
	"HumidityPeriod": "AA23",

	"MagnetometerData":   "AA31",
	"MagnetometerConfig": "AA32",
	"MagnetometerPeriod": "AA33",

	"BarometerData":        "AA41",
	"BarometerConfig":      "AA42",
	"BarometerPeriod":      "AA44",
	"BarometerCalibration": "AA43",

	"GyroscopeData":   "AA51",
	"GyroscopeConfig": "AA52",
	"GyroscopePeriod": "AA53",

	"TestData":   "AA61",
	"TestConfig": "AA62",

	"ConnectionParams":        "CCC1",
	"ConnectionReqConnParams": "CCC2",
	"ConnectionDisconnReq":    "CCC3",

	"OADImageIdentify": "FFC1",
	"OADImageBlock":    "FFC2",
}

//Period =[Input*10]ms,(lowerlimit 300 ms, max 2500ms),default 1000 ms
const (
	TemperaturePeriodHigh   = 0x32  // 500 ms,
	TemperaturePeriodMedium = 0x64  // 1000 ms,
	TemperaturePeriodLow    = 0x128 // 2000 ms,
)

func getUUID(name string) string {
	if sensorTagUUIDs[name] == "" {
		panic("Not found " + name)
	}
	return "F000" + sensorTagUUIDs[name] + "-0451-4000-B000-000000000000"
}

func newTemperatureSensor(tag *SensorTag) (TemperatureSensor, error) {

	dev := tag.Device

	retry := 3
	tries := 0
	var loadChars func() (TemperatureSensor, error)

	loadChars = func() (TemperatureSensor, error) {

		dbgtag("Load temp cfg")
		cfg, err := dev.GetCharByUUID(getUUID("TemperatureConfig"))
		if err != nil {
			return TemperatureSensor{}, err
		}

		if cfg == nil {

			if tries == retry {
				return TemperatureSensor{}, errors.New("Cannot find cfg characteristic")
			}

			tries++
			time.Sleep(time.Second * time.Duration(5*tries))
			dbgtag("Char not found, try to reload")

			return loadChars()
		}

		dbgtag("Load temp data")
		data, err := dev.GetCharByUUID(getUUID("TemperatureData"))
		if err != nil {
			return TemperatureSensor{}, err
		}

		dbgtag("Load temp period")
		period, err := dev.GetCharByUUID(getUUID("TemperaturePeriod"))
		if err != nil {
			return TemperatureSensor{}, err
		}

		return TemperatureSensor{tag, cfg, data, period}, err
	}

	return loadChars()
}

//Sensor generic sensor interface
type Sensor interface {
	GetName() string
	IsEnabled() (bool, error)
	Enable() error
	Disable() error
}

//TemperatureSensor the temperature sensor
type TemperatureSensor struct {
	tag    *SensorTag
	cfg    *profile.GattCharacteristic1
	data   *profile.GattCharacteristic1
	period *profile.GattCharacteristic1
}

// GetName return the sensor name
func (s TemperatureSensor) GetName() string {
	return "temperature"
}

//Enable measurements
func (s *TemperatureSensor) Enable() error {
	enabled, err := s.IsEnabled()
	if err != nil {
		return err
	}
	if enabled {
		return nil
	}
	options := make(map[string]dbus.Variant)
	err = s.cfg.WriteValue([]byte{1}, options)
	if err != nil {
		return err
	}
	return nil
}

//Disable measurements
func (s *TemperatureSensor) Disable() error {
	enabled, err := s.IsEnabled()
	if err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	options := make(map[string]dbus.Variant)
	err = s.cfg.WriteValue([]byte{0}, options)
	if err != nil {
		return err
	}
	return nil
}

//IsEnabled check if measurements are enabled
func (s *TemperatureSensor) IsEnabled() (bool, error) {
	options := make(map[string]dbus.Variant)

	val, err := s.cfg.ReadValue(options)
	if err != nil {
		return false, err
	}

	buf := bytes.NewBuffer(val)
	enabled, err := binary.ReadVarint(buf)
	if err != nil {
		return false, err
	}

	return (enabled == 1), nil
}

// Port from http://processors.wiki.ti.com/index.php/SensorTag_User_Guide#IR_Temperature_Sensor
var calcTmpLocal = func(raw uint16) float64 {
	return float64(raw) / 128.0
}

/* Conversion algorithm for target temperature */
// var calcTmpTarget = func(raw uint16) float64 {
//
// 	//-- calculate target temperature [°C] -
// 	Vobj2 := float64(raw) * 0.00000015625
// 	Tdie2 := calcTmpLocal(raw) + 273.15
//
// 	const S0 = 6.4E-14 // Calibration factor
// 	const a1 = 1.75E-3
// 	const a2 = -1.678E-5
// 	const b0 = -2.94E-5
// 	const b1 = -5.7E-7
// 	const b2 = 4.63E-9
// 	const c2 = 13.4
// 	const Tref = 298.15
//
// 	S := S0 * (1 + a1*(Tdie2-Tref) + a2*math.Pow((Tdie2-Tref), 2))
// 	Vos := b0 + b1*(Tdie2-Tref) + b2*math.Pow((Tdie2-Tref), 2)
// 	fObj := (Vobj2 - Vos) + c2*math.Pow((Vobj2-Vos), 2)
//
// 	tObj := math.Pow(math.Pow(Tdie2, 4)+(fObj/S), .25)
// 	tObj = (tObj - 273.15)
//
// 	return tObj
// }

//Read value from the sensor
func (s *TemperatureSensor) Read() (float64, error) {

	dbgtag("Reading temperature sensor")

	err := s.Enable()
	if err != nil {
		return 0, err
	}

	options := make(map[string]dbus.Variant)
	b, err := s.data.ReadValue(options)

	dbgtag("Read data: %v", b)

	if err != nil {
		return 0, err
	}

	// die := binary.LittleEndian.Uint16(b[0:2])
	amb := binary.LittleEndian.Uint16(b[2:])

	// dieValue := calcTmpTarget(uint16(die))
	ambientValue := calcTmpLocal(uint16(amb))

	return ambientValue, err
}

//StartNotify enable notifications
// func (s *TemperatureSensor) StartNotify(fn func(temperature float64)) error {
func (s *TemperatureSensor) StartNotify() error {

	dbgtag("Enabling notifications")

	err := s.Enable()
	if err != nil {
		return err
	}

	// notifications, err := s.data.Register()
	_, err = s.data.Register()
	if err != nil {
		return err
	}

	// go func() {
	// 	for event := range notifications {
	//
	// 		if event == nil {
	// 			return
	// 		}
	//
	// 		dbgtag("Got update %v", event)
	//
	// 		switch event.Body[0].(type) {
	// 		case dbus.ObjectPath:
	// 			dbgtag("Received body type does not match: [0] %v -> [1] %v", event.Body[0], event.Body[1])
	// 			continue
	// 		case string:
	// 			// dbgtag("body type match")
	// 		}
	//
	// 		if event.Body[0] != bluez.GattCharacteristic1Interface {
	// 			// dbgtag("Skip interface %s", event.Body[0])
	// 			continue
	// 		}
	//
	// 		props := event.Body[1].(map[string]dbus.Variant)
	//
	// 		if _, ok := props["Value"]; !ok {
	// 			// dbgtag("Cannot read Value property %v", props)
	// 			continue
	// 		}
	//
	// 		b := props["Value"].Value().([]byte)
	//
	// 		dbgtag("Read data: %v", b)
	//
	// 		amb := binary.LittleEndian.Uint16(b[2:])
	// 		ambientValue := calcTmpLocal(uint16(amb))
	//
	// 		// die := binary.LittleEndian.Uint16(b[0:2])
	// 		// dieValue := calcTmpTarget(uint16(die))
	//
	// 		fn(ambientValue)
	// 	}
	// }()

	return s.data.StartNotify()
}

//StopNotify disable notifications
func (s *TemperatureSensor) StopNotify() error {

	dbgtag("Disabling notifications")

	err := s.Enable()
	if err != nil {
		return err
	}

	if notifications != nil {
		close(notifications)
	}

	return s.data.StopNotify()
}

// NewSensorTag creates a new sensortag instance
func NewSensorTag(d *api.Device) (*SensorTag, error) {

	var connect = func(dev *api.Device) error {
		if !dev.IsConnected() {
			err := dev.Connect()
			if err != nil {
				return err
			}
		}
		return nil
	}

	d.On("changed", func(ev api.Event) {

		changed := ev.GetData().(api.PropertyChangedEvent)
		dbgtag("Property change %v", changed)

		if changed.Field == "Value" && changed.Iface == bluez.GattCharacteristic1Interface {

			b := changed.Value.([]byte)
			// die := binary.LittleEndian.Uint16(b[0:2])
			amb := binary.LittleEndian.Uint16(b[2:])
			// dieValue := calcTmpTarget(uint16(die))
			ambientValue := calcTmpLocal(uint16(amb))

			dataEvent := api.DataEvent{d, "temperature", ambientValue, "C"}
			d.Emit("data", dataEvent)
		}
		if changed.Field == "Connected" {
			conn := changed.Value.(bool)
			if !conn {
				dbgtag("Reconnecting to device %s", d.Properties.Address)
				err := connect(d)
				if err != nil {
					logger.Warningf("Reconnection failed! %s", err)
				}
			}
		}
	})

	connect(d)

	s := new(SensorTag)
	s.Device = d

	temp, err := newTemperatureSensor(s)
	if err != nil {
		return nil, err
	}
	s.Temperature = temp

	return s, nil
}

//SensorTag a SensorTag object representation
type SensorTag struct {
	*api.Device
	Temperature TemperatureSensor
}