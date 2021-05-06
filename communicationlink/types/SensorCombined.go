package types

import "unsafe"

/*
#cgo LDFLAGS: -L/opt/ros/foxy/lib -Wl,-rpath=/opt/ros/foxy/lib -lrcl -lrosidl_runtime_c -lrosidl_typesupport_cpp -lrosidl_typesupport_c -lstd_msgs__rosidl_generator_c -lpx4_msgs__rosidl_typesupport_c
#cgo CFLAGS: -I/opt/ros/foxy/include
#include <px4_msgs/msg/sensor_combined.h>
static inline const rosidl_message_type_support_t * ts_sc(){
	const rosidl_message_type_support_t * ts =rosidl_typesupport_c__get_message_type_support_handle__px4_msgs__msg__SensorCombined();
	return ts;
}
*/
import "C"

const (
	sensorCombinedRELATIVETIMESTAMPIINVALID int32 = 2147483647
	sensorCombinedCLIPPINGX                 uint8 = 1
	sensorCombinedCLIPPINGY                 uint8 = 2
	sensorCombinedCLIPPINGZ                 uint8 = 4
)

//SensorCombined ROS px4 message struct
type SensorCombined struct {
	Timestamp                      uint64
	GyroRad                        [3]float32
	GyroIntegralDt                 uint32
	AccelerometerTimestampRelative int32
	AccelerometerMS2               [3]float32
	AccelerometerIntegralDt        uint32
	AccelerometerClipping          uint8
}

//TypeSupport ROS msg typesupport
func (t *SensorCombined) TypeSupport() unsafe.Pointer {
	return unsafe.Pointer(C.ts_sc())
}
