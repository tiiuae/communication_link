package types

import "unsafe"

/*
#cgo LDFLAGS: -L/opt/ros/foxy/lib -Wl,-rpath=/opt/ros/foxy/lib -lrcl -lrosidl_runtime_c -lrosidl_typesupport_cpp -lrosidl_typesupport_c -lstd_msgs__rosidl_generator_c -lpx4_msgs__rosidl_typesupport_c
#cgo CFLAGS: -I/opt/ros/foxy/include
#include <px4_msgs/msg/battery_status.h>
static inline const rosidl_message_type_support_t * ts_bat(){
	const rosidl_message_type_support_t * ts =rosidl_typesupport_c__get_message_type_support_handle__px4_msgs__msg__BatteryStatus();
	return ts;
}
*/
import "C"

const (
	BatteryStatus_BATTERY_SOURCE_POWER_MODULE uint8 = 0
	BatteryStatus_BATTERY_SOURCE_EXTERNAL     uint8 = 1
	BatteryStatus_BATTERY_SOURCE_ESCS         uint8 = 2
	BatteryStatus_BATTERY_WARNING_NONE        uint8 = 0
	BatteryStatus_BATTERY_WARNING_LOW         uint8 = 1
	BatteryStatus_BATTERY_WARNING_CRITICAL    uint8 = 2
	BatteryStatus_BATTERY_WARNING_EMERGENCY   uint8 = 3
	BatteryStatus_BATTERY_WARNING_FAILED      uint8 = 4
	BatteryStatus_MAX_INSTANCES               uint8 = 4
)

type BatteryStatus struct {
	Timestamp           uint64
	VoltageV            float32
	VoltageFilteredV    float32
	CurrentA            float32
	CurrentFilteredA    float32
	AverageCurrentA     float32
	DischargedMah       float32
	Remaining           float32
	Scale               float32
	Temperature         float32
	CellCount           int32
	Connected           bool
	Source              uint8
	Priority            uint8
	Capacity            uint16
	CycleCount          uint16
	RunTimeToEmpty      uint16
	AverageTimeToEmpty  uint16
	SerialNumber        uint16
	ManufactureDate     uint16
	StateOfHealth       uint16
	MaxError            uint16
	Id                  uint8
	InterfaceError      uint16
	VoltageCellV        [10]float32
	MaxCellVoltageDelta float32
	IsPoweringOff       bool
	Warning             uint8
}

//TypeSupport ROS msg typesupport
func (t *BatteryStatus) TypeSupport() unsafe.Pointer {
	return unsafe.Pointer(C.ts_bat())
}
