package types

import "unsafe"

/*
#cgo LDFLAGS: -L/opt/ros/foxy/lib -Wl,-rpath=/opt/ros/foxy/lib -lrcl -lrosidl_runtime_c -lrosidl_typesupport_cpp -lrosidl_typesupport_c -lstd_msgs__rosidl_generator_c -lpx4_msgs__rosidl_typesupport_c
#cgo CFLAGS: -I/opt/ros/foxy/include
#include <px4_msgs/msg/vehicle_local_position.h>
static inline const rosidl_message_type_support_t * ts_vlp(){
	const rosidl_message_type_support_t * ts =rosidl_typesupport_c__get_message_type_support_handle__px4_msgs__msg__VehicleLocalPosition();
	return ts;
}
*/
import "C"

type VehicleLocalPosition struct {
	Timestamp           uint64
	TimestampSample     uint64
	XyValid             bool
	ZValid              bool
	VXyValid            bool
	VZValid             bool
	X                   float32
	Y                   float32
	Z                   float32
	DeltaXy             [2]float32
	XyResetCounter      uint8
	DeltaZ              float32
	ZResetCounter       uint8
	Vx                  float32
	Vy                  float32
	Vz                  float32
	ZDeriv              float32
	DeltaVxy            [2]float32
	VxyResetCounter     uint8
	DeltaVz             float32
	VzResetCounter      uint8
	Ax                  float32
	Ay                  float32
	Az                  float32
	Heading             float32
	DeltaHeading        float32
	HeadingResetCounter uint8
	XyGlobal            bool
	ZGlobal             bool
	RefTimestamp        uint64
	RefLat              float64
	RefLon              float64
	RefAlt              float32
	DistBottom          float32
	DistBottomValid     bool
        DistBottomSensorBitfield uint8
	Eph                 float32
	Epv                 float32
	Evh                 float32
	Evv                 float32
	VxyMax              float32
	VzMax               float32
	HaglMin             float32
	HaglMax             float32
}

//TypeSupport ROS msg typesupport
func (t *VehicleLocalPosition) TypeSupport() unsafe.Pointer {
	return unsafe.Pointer(C.ts_vlp())
}
