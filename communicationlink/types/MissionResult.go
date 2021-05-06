package types

import "unsafe"

/*
#cgo LDFLAGS: -L/opt/ros/foxy/lib -Wl,-rpath=/opt/ros/foxy/lib -lrcl -lrosidl_runtime_c -lrosidl_typesupport_cpp -lrosidl_typesupport_c -lstd_msgs__rosidl_generator_c -lpx4_msgs__rosidl_typesupport_c
#cgo CFLAGS: -I/opt/ros/foxy/include
#include <px4_msgs/msg/mission_result.h>
static inline const rosidl_message_type_support_t * ts_sc(){
	const rosidl_message_type_support_t * ts =rosidl_typesupport_c__get_message_type_support_handle__px4_msgs__msg__MissionResult();
	return ts;
}
*/
import "C"

// uint8 MISSION_EXECUTION_MODE_NORMAL = 0	# Execute the mission according to the planned items
// uint8 MISSION_EXECUTION_MODE_REVERSE = 1	# Execute the mission in reverse order, ignoring commands and converting all waypoints to normal ones
// uint8 MISSION_EXECUTION_MODE_FAST_FORWARD = 2	# Execute the mission as fast as possible, for example converting loiter waypoints to normal ones

type MissionResult struct {
	Timestamp           uint64 // time since system start (microseconds)
	InstanceCount       uint32 // Instance count of this mission. Increments monotonically whenever the mission is modified
	SeqReached          int32  // Sequence of the mission item which has been reached, default -1
	SeqCurrent          uint16 // Sequence of the current mission item
	SeqTotal            uint16 // Total number of mission items
	Valid               bool   // true if mission is valid
	Warning             bool   // true if mission is valid, but has potentially problematic items leading to safety warnings
	Finished            bool   // true if mission has been completed
	Failure             bool   // true if the mission cannot continue or be completed for some reason
	StayInFailsafe      bool   // true if the commander should not switch out of the failsafe mode
	FlightTermination   bool   // true if the navigator demands a flight termination from the commander app
	ItemDoJumpChanged   bool   // true if the number of do jumps remaining has changed
	ItemChangedIndex    uint16 // indicate which item has changed
	ItemDoJumpRemaining uint16 // set to the number of do jumps remaining for that item
	ExecutionMode       uint8  // indicates the mode in which the mission is executed
}

func (t *MissionResult) TypeSupport() unsafe.Pointer {
	return unsafe.Pointer(C.ts_sc())
}
