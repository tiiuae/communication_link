package types

import "unsafe"

/*
#cgo LDFLAGS: -L/opt/ros/foxy/lib -Wl,-rpath=/opt/ros/foxy/lib -lrcl -lrosidl_runtime_c -lrosidl_typesupport_cpp -lrosidl_typesupport_c -lstd_msgs__rosidl_generator_c -lpx4_msgs__rosidl_typesupport_c
#cgo CFLAGS: -I/opt/ros/foxy/include
#include <px4_msgs/msg/vehicle_global_position.h>
static inline const rosidl_message_type_support_t * ts_vgb(){
	const rosidl_message_type_support_t * ts =rosidl_typesupport_c__get_message_type_support_handle__px4_msgs__msg__VehicleGlobalPosition();
	return ts;
}
*/
import "C"

//VehicleGlobalPosition ROS px4 message struct
type VehicleGlobalPosition struct {
	Timestamp          uint64
	Timestamp_sample   uint64
	Lat                float64
	Lon                float64
	Alt                float32
	AltEllipsoid       float32
	DeltaAlt           float32
	LatLonResetCounter uint8
	AltResetCounter    uint8
	Eph                float32
	Epv                float32
	TerrainAlt         float32
	TerrainAltValid    bool
	DeadReckoning      bool
}

//TypeSupport ROS msg typesupport
func (t *VehicleGlobalPosition) TypeSupport() unsafe.Pointer {
	return unsafe.Pointer(C.ts_vgb())
}
