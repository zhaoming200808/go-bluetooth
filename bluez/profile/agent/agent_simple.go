package agent

import (
	"fmt"

	"github.com/godbus/dbus"
	"github.com/muka/go-bluetooth/api"
	log "github.com/sirupsen/logrus"
)

const SimpleAgentPath = "/go_bluetooth/agent"
const SimpleAgentPinCode = "0000"
const SimpleAgentPassKey uint32 = 1024

// NewDefaultSimpleAgent return a SimpleAgent instance with default pincode and passcode
func NewDefaultSimpleAgent() *SimpleAgent {
	ag := &SimpleAgent{
		path:    SimpleAgentPath,
		passKey: SimpleAgentPassKey,
		pinCode: SimpleAgentPinCode,
	}
	return ag
}

// NewSimpleAgent return a SimpleAgent instance
func NewSimpleAgent() *SimpleAgent {
	ag := &SimpleAgent{
		path: SimpleAgentPath,
	}
	return ag
}

// SimpleAgent implement interface Agent1Client
type SimpleAgent struct {
	path    dbus.ObjectPath
	pinCode string
	passKey uint32
}

func (self *SimpleAgent) SetPassKey(passkey uint32) {
	self.passKey = passkey
}

func (self *SimpleAgent) SetPassCode(pinCode string) {
	self.pinCode = pinCode
}

func (self *SimpleAgent) Path() dbus.ObjectPath {
	return self.path
}

func (self *SimpleAgent) Interface() string {
	return Agent1Interface
}

func (self *SimpleAgent) Release() *dbus.Error {
	return nil
}

func (self *SimpleAgent) RequestPinCode(path dbus.ObjectPath) (string, *dbus.Error) {

	log.Debugf("SimpleAgent: RequestPinCode: %s", path)

	adapterID, err := api.ParseAdapterIDFromDevicePath(path)
	if err != nil {
		log.Warnf("SimpleAgent: Failed to load adapter %s", err)
		return "", &dbus.ErrMsgNoObject
	}

	err = SetTrusted(adapterID, path)
	if err != nil {
		return "", dbus.MakeFailedError(err)
	}

	return SimpleAgentPinCode, nil
}

func (self *SimpleAgent) DisplayPinCode(device dbus.ObjectPath, pincode string) *dbus.Error {
	log.Info(fmt.Sprintf("SimpleAgent: DisplayPinCode (%s, %s)", device, pincode))
	return nil
}

func (self *SimpleAgent) RequestPasskey(path dbus.ObjectPath) (uint32, *dbus.Error) {

	adapterID, err := api.ParseAdapterIDFromDevicePath(path)
	if err != nil {
		log.Warnf("SimpleAgent: Failed to load adapter %s", err)
		return 0, &dbus.ErrMsgNoObject
	}

	err = SetTrusted(adapterID, path)
	if err != nil {
		return 0, dbus.MakeFailedError(err)
	}

	return SimpleAgentPassKey, nil
}

func (self *SimpleAgent) DisplayPasskey(device dbus.ObjectPath, passkey uint32, entered uint16) *dbus.Error {
	log.Debugf("SimpleAgent: DisplayPasskey %s, %06d entered %d", device, passkey, entered)
	return nil
}

func (self *SimpleAgent) RequestConfirmation(path dbus.ObjectPath, passkey uint32) *dbus.Error {

	log.Debugf("SimpleAgent: RequestConfirmation (%s, %06d)", path, passkey)

	adapterID, err := api.ParseAdapterIDFromDevicePath(path)
	if err != nil {
		log.Warnf("SimpleAgent: Failed to load adapter %s", err)
		return &dbus.ErrMsgNoObject
	}

	err = SetTrusted(adapterID, path)
	if err != nil {
		return dbus.MakeFailedError(err)
	}

	return nil
}

func (self *SimpleAgent) RequestAuthorization(device dbus.ObjectPath) *dbus.Error {
	log.Debugf("SimpleAgent: RequestAuthorization (%s)", device)
	return nil
}

func (self *SimpleAgent) AuthorizeService(device dbus.ObjectPath, uuid string) *dbus.Error {
	log.Debugf("SimpleAgent: AuthorizeService (%s, %s)", device, uuid) // directly authrized
	return nil
}

func (self *SimpleAgent) Cancel() *dbus.Error {
	log.Debugf("SimpleAgent: Cancel")
	return nil
}
